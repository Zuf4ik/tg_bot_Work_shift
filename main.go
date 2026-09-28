// ------------------------------- TG ЗАПУСК -----------------------------
/*package main

import (
	"fmt"
	"log"

	"tg_bot/config"

	telebot "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	token := config.GetBotToken()

	if token == "" {
		log.Fatal("8884837370:AAGiJekPCq_ut3DRyHEpTfiCqLYUZpblmuw")
	}

	bot, err := telebot.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Bot connected successfully!")
	fmt.Println("Bot username:", bot.Self.UserName)
}
// ------------------------------- TG ЗАПУСК -----------------------------

*/

package main

import (
	"fmt"
	"time"

	"tg_bot/model"
	"tg_bot/service"
)

func main() {
	shift := model.Shift{
		StartTime:  time.Date(2026, 9, 27, 10, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 9, 27, 20, 0, 0, 0, time.Local),
		Break:      1 * time.Hour,
		HourlyRate: 250,
	}

	hours, err := service.CalculateHours(shift)
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	salary, err := service.CalculateSalary(shift)
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	fmt.Println("Отработано часов:", hours)
	fmt.Println("Зарплата:", salary, "₽")
}
