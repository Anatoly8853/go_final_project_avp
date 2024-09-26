package repository

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"go_final_project_avp/config"
	"go_final_project_avp/tasks"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
	db, err = sqlx.Connect("sqlite3", cfg.DBFile)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// RunMigrations миграция таблиц в БД если отсутствуют.
func (r *Repository) RunMigrations(config config.Config) error {
	dbfile := "scheduler.db"

	appPath, err := os.Executable()
	if err != nil {
		r.app.Log.Fatal(err)
		return err
	}

	if len(config.DBFile) > 0 {
		dbfile = config.DBFile
	}

	dbFile := filepath.Join(filepath.Dir(appPath), dbfile)
	if _, err = os.Stat(dbFile); os.IsNotExist(err) {
		r.app.Log.Infof("Файл базы данных не найден, создаём новый: %s", config.DBFile)
		return nil
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
	res, err := r.db.QueryContext(ctx, getTasks, limit)
	if err != nil {
		r.app.Log.Debugf("GetTasks QueryContext: %s", err)
		return nil, err
	}

	defer res.Close()

	var task []tasks.Task
	for res.Next() {
		t := tasks.Task{}

		err = res.Scan(&t.Id, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			r.app.Log.Debugf("GetTasks Scan: %s", err)
			return nil, err
		}

		task = append(task, t)
	}

	if err = res.Err(); err != nil {
		r.app.Log.Debugf("GetTasks res.Err: %s", err)
		return nil, err
	}

	return task, nil
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
		r.app.Log.Debugf("GetTasksId не удается преобразовать id - %v : %v", id, err)
		return t, err
	}

	res := r.db.QueryRowContext(ctx, getTasksId, ids)

	err = res.Scan(&t.Id, &t.Date, &t.Title, &t.Comment, &t.Repeat)

	if t.Id == "" {
		err = fmt.Errorf("задача не найдена c id - %v", id)
		r.app.Log.Debugf("GetTasksId : %v", err)
		return t, err
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
		r.app.Log.Debugf("CreateTask ExecContext: %s", err)
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		r.app.Log.Debugf("CreateTask res.LastInsertId(): %s", err)
		return 0, err
	}

	return id, nil
}

// GetSearch выбрать задачи через строку поиска.
func (r *Repository) GetSearch(search string) ([]tasks.Task, error) {
	ctx := context.Background()

	// Формируем запрос
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
	query += " ORDER BY date ASC"

	query += " LIMIT ?"
	args = append(args, limit)

	// Выполнение запроса
	res, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.app.Log.Debugf("GetSearch QueryContext Ошибка выполнения запроса: %s", err)
		return nil, err
	}
	defer res.Close()

	var task []tasks.Task // Инициализация пустого слайса

	for res.Next() {
		var t tasks.Task
		if err = res.Scan(&t.Id, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			r.app.Log.Debugf("GetSearch res.Scan Ошибка обработки результата: %s", err)
			return nil, err
		}
		task = append(task, t)
	}

	// Возврат пустого слайса, если задач нет
	if task == nil {
		task = []tasks.Task{}
	}

	if err = res.Err(); err != nil {
		r.app.Log.Debugf("GetSearch res.Err: %s", err)
		return nil, err
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

// UpdateTask обновляем данные в бд.
func (r *Repository) UpdateTask(task *tasks.Task) error {
	ctx := context.Background()
	_, err := r.db.ExecContext(ctx, updateTask, task.Date, task.Title, task.Comment, task.Repeat, task.Id)
	if err != nil {
		r.app.Log.Debugf("UpdateTask ExecContext: %s", err)
		return err
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
	_, err := r.db.ExecContext(ctx, doneTask, taskDate, id)
	if err != nil {
		r.app.Log.Debugf("DoneTask ExecContext: %s", err)
		return err
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
		r.app.Log.Debugf("GetTasksId не удается преобразовать id - %v : %v", id, err)
		return err
	}

	res, err := r.db.ExecContext(ctx, deleteTask, ids)
	if err != nil {
		r.app.Log.Debugf("DeleteTask ExecContext: %s", err)
		return err
	}

	count, err := res.RowsAffected()
	if err != nil {
		r.app.Log.Debugf("DeleteTask res.RowsAffected(): %s", err)
		return err
	}

	if count == 0 {
		r.app.Log.Debug("DeleteTask нет такого id:", id)
		return err
	}

	return nil
}
