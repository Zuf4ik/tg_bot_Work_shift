package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"tg_bot/model"
	"tg_bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleIncomeMenu(message *tgbotapi.Message) {
	delete(b.userStates, message.Chat.ID)
	msg := tgbotapi.NewMessage(message.Chat.ID, "📊 За какой период показать статистику?")
	msg.ReplyMarkup = incomePeriodKeyboard()
	if _, err := b.api.Send(msg); err != nil {
		log.Println("не удалось показать выбор периода дохода:", err)
	}
}

func (b *Bot) handleCurrentMonthIncome(message *tgbotapi.Message) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		log.Printf("ошибка загрузки часового пояса: %v", err)
		return
	}

	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	b.sendIncomeReport(message.Chat.ID, start, end)
}

func (b *Bot) handlePreviousMonthIncome(message *tgbotapi.Message) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		log.Printf("ошибка загрузки часового пояса: %v", err)
		return
	}

	now := time.Now().In(loc)
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	start := thisMonth.AddDate(0, -1, 0)
	end := thisMonth.Add(-time.Nanosecond)

	b.sendIncomeReport(message.Chat.ID, start, end)
}

func (b *Bot) handleIncomeStartDate(message *tgbotapi.Message) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		log.Printf("ошибка загрузки часового пояса: %v", err)
		return
	}
	date, err := time.ParseInLocation(
		"02.01.2006",
		strings.TrimSpace(message.Text),
		loc,
	)
	if err != nil {
		b.api.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Не понял дату. Введи её в формате ДД.ММ.ГГГГ, например 01.10.2026."))
		return
	}
	if date.Year() < 2020 || date.Year() > 2036 {
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Год должен быть в диапазоне от 2020 до 2036.",
		))
		return
	}
	state := b.userStates[message.Chat.ID]
	if state == nil {
		return
	}
	state.ReportStart = time.Date(
		date.Year(), date.Month(), date.Day(),
		0, 0, 0, 0,
		loc,
	)
	state.Step = "waiting_income_end"
	b.api.Send(tgbotapi.NewMessage(message.Chat.ID, "Теперь введи конечную дату периода в формате ДД.ММ.ГГГГ. Например: 31.10.2026"))
}

func (b *Bot) handleIncomeEndDate(message *tgbotapi.Message) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		log.Printf("ошибка загрузки часового пояса: %v", err)
		return
	}
	date, err := time.ParseInLocation(
		"02.01.2006",
		strings.TrimSpace(message.Text),
		loc,
	)
	if err != nil {
		b.api.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Не понял дату. Введи её в формате ДД.ММ.ГГГГ, например 31.10.2026."))
		return
	}
	if date.Year() < 2020 || date.Year() > 2036 {
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Год должен быть в диапазоне от 2020 до 2036.",
		))
		return
	}
	state := b.userStates[message.Chat.ID]
	if state == nil {
		return
	}
	end := time.Date(
		date.Year(), date.Month(), date.Day(),
		23, 59, 59, int(time.Second-time.Nanosecond),
		loc,
	)
	start := state.ReportStart
	if end.Before(start) {
		b.api.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Конечная дата раньше начальной. Введи конечную дату ещё раз."))
		return
	}
	delete(b.userStates, message.Chat.ID)
	b.sendIncomeReport(message.Chat.ID, start, end)
}

func (b *Bot) sendIncomeReport(chatID int64, start, end time.Time) {

	shifts, err := b.shiftService.GetShifts(context.Background(), chatID)
	if err != nil {
		log.Println("ошибка получения смен для отчёта:", err)
		b.api.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось сформировать отчёт о доходах."))
		return
	}

	now := time.Now()
	var actualSalary, forecastSalary, actualHours, forecastHours float64
	var actualExtras, forecastExtras float64
	var actualCount, forecastCount int

	for _, shift := range shifts {
		// В отчёт попадают смены, дата начала которых входит в выбранный диапазон.

		if shift.StartTime.Before(start) || shift.StartTime.After(end) {
			continue
		}
		salary, salaryErr := service.CalculateSalary(shift)
		hours, hoursErr := service.CalculateHours(shift)
		if salaryErr != nil || hoursErr != nil {
			log.Printf("пропущена смена #%d при расчёте отчёта: salary=%v hours=%v", shift.ID, salaryErr, hoursErr)
			continue
		}

		// Получаем дополнительные работы для этой смены.
		extras, err := b.extraRepo.GetExtras(
			context.Background(),
			chatID,
			shift.ID,
		)
		if err != nil {
			log.Printf("ошибка получения дополнительных работ смены #%d: %v", shift.ID, err)
			b.api.Send(tgbotapi.NewMessage(
				chatID,
				"❌ Не удалось получить дополнительные работы для отчёта.",
			))
			return
		}

		// Рассчитываем дополнительные выплаты.
		extraSalary, err := service.CalculateExtraSalary(shift, extras)
		if err != nil {
			log.Printf("ошибка расчёта дополнительных работ смены #%d: %v", shift.ID, err)
			b.api.Send(tgbotapi.NewMessage(
				chatID,
				"❌ Не удалось рассчитать дополнительные работы.",
			))
			return
		}

		// Основная зарплата + дополнительные выплаты.
		salary += extraSalary

		if !shift.EndTime.After(now) {
			actualSalary += salary
			actualExtras += extraSalary
			actualHours += hours
			actualCount++
		} else {
			forecastSalary += salary
			forecastExtras += extraSalary
			forecastHours += hours
			forecastCount++
		}

	}

	text := fmt.Sprintf(
		"📊 Статистика за период\n"+
			"📅 %s — %s\n\n"+
			"✅ Заработано (завершённые смены): %.2f ₽\n"+
			"   Смен: %d · Часов: %.2f\n\n"+
			"🔮 Ожидается (незавершённые/плановые смены): %.2f ₽\n"+
			"   Смен: %d · Часов: %.2f\n\n"+
			"📊 Итого с учётом прогноза: %.2f ₽",
		start.Format("02.01.2006"),
		end.Format("02.01.2006"),
		actualSalary,
		actualCount,
		actualHours,
		forecastSalary,
		forecastCount,
		forecastHours,
		actualSalary+forecastSalary,
	)

	text += fmt.Sprintf(
		"\n\n📦 Дополнительные работы\n"+
			"✅ За завершённые смены: %.2f ₽\n"+
			"🔮 За плановые смены: %.2f ₽\n"+
			"💰 Всего доплат: %.2f ₽",
		actualExtras,
		forecastExtras,
		actualExtras+forecastExtras,
	)

	// Премию показываем только для полного календарного месяца.
	monthStart := time.Date(
		start.Year(), start.Month(), 1,
		0, 0, 0, 0, start.Location(),
	)
	nextMonth := monthStart.AddDate(0, 1, 0)

	if start.Equal(monthStart) &&
		!end.Before(nextMonth.Add(-time.Second)) &&
		end.Before(nextMonth) {

		var completedShifts []model.Shift
		var allMonthShifts []model.Shift

		for _, shift := range shifts {
			if shift.StartTime.Before(monthStart) ||
				!shift.StartTime.Before(nextMonth) {
				continue
			}

			allMonthShifts = append(allMonthShifts, shift)

			if !shift.EndTime.After(now) {
				completedShifts = append(completedShifts, shift)
			}
		}

		actualBonus, actualErr := b.salaryService.CalculateBonusFromShifts(
			context.Background(),
			monthStart.Year(),
			monthStart.Month(),
			completedShifts,
		)

		forecastBonus, forecastErr := b.salaryService.CalculateBonusFromShifts(
			context.Background(),
			monthStart.Year(),
			monthStart.Month(),
			allMonthShifts,
		)

		if actualErr != nil || forecastErr != nil {
			log.Printf(
				"ошибка расчёта месячной премии: actual=%v forecast=%v",
				actualErr, forecastErr,
			)
			text += "\n\n⚠️ Не удалось рассчитать месячную премию."
		} else {
			text += fmt.Sprintf(
				"\n\n📊 Месячная статистика\n"+
					"⏱ Отработано: %.2f ч\n"+
					"📅 Норма: %.2f ч\n"+
					"⏳ До нормы: %.2f ч\n"+
					"💎 Премия за завершённые смены: %.2f ₽\n"+
					"🔮 Премия с учётом плана: %.2f ₽",
				actualBonus.WorkedHours,
				actualBonus.NormHours,
				actualBonus.Remaining,
				actualBonus.Bonus,
				forecastBonus.Bonus,
			)
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = mainKeyboard()
	if _, err := b.api.Send(msg); err != nil {
		log.Println("не удалось отправить отчёт о доходах:", err)
	}
}
