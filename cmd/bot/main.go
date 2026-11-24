package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
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

	var client *http.Client

	if os.Getenv("USE_PROXY") == "true" {
		proxyAddress := os.Getenv("SOCKS5_PROXY")
		if proxyAddress == "" {
			log.Fatal("SOCKS5_PROXY не указан в .env")
		}

		proxyURL, err := url.Parse(proxyAddress)
		if err != nil {
			log.Fatal("ошибка адреса прокси:", err)
		}

		dialer, err := proxy.FromURL(proxyURL, &net.Dialer{
			Timeout: 10 * time.Second,
		})
		if err != nil {
			log.Fatal("ошибка настройки прокси:", err)
		}

		client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			},
			Timeout: 60 * time.Second,
		}

		log.Println("Telegram: подключение через SOCKS5")
	} else {
		client = &http.Client{
			Timeout: 60 * time.Second,
		}

		log.Println("Telegram: прямое подключение")
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
	extraRepo := repository.NewExtraRepository(db)

	shiftService := service.NewShiftService(shiftRepo)
	salaryService := service.NewSalaryService(calendarRepo)

	// Пока SalaryService не подключён к обработчикам Telegram.
	// На следующем этапе передадим его в botservice.New.

	bot := botservice.New(api, userRepo, shiftService, salaryService, extraRepo)
	bot.Start()
}
