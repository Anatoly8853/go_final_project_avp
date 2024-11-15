# Todo Scheduler Web Server

## Описание проекта

Проект представляет собой веб-сервер для планировщика задач с поддержкой баз данных SQLite.
Он позволяет создавать, редактировать, удалять задачи, а также управлять выполнением задач через API. 
Проект включает JWT-аутентификацию для защиты API-запросов и имеет интерфейс для работы с задачами.

## Выполненные задания со звёздочкой

- **Задания повышенной сложности выполнялись**. 
- Установка порта через переменную окружения (выполнено)
- Установка названия файла базы данных через переменную окружения (выполнено)
- Правила повторения задач по неделям и месяцам (выполнено)
- Функционал поиска задач (выполнено)
- Функционал аутентификации (выполнено)
- Создание докер образа (выполнено)

## Инструкция по запуску кода локально
Приложение требует Go версии 1.22.0
Для запуска приложения необходимо выполнить

go run main.go

Программа использует переменные окружения прописанные в файле .env:

TODO_PORT: используется для определения порта, через который будет осуществлен запуск. По умолчанию - 7540.

TODO_DBFILE: используется для определения директории и название файла базы данных. 
По умолчанию файл сохраняется в директории проекта с названием app/scheduler.db.

TODO_PASSWORD: используется для пароля аутентификации на сервере.

TODO_JWT_SECRET: используется для JWT секретная фраза.

### Структура проекта:

Директория `.github/workflows` содержит файл `go.yml` сборка проверка GitHub. 

Директория `app` содержит файл `scheduler.db` базы данных.

Директория `cmd` содержит файл `main.go` основной файл для запуска проекта.

Директория `config` содержит файл `config.go` конфигурация проекта.

Директория `handlers` содержит файл `handlers.go` и `auth.go` обработка команд сервера проекта.

Директория `log` содержит файлы логов проекта.

Директория `repository` содержит файл `repository` функции для работы с БД SQLite.

Директория `tasks` содержит файл `tasks` структура и вспомогательные функции.

Директория `tests` находятся тесты для проверки API, которое должно быть реализовано в веб-сервере.

Директория `web` содержит файлы фронтенда.

Файл `.env` переменные окружения.

Файл `Dockerfile` сборка и запуск Docker-образа.



