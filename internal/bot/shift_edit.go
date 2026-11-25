package bot

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"tg_bot/model"
	"tg_bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) sendShiftUpdatedMessage(
	chatID int64,
	shiftID int,
	text string,
) {
	button := tgbotapi.NewInlineKeyboardButtonData(
		"↩️ Вернуться к смене",
		fmt.Sprintf("calendar:shift:%d", shiftID),
	)

	msg := tgbotapi.NewMessage(chatID, text)

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(button),
	)

	sent, err := b.api.Send(msg)
	if err != nil {
		log.Println("ошибка отправки результата редактирования:", err)
		return
	}

	b.activeInlineMessages[chatID] = sent.MessageID
}

func (b *Bot) handleEditingStartTime(message *tgbotapi.Message) {
	startTime, err := parseTime(message.Text)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Неверный формат времени.\n\nВведи время в формате ЧЧ:ММ, например: 10:00",
		)
		b.api.Send(msg)
		return
	}

	state := b.userStates[message.Chat.ID]

	shifts, err := b.shiftService.GetShifts(
		context.Background(),
		message.Chat.ID,
	)
	if err != nil {
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"Ошибка получения смены.",
		))
		return
	}

	var oldShift *model.Shift

	for i := range shifts {
		if shifts[i].ID == state.ShiftID {
			oldShift = &shifts[i]
			break
		}
	}

	if oldShift == nil {
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"Смена не найдена.",
		))
		delete(b.userStates, message.Chat.ID)
		return
	}

	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		log.Println("ошибка загрузки часового пояса:", err)
		return
	}

	// Получаем дату смены по времени Уфы.
	shiftDate := oldShift.StartTime.In(loc)

	// Объединяем дату смены с введённым временем.
	state.StartTime = time.Date(
		shiftDate.Year(),
		shiftDate.Month(),
		shiftDate.Day(),
		startTime.Hour(),
		startTime.Minute(),
		0,
		0,
		loc,
	)

	state.Step = "editing_end_time"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"🕐 Теперь введи новое время окончания смены в формате ЧЧ:ММ:",
	)

	msg.ReplyMarkup = mainKeyboard()

	b.api.Send(msg)
}

func (b *Bot) handleEditingEndTime(message *tgbotapi.Message) {
	endTime, err := parseTime(message.Text)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Неверный формат времени.\n\nВведи время в формате ЧЧ:ММ, например: 18:00",
		)
		b.api.Send(msg)
		return
	}

	state := b.userStates[message.Chat.ID]

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

	shift := model.Shift{
		ID:         state.ShiftID,
		TelegramID: message.Chat.ID,
		StartTime:  state.StartTime,
		EndTime:    state.EndTime,
	}

	oldShift, err := b.shiftService.GetShiftByID(
		context.Background(),
		state.ShiftID,
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось найти смену.",
		)

		b.sendShiftUpdatedMessage(
			message.Chat.ID,
			state.ShiftID,
			msg.Text,
		)

		delete(b.userStates, message.Chat.ID)

		delete(b.userStates, message.Chat.ID)
		return
	}

	shift.Break = oldShift.Break
	shift.HourlyRate = oldShift.HourlyRate

	// Проверяем новую смену перед сохранением.
	if _, err := service.CalculateHours(shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Некорректное время смены: "+err.Error(),
		)
		b.api.Send(msg)

		return
	}

	if _, err := service.CalculateSalary(shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось пересчитать зарплату: "+err.Error(),
		)
		b.api.Send(msg)

		return
	}

	err = b.shiftService.UpdateShift(
		context.Background(),
		state.ShiftID,
		message.From.ID,
		shift,
	)
	if err != nil {
		log.Println("ошибка обновления смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось обновить смену: "+err.Error(),
		)
		b.api.Send(msg)

		return
	}

	salary, err := service.CalculateSalary(shift)
	if err != nil {
		log.Println("ошибка расчёта зарплаты:", err)
		return
	}

	hours, err := service.CalculateHours(shift)
	if err != nil {
		log.Println("ошибка расчёта часов:", err)
		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Смена #%d обновлена!\n\n"+
				"🕐 Время: %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ Часы: %.2f\n"+
				"💰 Зарплата: %.2f ₽",
			state.ShiftID,
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			int(shift.Break.Minutes()),
			hours,
			salary,
		),
	)

	msg.ReplyMarkup = mainKeyboard()

	b.api.Send(msg)

	delete(b.userStates, message.Chat.ID)
}

func (b *Bot) handleEditingBreak(message *tgbotapi.Message) {
	breakMinutes, err := strconv.Atoi(message.Text)
	if err != nil || breakMinutes < 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверное значение.\n\nВведи количество минут, например: 60",
		)
		b.api.Send(msg)
		return
	}

	state := b.userStates[message.Chat.ID]

	shift, err := b.shiftService.GetShiftByID(
		context.Background(),
		state.ShiftID,
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось найти смену.",
		)
		b.api.Send(msg)

		delete(b.userStates, message.Chat.ID)
		return
	}

	shift.Break = time.Duration(breakMinutes) * time.Minute

	if _, err := service.CalculateHours(*shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Нельзя установить такой перерыв: "+err.Error(),
		)
		b.api.Send(msg)
		return
	}

	if _, err := service.CalculateSalary(*shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось пересчитать зарплату: "+err.Error(),
		)
		b.api.Send(msg)
		return
	}

	err = b.shiftService.UpdateShift(
		context.Background(),
		state.ShiftID,
		message.From.ID,
		*shift,
	)
	if err != nil {
		log.Println("ошибка обновления перерыва:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось обновить перерыв.",
		)
		b.api.Send(msg)
		return
	}

	hours, err := service.CalculateHours(*shift)
	if err != nil {
		log.Println("ошибка расчёта часов:", err)
		return
	}

	salary, err := service.CalculateSalary(*shift)
	if err != nil {
		log.Println("ошибка расчёта зарплаты:", err)
		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Смена #%d обновлена!\n\n"+
				"🕐 Время: %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ Часы: %.2f\n"+
				"💰 Зарплата: %.2f ₽",
			state.ShiftID,
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			breakMinutes,
			hours,
			salary,
		),
	)

	b.sendShiftUpdatedMessage(
		message.Chat.ID,
		state.ShiftID,
		msg.Text,
	)

	delete(b.userStates, message.Chat.ID)

	delete(b.userStates, message.Chat.ID)
}

func (b *Bot) handleEditingRate(message *tgbotapi.Message) {
	rate, err := strconv.ParseFloat(strings.TrimSpace(message.Text), 64)
	if err != nil || math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 || rate > 10000 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверная ставка.\n\nВведи положительное число, например: 300",
		)
		b.api.Send(msg)
		return
	}

	state := b.userStates[message.Chat.ID]

	shift, err := b.shiftService.GetShiftByID(
		context.Background(),
		state.ShiftID,
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось найти смену.",
		)
		b.api.Send(msg)

		delete(b.userStates, message.Chat.ID)
		return
	}

	shift.HourlyRate = rate

	if _, err := service.CalculateHours(*shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось рассчитать продолжительность смены: "+err.Error(),
		)
		b.api.Send(msg)
		return
	}

	if _, err := service.CalculateSalary(*shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось пересчитать зарплату: "+err.Error(),
		)
		b.api.Send(msg)
		return
	}

	err = b.shiftService.UpdateShift(
		context.Background(),
		state.ShiftID,
		message.From.ID,
		*shift,
	)
	if err != nil {
		log.Println("ошибка обновления ставки:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось обновить ставку.",
		)
		b.api.Send(msg)
		return
	}

	hours, err := service.CalculateHours(*shift)
	if err != nil {
		log.Println("ошибка расчёта часов:", err)
		return
	}

	salary, err := service.CalculateSalary(*shift)
	if err != nil {
		log.Println("ошибка расчёта зарплаты:", err)
		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Смена #%d обновлена!\n\n"+
				"🕐 Время: %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ Часы: %.2f\n"+
				"💰 Новая ставка: %.2f ₽/час\n"+
				"💵 Зарплата: %.2f ₽",
			state.ShiftID,
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			int(shift.Break.Minutes()),
			hours,
			rate,
			salary,
		),
	)

	b.sendShiftUpdatedMessage(
		message.Chat.ID,
		state.ShiftID,
		msg.Text,
	)

	delete(b.userStates, message.Chat.ID)

	delete(b.userStates, message.Chat.ID)
}
