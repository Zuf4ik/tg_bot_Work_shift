package config

import (
	"os"

	"github.com/joho/godotenv"
)

func Load() error {
	return godotenv.Load("config/.env")
}

func GetBotToken() string {
	return os.Getenv("BOT_TOKEN")
}

func GetDBHost() string {
	return os.Getenv("DB_HOST")
}

func GetDBPort() string {
	return os.Getenv("DB_PORT")
}

func GetDBUser() string {
	return os.Getenv("DB_USER")
}

func GetDBPassword() string {
	return os.Getenv("DB_PASSWORD")
}

func GetDBName() string {
	return os.Getenv("DB_NAME")
}
