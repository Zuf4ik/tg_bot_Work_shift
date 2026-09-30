package main

import (
	"log"

	"tg_bot/config"
	botservice "tg_bot/internal/bot"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatal("ошибка загрузки .env:", err)
	}

	api, err := tgbotapi.NewBotAPI(config.GetBotToken())
	if err != nil {
		log.Fatal("ошибка создания Telegram-бота:", err)
	}

	log.Printf("Бот подключен: @%s", api.Self.UserName)

	bot := botservice.New(api)
	bot.Start()
}
