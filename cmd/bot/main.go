package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"tg_bot/config"
	"tg_bot/internal/database"
	"tg_bot/internal/repository"
	"tg_bot/model"
	"tg_bot/service"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatal("ошибка загрузки .env:", err)
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatal("ошибка подключения к БД:", err)
	}
	defer db.Close()

	repo := repository.NewShiftRepository(db)
	shiftService := service.NewShiftService(repo)

	ctx := context.Background()

	shift := model.Shift{
		StartTime:  time.Date(2026, 9, 28, 20, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 9, 29, 2, 0, 0, 0, time.Local),
		Break:      0,
		HourlyRate: 250,
	}

	id, err := shiftService.CreateShift(ctx, shift)
	if err != nil {
		log.Fatal("ошибка создания смены:", err)
	}

	fmt.Println("Смена создана через Service, ID:", id)

	shifts, err := shiftService.GetShifts(ctx)
	if err != nil {
		log.Fatal("ошибка получения смен:", err)
	}

	for _, s := range shifts {
		hours, err := service.CalculateHours(s)
		if err != nil {
			log.Fatal(err)
		}

		salary, err := service.CalculateSalary(s)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf(
			"ID: %d | %s - %s | %.2f ч. | %.2f руб.\n",
			s.ID,
			s.StartTime.Format("15:04"),
			s.EndTime.Format("15:04"),
			hours,
			salary,
		)
	}

	err = shiftService.DeleteShift(ctx, id)
	if err != nil {
		log.Fatal("ошибка удаления смены:", err)
	}

	fmt.Println("Тестовая смена удалена")
}
