package config

import (
	slogavp "github.com/Anatoly8853/slog-avp/v2"
	"github.com/spf13/viper"
)

type Config struct {
	Port      string `mapstructure:"TODO_PORT"`
	DBFile    string `mapstructure:"TODO_DBFILE"`
	Password  string `mapstructure:"TODO_PASSWORD"`
	JwtSecret string `mapstructure:"TODO_JWT_SECRET"`
}

func LoadConfig(app *slogavp.Application) (cfg Config) {
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
