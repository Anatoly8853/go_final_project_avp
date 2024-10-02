FROM golang:1.23.2

# Определяем переменные окружения
ENV TODO_PORT=7540
ENV TODO_DBFILE=/app/scheduler.db
ENV TODO_PASSWORD=12345

WORKDIR /app

# Копируем модульные файлы и весь код за один раз
COPY . .

# Указываем переменные для кросс-компиляции
ENV GOOS=linux GOARCH=amd64

# Сборка проекта
RUN go build -o /gofinalprojectavp ./cmd/main.go

# Открываем порт для веб-сервера (учитываем значение переменной TODO_PORT)
EXPOSE ${TODO_PORT}

CMD ["/gofinalprojectavp"]