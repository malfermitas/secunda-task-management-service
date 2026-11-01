package mysql

import (
	"errors"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

const mysqlDuplicateEntry = 1062

func isDuplicateKey(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntry
}
