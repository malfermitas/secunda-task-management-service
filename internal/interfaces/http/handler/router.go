package handler

import (
	"net/http"

	analyticsapp "secunda-task-management-service/internal/analytics/app"
	"secunda-task-management-service/internal/interfaces/http/middleware"
	taskapp "secunda-task-management-service/internal/task/app"
	teamapp "secunda-task-management-service/internal/team/app"
	userapp "secunda-task-management-service/internal/user/app"

	"github.com/gin-gonic/gin"
)

// Dependencies contains the application services required by the HTTP layer.
type Dependencies struct {
	Auth        userapp.AuthService
	Teams       teamapp.Service
	Tasks       taskapp.Service
	Stats       analyticsapp.Service
	TokenParser middleware.TokenParser
}

// Router wraps the gin engine and implements http.Handler.
type Router struct {
	engine *gin.Engine
}

// NewRouter builds the HTTP router with all API routes registered.
func NewRouter(deps Dependencies) *Router {
	engine := gin.New()
	engine.Use(gin.Recovery())

	authHandler := &AuthHandler{svc: deps.Auth}
	teamsHandler := &TeamsHandler{teams: deps.Teams, stats: deps.Stats}
	tasksHandler := &TasksHandler{svc: deps.Tasks}

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := engine.Group("/api/v1")
	api.POST("/register", authHandler.Register)
	api.POST("/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.Auth(deps.TokenParser))
	{
		protected.POST("/teams", teamsHandler.Create)
		protected.GET("/teams", teamsHandler.List)
		protected.POST("/teams/:id/invite", teamsHandler.Invite)
		protected.GET("/teams/:team_id/stats", teamsHandler.Stats)

		protected.POST("/tasks", tasksHandler.Create)
		protected.GET("/tasks", tasksHandler.List)
		protected.PUT("/tasks/:id", tasksHandler.Update)
		protected.GET("/tasks/:id/history", tasksHandler.History)
		protected.GET("/tasks/:id/comments", tasksHandler.ListComments)
		protected.POST("/tasks/:id/comments", tasksHandler.AddComment)
	}
	return &Router{engine: engine}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.engine.ServeHTTP(w, req)
}
