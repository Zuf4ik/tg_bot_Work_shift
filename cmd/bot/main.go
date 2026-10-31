package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"tg_bot/config"
	botservice "tg_bot/internal/bot"
	"tg_bot/internal/database"
	"tg_bot/internal/repository"
	"tg_bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/net/proxy"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatal("ошибка загрузки .env:", err)
	}

	proxyURL, err := url.Parse("socks5://127.0.0.1:10808")
	if err != nil {
		log.Fatal("ошибка адреса прокси:", err)
	}

	dialer, err := proxy.FromURL(proxyURL, &net.Dialer{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatal("ошибка настройки прокси:", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		},
		Timeout: 60 * time.Second,
	}

	api, err := tgbotapi.NewBotAPIWithClient(
		config.GetBotToken(),
		tgbotapi.APIEndpoint,
		client,
	)
	if err != nil {
		log.Fatal("ошибка создания Telegram-бота:", err)
	}

	log.Printf("Бот подключен: @%s", api.Self.UserName)

	db, err := database.Connect()
	if err != nil {
		log.Fatal("ошибка подключения к базе данных:", err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	shiftRepo := repository.NewShiftRepository(db)
	calendarRepo := repository.NewCalendarRepository(db)

	shiftService := service.NewShiftService(shiftRepo)
	salaryService := service.NewSalaryService(calendarRepo)

	// Пока SalaryService не подключён к обработчикам Telegram.
	// На следующем этапе передадим его в botservice.New.

	bot := botservice.New(api, userRepo, shiftService, salaryService)
	bot.Start()
}
