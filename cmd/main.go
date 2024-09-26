package main

import (
	"go_final_project_avp/config"
	"go_final_project_avp/handler"
	"go_final_project_avp/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Настраиваем логгер
	app := config.SetupApplication()
	cfg := config.LoadConfig(app)

	db, err := repository.NewOpenDB(cfg)
	if err != nil {
		app.Log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	/*
		defer func(db *sqlx.DB) {
			_ = db.Close()
		}(db)
	*/

	repo := repository.NewRepository(db, app)

	if err = repo.RunMigrations(cfg); err != nil {
		app.Log.Fatalf("Не удалось выполнить миграцию: %v", err)
	}

	newHandler := handler.NewHandler(cfg, repo, app)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.StaticFile("/favicon.ico", "./web/favicon.ico")
	r.LoadHTMLGlob("web/*.html")
	// Маршрут для логин страницы
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	// Маршрут для аутентификации
	r.POST("/api/signin", newHandler.SignIn)
	// Применяем middleware для защищённых маршрутов
	authRoutes := r.Group("/")
	authRoutes.Use(newHandler.AuthMiddleware())
	{
		authRoutes.GET("/", handler.Index)
		authRoutes.GET("/api/task", newHandler.GetTasksId)
		authRoutes.PUT("/api/task", newHandler.UpdateTask)
		authRoutes.GET("/api/tasks", newHandler.GetTasks)
		authRoutes.GET("api/nextdate", newHandler.GetNextDate)
		authRoutes.POST("/api/task", newHandler.CreateTask)
		authRoutes.POST("/api/task/done", newHandler.DoneTask)
		authRoutes.DELETE("/api/task", newHandler.DeleteTask)
	}
	if err = r.Run(":" + cfg.Port); err != nil {
		app.Log.Fatalf("Не удалось запустить сервер: %v", err)
	}
}
