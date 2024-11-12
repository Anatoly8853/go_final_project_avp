package handler

import (
	slogavp "github.com/Anatoly8853/slog-avp/v2"
	"go_final_project_avp/internal/config"
	"go_final_project_avp/internal/repository"
	"go_final_project_avp/internal/tasks"

	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	config config.Config
	app    *slogavp.Application
	repo   *repository.Repository
}

func NewHandler(config config.Config, repo *repository.Repository, app *slogavp.Application) *Handler {
	return &Handler{config: config, repo: repo, app: app}
}

// Index главная страница.
func Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

// GetTasks данные главной страницы.
func (h *Handler) GetTasks(c *gin.Context) {
	search := c.Query("search")

	repoTasks, err := h.repo.GetTasks()
	if err != nil {
		h.app.Log.Debugf("GetTasks repoTasks: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "ошибка вывода данных"})
		return
	}

	// Убираем все пробелы с начала и конца строки
	trimmed := strings.TrimSpace(search)
	if len(trimmed) > 0 {
		repoTasks, err = h.repo.GetSearch(search)
		if err != nil {
			h.app.Log.Debugf("GetTasks repoTasks: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "ошибка поиска"})
			return
		}
	}

	// Если список задач пустой, возвращаем пустой слайс
	if repoTasks == nil {
		repoTasks = []tasks.Task{}
	}

	c.JSON(http.StatusOK, gin.H{"tasks": repoTasks})
}

// GetNextDate обработчик для маршрута /api/nextdate правила повторения.
func (h *Handler) GetNextDate(c *gin.Context) {
	// Чтение параметров запроса
	nowStr := c.Query("now")
	dateStr := c.Query("date")
	repeat := c.Query("repeat")

	// Проверка, что все параметры присутствуют
	if nowStr == "" || dateStr == "" || repeat == "" {
		h.app.Log.Debug("GetNextDate параметры 'now', 'date' и 'repeat' обязательны")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Параметры 'now', 'date' и 'repeat' обязательны"})
		return
	}

	// Парсинг времени для параметра now
	now, err := time.Parse(tasks.TimeFormat, nowStr)
	if err != nil {
		h.app.Log.Debugf("GetNextDate Некорректная дата 'now', ожидается формат 20060102: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректная дата 'now', ожидается формат 20060102"})
		return
	}
	// Сбрасываем время
	nowDate := tasks.TruncateToDate(now)

	// Вызов функции NextDate для вычисления следующей даты
	nextDate, err := tasks.NextDate(nowDate, dateStr, repeat)
	if err != nil {
		h.app.Log.Debugf("GetNextDate tasks.NextDate правило повторения указано в неправильном формате: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "правило повторения указано в неправильном формате"})
		return
	}

	// Установка заголовка и отправка строки без кавычек
	c.Writer.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	c.Writer.WriteHeader(http.StatusOK)
	_, err = c.Writer.Write([]byte(nextDate))
	if err != nil {
		h.app.Log.Debug("GetNextDate Ошибка при отправке ответа")
	}
}

// CreateTask добавляем задачи.
func (h *Handler) CreateTask(c *gin.Context) {
	var newTask *tasks.Task

	// Парсинг запроса
	if err := c.ShouldBindJSON(&newTask); err != nil {
		h.app.Log.Debugf("CreateTask ShouldBindJSON неверные данные: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверные данные"})
		return
	}

	// Валидация даты
	if err := tasks.ValidateAndSetDate(newTask, time.Now()); err != nil {
		h.app.Log.Debugf("CreateTask дата представлена в формате, отличном от 20060102: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "дата представлена в формате, отличном от 20060102"})
		return
	}

	id, err := h.repo.CreateTask(newTask)
	if err != nil {
		h.app.Log.Debugf("CreateTask repo.CreateTask ошибка добавления в бд: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не указан заголовок задачи"})
	}

	// Ответ в формате JSON
	c.JSON(http.StatusOK, gin.H{"id": strconv.Itoa(int(id))})

}

// GetTasksId данные главной страницы.
func (h *Handler) GetTasksId(c *gin.Context) {

	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "идентификатор задачи обязателен"})
		return
	}

	repoTasks, err := h.repo.GetTasksId(id)
	if err != nil {
		h.app.Log.Debugf("GetTasks repoTasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "нет задачи с таким id"})
		return
	}

	c.JSON(http.StatusOK, repoTasks)
}

// UpdateTask обновляем данные после изменения.
func (h *Handler) UpdateTask(c *gin.Context) {
	var newTask *tasks.Task

	err := c.BindJSON(&newTask)
	if err != nil {
		h.app.Log.Debugf("UpdateTask ошибка парсинга: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "ошибка парсинга"})
		return
	}

	// Валидация даты
	if _, err = time.Parse(tasks.TimeFormat, newTask.Date); err != nil {
		h.app.Log.Debugf("UpdateTask дата представлена в формате, отличном от 20060102: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "дата представлена в формате, отличном от 20060102"})
		return
	}
	// Сбрасываем время
	nowDate := tasks.TruncateToDate(time.Now())

	// Вызов функции NextDate для вычисления следующей даты
	_, err = tasks.NextDate(nowDate, newTask.Date, newTask.Repeat)
	if err != nil {
		h.app.Log.Debugf("UpdateTask tasks.NextDate правило повторения указано в неправильном формате: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "правило повторения указано в неправильном формате"})
		return
	}

	err = h.repo.UpdateTask(newTask)
	if err != nil {
		h.app.Log.Debugf("UpdateTask repo.UpdateTask ошибка добавления в бд: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "задача не найдена"})
		return
	}

	// Ответ в формате JSON
	c.JSON(http.StatusOK, gin.H{})
}

// DoneTask обновляем данные задачи.
func (h *Handler) DoneTask(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		h.app.Log.Debug("DoneTask идентификатор задачи обязателен:")
		c.JSON(http.StatusBadRequest, gin.H{"error": "идентификатор задачи обязателен"})
		return
	}

	newTask, err := h.repo.GetTasksId(id)
	if err != nil {
		h.app.Log.Debugf("DoneTask repo.GetTasksId: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Задача не найдена"})
		return
	}

	if newTask.Repeat == "" {
		err = h.repo.DeleteTask(newTask.Id)
		if err != nil {
			h.app.Log.Debugf("DoneTask repoTasks: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Задача не найдена"})
			return
		}
		// Ответ в формате JSON
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	// Сбрасываем время
	nowDate := tasks.TruncateToDate(time.Now())

	// Вызов функции NextDate для вычисления следующей даты
	nextDate, err := tasks.NextDate(nowDate, newTask.Date, newTask.Repeat)
	if err != nil {
		h.app.Log.Debugf("UpdateTask tasks.NextDate правило повторения указано в неправильном формате: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "правило повторения указано в неправильном формате"})
		return
	}

	err = h.repo.DoneTask(nextDate, newTask.Id)
	if err != nil {
		h.app.Log.Debugf("UpdateTask repo.UpdateTask ошибка добавления в бд: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Задача не найдена"})
		return
	}

	// Ответ в формате JSON
	c.JSON(http.StatusOK, gin.H{})
}

func (h *Handler) DeleteTask(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		h.app.Log.Debug("DeleteTask идентификатор задачи обязателен:")
		c.JSON(http.StatusBadRequest, gin.H{"error": "идентификатор задачи обязателен"})
		return
	}

	err := h.repo.DeleteTask(id)
	if err != nil {
		h.app.Log.Debugf("DeleteTask repoTasks: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Задача не найдена"})
		return
	}
	// Ответ в формате JSON
	c.JSON(http.StatusOK, gin.H{})
}
