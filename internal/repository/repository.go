package repository

import (
	"database/sql"
	"go_final_project_avp/internal/config"
	"go_final_project_avp/internal/tasks"

	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type Repository struct {
	db  *sqlx.DB
	app *config.Application
}

const limit = 50

func NewRepository(db *sqlx.DB, app *config.Application) *Repository {
	if db == nil {
		app.Log.Fatal("Ошибка: подключение к базе данных не инициализировано")
	}
	return &Repository{db: db, app: app}
}

// NewOpenDB подключение к БД.
func NewOpenDB(cfg config.Config) (db *sqlx.DB, err error) {
	dbDir := filepath.Dir(cfg.DBFile)

	// Проверяем, существует ли директория для базы данных
	if _, err = os.Stat(dbDir); os.IsNotExist(err) {
		// Создаем директорию, если она не существует
		if err = os.MkdirAll(dbDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("не удалось создать директорию %s: %w", dbDir, err)
		}
	}

	// Проверяем, существует ли файл базы данных
	if _, err = os.Stat(cfg.DBFile); os.IsNotExist(err) {
		// Создаем файл базы данных, если он не существует
		file, err := os.Create(cfg.DBFile)
		if err != nil {
			return nil, fmt.Errorf("не удалось создать файл базы данных %s: %w", cfg.DBFile, err)
		}
		defer func(file *os.File) {
			err = file.Close()
			if err != nil {

			}
		}(file) // Закрываем файл после создания
	}

	// Подключаемся к базе данных SQLite
	db, err = sqlx.Connect("sqlite3", cfg.DBFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	return db, nil
}

// RunMigrations миграция таблиц в БД если отсутствуют.
func (r *Repository) RunMigrations(cfg config.Config) error {
	dbfile := "app/scheduler.db"

	appPath, err := os.Executable()
	if err != nil {
		r.app.Log.Fatal(err)
		return err
	}

	if len(cfg.DBFile) > 0 {
		dbfile = cfg.DBFile
	}

	dbFile := filepath.Join(filepath.Dir(appPath), dbfile)
	if _, err = os.Stat(dbFile); os.IsNotExist(err) {
		r.app.Log.Infof("Файл базы данных не найден, создаём новый: %s", cfg.DBFile)
	}

	var install bool
	if err != nil {
		install = true
	}

	if !install {
		return err
	}

	ctx := context.Background()
	createTableScheduler := `CREATE TABLE IF NOT EXISTS scheduler (
     id INTEGER PRIMARY KEY AUTOINCREMENT,    
     date TEXT CHECK(LENGTH(date) <= 8) NOT NULL,
     title TEXT NOT NULL ,
     comment TEXT,
     repeat TEXT CHECK(LENGTH(repeat) <= 128)  -- строка до 128 символов
);`
	_, err = r.db.ExecContext(ctx, createTableScheduler)
	if err != nil {
		return err
	}

	createIndexDate := "CREATE INDEX IF NOT EXISTS index_date ON scheduler (date);"

	//index creation
	_, err = r.db.ExecContext(ctx, createIndexDate)
	if err != nil {
		return err
	}

	return nil
}

const getTasks = ` -- name: GetTasks
	SELECT id, date, title, comment, repeat
    FROM scheduler
    ORDER BY date ASC
    LIMIT $1
	`

// GetTasks получаем список ближайших задач.
func (r *Repository) GetTasks() ([]tasks.Task, error) {
	ctx := context.Background()

	// Выполняем запрос к БД
	res, err := r.db.QueryContext(ctx, getTasks, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса QueryContext: %w", err)
	}
	defer func(res *sql.Rows) {
		err = res.Close()
		if err != nil {

		}
	}(res)

	var tasksList []tasks.Task
	for res.Next() {
		var t tasks.Task
		// Сканируем результат в структуру
		if err = res.Scan(&t.Id, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			return nil, fmt.Errorf("ошибка сканирования задачи res.Scan: %w", err)
		}
		tasksList = append(tasksList, t)
	}

	// Проверяем на наличие ошибок после итерации
	if err = res.Err(); err != nil {
		return nil, fmt.Errorf("ошибка после обработки результата res.Err: %w", err)
	}

	return tasksList, nil
}

const getTasksId = ` -- name: GetTasksId
	SELECT id, date, title, comment, repeat
    FROM scheduler
    WHERE id = $1
	`

// GetTasksId получаем задачу по id.
func (r *Repository) GetTasksId(id string) (tasks.Task, error) {
	ctx := context.Background()
	t := tasks.Task{}

	ids, err := strconv.Atoi(id)
	if err != nil {
		return t, fmt.Errorf("не удается преобразовать id - %v : %v", id, err)
	}

	res := r.db.QueryRowContext(ctx, getTasksId, ids)

	err = res.Scan(&t.Id, &t.Date, &t.Title, &t.Comment, &t.Repeat)

	if t.Id == "" {
		return t, fmt.Errorf("задача не найдена c id - %v", id)
	}

	return t, nil
}

const createTask = ` -- name: CreateTask
	INSERT INTO scheduler 
	    (date, title, comment, repeat)
	VALUES ($1, $2, $3, $4)
	`

// CreateTask добавляем задачи в бд.
func (r *Repository) CreateTask(task *tasks.Task) (int64, error) {
	ctx := context.Background()
	res, err := r.db.ExecContext(ctx, createTask, task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return 0, fmt.Errorf("ошибка выполнения запроса  ExecContext: %s", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка нет id res.LastInsertId(): %s", err)
	}

	return id, nil
}

// GetSearch выбрать задачи через строку поиска.
func (r *Repository) GetSearch(search string) ([]tasks.Task, error) {
	ctx := context.Background()

	// Формируем запрос
	//WHERE 1=1 является трюком для упрощения добавления дополнительных условий в запрос
	//Здесь, если условия добавляются динамически, они всегда будут присоединены
	//через AND, что упрощает процесс формирования запросов.
	query := "SELECT id, date, title, comment, repeat FROM scheduler WHERE 1=1"
	var args []interface{}

	// Если указан search, проверяем его
	if search != "" {
		// Попытка парсинга как даты
		if date, err := time.Parse(tasks.DisplayDateFormat, search); err == nil {
			// Если search является датой
			query += " AND date = ?"
			args = append(args, date.Format(tasks.TimeFormat))
		} else {
			// Если search — это строка, ищем по заголовку и комментарию
			query += " AND (title LIKE ? OR comment LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm)
		}
	}

	// Добавляем сортировку по дате
	query += " ORDER BY date ASC LIMIT ?"

	args = append(args, limit)

	// Выполнение запроса
	res, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса QueryContext: %w", err)
	}
	defer func(res *sql.Rows) {
		err = res.Close()
		if err != nil {

		}
	}(res)

	var task []tasks.Task // Инициализация пустого слайса

	for res.Next() {
		var t tasks.Task
		if err = res.Scan(&t.Id, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			return nil, fmt.Errorf("ошибка сканирования задачи res.Scan: %w", err)
		}
		task = append(task, t)
	}

	// Возврат пустого слайса, если задач нет
	if task == nil {
		task = []tasks.Task{}
	}

	if err = res.Err(); err != nil {
		return nil, fmt.Errorf("ошибка после обработки результата res.Err: %w", err)
	}

	return task, nil
}

const updateTask = ` -- name: UpdateTask
	UPDATE scheduler 
    SET date = $1, 
        title = $2, 
        comment = $3, 
        repeat = $4 
    WHERE id = $5
	`

// UpdateTask обновляет данные в БД, если задача с таким ID существует.
func (r *Repository) UpdateTask(task *tasks.Task) error {
	ctx := context.Background()

	// Выполняем запрос на обновление
	result, err := r.db.ExecContext(ctx, updateTask, task.Date, task.Title, task.Comment, task.Repeat, task.Id)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса ExecContext: %w", err)
	}

	// Проверяем количество затронутых строк
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения затронутых строк: %w", err)
	}

	// Если ни одна строка не была обновлена, возвращаем ошибку
	if rowsAffected == 0 {
		return fmt.Errorf("задача с id %d не найдена", task.Id)
	}

	return nil
}

const doneTask = ` -- name: DoneTask
	UPDATE scheduler 
    SET date = $1
    WHERE id = $2
	`

// DoneTask отметка о выполнении.
func (r *Repository) DoneTask(taskDate string, id string) error {
	ctx := context.Background()
	result, err := r.db.ExecContext(ctx, doneTask, taskDate, id)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса ExecContext: %w", err)
	}
	// Проверяем количество затронутых строк
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения затронутых строк: %w", err)
	}

	// Если ни одна строка не была обновлена, возвращаем ошибку
	if rowsAffected == 0 {
		return fmt.Errorf("задача с id %d не найдена", id)
	}

	return nil
}

const deleteTask = ` -- name: DeleteTask
	DELETE FROM scheduler 
	       WHERE id = $1
`

// DeleteTask удаляем задачи из БД.
func (r *Repository) DeleteTask(id string) error {
	ctx := context.Background()

	ids, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("не удается преобразовать id - %v : %v", id, err)
	}

	res, err := r.db.ExecContext(ctx, deleteTask, ids)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса ExecContext: %w", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка res.RowsAffected(): %s", err)
	}

	if count == 0 {
		return fmt.Errorf("ошибка нет такого id: %v", id)
	}

	return nil
}
