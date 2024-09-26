FROM golang:1.22.0

# Определяем переменные окружения
ENV TODO_PORT=7540
ENV TODO_DBFILE=/scheduler.db
ENV TODO_PASSWORD=12345

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o /gofinalprojectavp ./cmd/main.go

# Открываем порт для веб-сервера
EXPOSE 7540

CMD ["/gofinalprojectavp"]