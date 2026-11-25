package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"tg_bot/model"
	"tg_bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// startOfShiftDay возвращает начало выбранного дня для создания смены.
func startOfShiftDay(date time.Time) time.Time {
	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		0, 0, 0, 0,
		date.Location(),
	)
}

func (b *Bot) handleStartTime(message *tgbotapi.Message) {
	startTime, err := parseTime(message.Text)

	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Неверный формат времени.\n\nВведи время в формате ЧЧ:ММ, например: 10:00",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	state := b.userStates[message.Chat.ID]
	if state == nil {
		return
	}

	// Используем дату, которую пользователь ввёл перед временем начала.
	startTime = time.Date(
		state.StartTime.Year(),
		state.StartTime.Month(),
		state.StartTime.Day(),
		startTime.Hour(),
		startTime.Minute(),
		0,
		0,
		state.StartTime.Location(),
	)

	state.StartTime = startTime
	state.Step = "waiting_end_time"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"Время начала: "+startTime.Format("15:04")+
			"\n\nТеперь введи время окончания смены в формате ЧЧ:ММ:",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}
}

func (b *Bot) handleEndTime(message *tgbotapi.Message) {
	endTime, err := parseTime(message.Text)

	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Неверный формат времени.\n\nВведи время в формате ЧЧ:ММ, например: 18:00",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	state := b.userStates[message.Chat.ID]
	if state == nil {
		return
	}

	endTime = time.Date(
		state.StartTime.Year(),
		state.StartTime.Month(),
		state.StartTime.Day(),
		endTime.Hour(),
		endTime.Minute(),
		0,
		0,
		state.StartTime.Location(),
	)

	if endTime.Before(state.StartTime) {
		endTime = endTime.Add(24 * time.Hour)
	}

	state.EndTime = endTime

	log.Printf(
		"Пользователь %d ввёл время окончания: %s",
		message.Chat.ID,
		endTime.Format("15:04"),
	)

	state.Step = "waiting_break"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"Время окончания: "+endTime.Format("15:04")+
			"\n\nТеперь введи продолжительность перерыва в минутах.\nНапример: 60",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}
}

func (b *Bot) handleBreakTime(message *tgbotapi.Message) {
	breakMinutes, err := strconv.Atoi(message.Text)
	if err != nil || breakMinutes < 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверное значение.\n\nВведи количество минут, например: 60",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	state := b.userStates[message.Chat.ID]

	rate, err := b.userRepo.GetHourlyRate(
		context.Background(),
		message.Chat.ID,
	)

	if err != nil || rate <= 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Часовая ставка не установлена.\n\nСначала нажми «💰 Указать ставку».",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		delete(b.userStates, message.Chat.ID)
		return
	}

	shift := model.Shift{
		TelegramID: message.Chat.ID,
		StartTime:  state.StartTime,
		EndTime:    state.EndTime,
		Break:      time.Duration(breakMinutes) * time.Minute,
		HourlyRate: rate,
	}

	hours, err := service.CalculateHours(shift)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось рассчитать продолжительность смены: "+err.Error(),
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	salary, err := service.CalculateSalary(shift)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось рассчитать зарплату: "+err.Error(),
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	_, err = b.shiftService.CreateShift(
		context.Background(),
		shift,
	)

	if err != nil {
		log.Println("ошибка сохранения смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось сохранить смену: "+err.Error(),
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Смена рассчитана!\n\n"+
				"📅 Дата: %s\n"+
				"🕐 Время: %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ Часы: %.2f\n"+
				"💰 Ставка: %.2f ₽/час\n"+
				"💵 Зарплата: %.2f ₽",
			shift.StartTime.Format("02.01.2006"),
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			breakMinutes,
			hours,
			rate,
			salary,
		),
	)

	// Кнопка возврата к дню добавленной смены.
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		log.Println("ошибка загрузки часового пояса:", err)
		return
	}

	shiftDate := shift.StartTime.In(loc)

	backButton := tgbotapi.NewInlineKeyboardButtonData(
		"🗓 Вернуться к календарю",
		"calendar:day:"+shiftDate.Format("2006-01-02"),
	)

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(backButton),
	)

	sent, err := b.api.Send(msg)
	if err != nil {
		log.Println("ошибка отправки сообщения:", err)
		return
	}

	// Регистрируем сообщение с кнопкой как активное.
	b.activeInlineMessages[message.Chat.ID] = sent.MessageID

	delete(b.userStates, message.Chat.ID)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse("15:04", value)
}
