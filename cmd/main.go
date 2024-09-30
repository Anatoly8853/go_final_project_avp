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
	r.GET("/login.html", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	// Маршрут для аутентификации
	r.POST("/api/signin", newHandler.SignIn)

	r.GET("/", handler.Index)
	r.GET("/index.html", handler.Index)
	r.GET("api/nextdate", newHandler.GetNextDate)
	// Применяем middleware для защищённых маршрутов
	authRoutes := r.Group("/api")
	authRoutes.Use(newHandler.AuthMiddleware())
	{
		authRoutes.GET("/tasks", newHandler.GetTasks)
		authRoutes.GET("/task", newHandler.GetTasksId)
		authRoutes.PUT("/task", newHandler.UpdateTask)
		authRoutes.POST("/task", newHandler.CreateTask)
		authRoutes.DELETE("/task", newHandler.DeleteTask)
		authRoutes.POST("/task/done", newHandler.DoneTask)
	}

	if err = r.Run(":" + cfg.Port); err != nil {
		app.Log.Fatalf("Не удалось запустить сервер: %v", err)
	}
}
