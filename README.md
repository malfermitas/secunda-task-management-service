# Secunda Task Management Service

REST API для управления задачами внутри команд с ролевой моделью доступа, историей изменений, кешированием в Redis и SQL-аналитикой.

## Стек

- Go 1.26
- MySQL 8.4
- Redis 7
- Docker Compose
- JWT-аутентификация
- Swagger/OpenAPI (`swaggo/swag`)
- Prometheus, OpenTelemetry, Jaeger (опциональный стек наблюдаемости)

## Требования

- Go 1.26+
- Docker и Docker Compose
- [Task](https://taskfile.dev/installation/) как раннер команд

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

На Windows также подойдут `winget install Task.Task` или `choco install go-task`. Посмотреть все задачи можно командой `task --list`.

## Быстрый старт

1. Скопируйте пример окружения и задайте JWT-секрет длиной не менее 32 символов:

```bash
cp .env.example .env
```

2. Запустите приложение, MySQL, Redis и миграции:

```bash
task up
```

3. Откройте:

- API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Метрики Prometheus: `http://localhost:9090/metrics`
- Jaeger UI (опциональный стек): `http://localhost:16686`
- Grafana (опциональный стек): `http://localhost:3000`

4. Смотрите логи или остановите стек:

```bash
task logs
task down
```

## Конфигурация

Конфигурация загружается Viper со следующим приоритетом:

```text
defaults < config file < environment variables
```

`CONFIG_FILE` указывает на YAML/JSON/TOML-файл; иначе приложение ищет `config.*` в `.` и `./config`. Само Go-приложение не читает `.env`; это делает Docker Compose.

| Секция | Назначение |
| --- | --- |
| `app` | Имя сервиса, окружение, версия, настройки HTTP-сервера и graceful shutdown |
| `db` | Подключение к MySQL и настройки пула |
| `redis` | Адрес Redis, пул и таймауты |
| `jwt` | JWT-секрет и TTL |
| `cache` | TTL кеша списка задач |
| `log` | Уровень и формат zap (`json` или `text`) |
| `metrics` | Отдельный listener для Prometheus (по умолчанию `:9090`) |
| `tracing` | Трейсинг через OTLP gRPC, по умолчанию выключен |

Полный список — в `config.example.yaml`; имена переменных окружения — в `.env.example`.

## Локальный запуск

```bash
export JWT_SECRET="0123456789abcdef0123456789abcdef"
export CONFIG_FILE=config.example.yaml
go run ./cmd/api
```

MySQL и Redis должны быть доступны по адресам из выбранной конфигурации.

## Миграции

Миграции лежат в `migrations/` и применяются сервисом `migrate` в Compose:

```bash
task migrate-up
```

Раннер миграций использует `multiStatements=true`. Приложение само не применяет миграции при старте.

## API

Базовый путь: `/api/v1`.

| Метод | Путь | Доступ | Описание |
| --- | --- | --- | --- |
| `POST` | `/register` | Публичный | Регистрация пользователя |
| `POST` | `/login` | Публичный | Аутентификация и выдача JWT |
| `POST` | `/teams` | Аутентифицированный | Создание команды; текущий пользователь становится owner |
| `GET` | `/teams` | Аутентифицированный | Список команд, где пользователь состоит |
| `POST` | `/teams/{id}/invite` | Owner/admin | Приглашение пользователя с ролью admin или member |
| `POST` | `/tasks` | Участник команды | Создание задачи |
| `GET` | `/tasks` | Участник команды | Список задач с фильтрами и пагинацией |
| `PUT` | `/tasks/{id}` | Owner/admin, создатель или исполнитель | Обновление задачи с проверкой версии |
| `GET` | `/tasks/{id}/history` | Участник команды | История изменений задачи |
| `POST` | `/tasks/{id}/comments` | Участник команды | Добавление комментария |
| `GET` | `/tasks/{id}/comments` | Участник команды | Список комментариев |
| `GET` | `/teams/{team_id}/stats` | Owner/admin | SQL-отчёт аналитики |

Аутентифицированные запросы используют:

```http
Authorization: Bearer <jwt>
```

### Правила доступа

- Пользователь видит только команды, где он участник или создатель.
- Для доступа к задачам нужно состоять в команде задачи.
- История и комментарии доступны только участникам команды.
- Аналитика доступна только owner/admin команды и всегда фильтруется по `team_id`.
- Исполнитель должен быть участником той же команды.
- Owner/admin могут редактировать любую задачу команды.
- Создатель задачи может редактировать свою задачу.
- Исполнитель может менять только статус задачи.
- Роль `owner` нельзя выдать через приглашение.
- Каждое обновление задачи записывает старые и новые значения, автора и время в `task_history`.

### Фильтры списка задач

```http
GET /api/v1/tasks?team_id=1&status=todo&assignee_id=5&limit=20&offset=0
```

- `team_id` обязателен.
- `status` — одно из значений `todo`, `in_progress`, `done`.
- `limit` по умолчанию 20, максимум 100.
- `offset` по умолчанию 0.
- Результат кешируется в Redis на 5 минут для команды и нормализованного набора фильтров.
- Создание и обновление задачи инвалидируют поколение кеша команды после коммита.

### Надёжность

- Создание команды и членство owner записываются в одной транзакции.
- Создание задачи и начальная запись истории выполняются в одной транзакции.
- Обновление задачи и запись истории выполняются в одной транзакции.
- `SELECT ... FOR UPDATE` вместе с версией задачи предотвращает потерю обновлений; устаревшая версия возвращает `409`.
- Ошибки Redis не ломают запросы (fail-open); изменения задач инвалидируют поколение кеша команды после коммита.

## Примеры

Регистрация и получение токена:

```bash
curl -s -X POST http://localhost:8080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","password":"password123","name":"Jane Doe"}'

TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","password":"password123"}' \
  | jq -r .token)
```

Создание команды:

```bash
curl -s -X POST http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Platform Team"}'
```

Создание задачи:

```bash
curl -s -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"team_id":1,"title":"Fix login","description":"SSO login is broken","assignee_id":2}'
```

Список задач:

```bash
curl -s "http://localhost:8080/api/v1/tasks?team_id=1&status=todo&limit=20&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

Обновление задачи с оптимистической проверкой версии:

```bash
curl -s -X PUT http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress","version":0}'
```

Добавление и список комментариев:

```bash
curl -s -X POST http://localhost:8080/api/v1/tasks/1/comments \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"content":"I reproduced this on staging."}'

curl -s http://localhost:8080/api/v1/tasks/1/comments \
  -H "Authorization: Bearer $TOKEN"
```

История и аналитика:

```bash
curl -s http://localhost:8080/api/v1/tasks/1/history \
  -H "Authorization: Bearer $TOKEN"

curl -s http://localhost:8080/api/v1/teams/1/stats \
  -H "Authorization: Bearer $TOKEN"
```

## Swagger / OpenAPI

Спецификация генерируется в `docs/`:

```bash
task swagger
```

Команда запускает `swag fmt` и перегенерирует `docs/docs.go`, `docs/swagger.json` и `docs/swagger.yaml`. UI доступен по адресу `/swagger/index.html`; сырая спецификация — `/swagger/doc.json`.

JWT-схема безопасности называется `BearerAuth`. Публичные эндпоинты — `/register` и `/login`; остальные документированные эндпоинты требуют bearer-токен. В локальном Compose Swagger UI открыт публично; в production закройте или отключите `/swagger`.

## Postman

Импортируйте:

```text
postman/secunda-task-management.postman_collection.json
```

Коллекция содержит папки Authentication, Teams, Tasks и Analytics. У каждого запроса есть описание с правами, полями тела и ожидаемыми статусами.

Переменные коллекции:

- `baseUrl` по умолчанию `http://localhost:8080/api/v1`
- `token` заполняется тест-скриптом запроса Login
- `teamId`, `taskId`, `taskVersion`, `commentId` заполняются тест-скриптами запросов
- `inviteeId` по умолчанию `2`; замените на существующий ID пользователя

## Наблюдаемость

Запуск опционального стека OTel Collector, Jaeger, Prometheus и Grafana:

```bash
task observability-up
```

Это включает трейсинг со 100% сэмплированием для локального запуска. Остановка:

```bash
task observability-down
```

- Структурированные JSON-логи пишутся в stdout через zap и содержат `request_id`, `trace_id`, `span_id` и `user_id`, когда они доступны.
- HTTP-метрики и трейсы эмитятся через `otelgin`.
- Сигналы MySQL эмитятся через `otelsql`; Redis — через `redisotel`.
- Счётчики hit/miss и инвалидации кеша эмитятся приложением.
- Правила алертов Prometheus лежат в `deploy/prometheus-alerts.yml`.

## Тесты

```bash
go build ./...
go vet ./...
go test ./...
task test-integration
```

Интеграционные тесты используют build-тег `integration` и требуют внешних MySQL/Redis.
