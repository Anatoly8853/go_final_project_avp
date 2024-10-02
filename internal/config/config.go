package config

import (
	"go_final_project_avp/internal/loggers"

	"github.com/gookit/slog"
	"github.com/spf13/viper"
)

type Application struct {
	Log *slog.Logger
}

func SetupApplication() *Application {
	// Настройка логгера перед его инициализацией
	loggers.SetLogConsole(false) // Логи будут записываться в файл
	loggers.SetIsDebugMode(true)
	loggers.SetIsInfoMode(true)
	loggers.SetIsWarnMode(true)
	// Настраиваем логгер
	logger := loggers.SetupLogger()
	// Создаем экземпляр Application с настроенным логгером
	return &Application{Log: logger}
}

type Config struct {
	Port      string `mapstructure:"TODO_PORT"`
	DBFile    string `mapstructure:"TODO_DBFILE"`
	Password  string `mapstructure:"TODO_PASSWORD"`
	JwtSecret string `mapstructure:"TODO_JWT_SECRET"`
}

func LoadConfig(app *Application) (cfg Config) {
	// Чтение файла app.env
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	// Чтение переменных окружения
	viper.AutomaticEnv() // Автоматически читать переменные окружения
	// Попытка чтения из файла конфигурации
	if err := viper.MergeInConfig(); err != nil {
		app.Log.Printf("Error reading .env file, %s", err)
	}
	// Если не удалось прочитать файл конфигурации
	if err := viper.ReadInConfig(); err != nil {
		app.Log.Fatalf("Error reading config file, %s", err)
	}
	// Декодирование конфигурации в структуру
	err := viper.Unmarshal(&cfg)
	if err != nil {
		app.Log.Fatalf("unable to decode into struct, %v", err)
	}

	return cfg
}
