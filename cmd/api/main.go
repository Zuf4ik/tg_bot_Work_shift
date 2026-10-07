package main

import (
	"log"
	"net/http"

	"encoding/json"
	"strconv"
	"tg_bot/config"
	"tg_bot/internal/database"
	"tg_bot/internal/repository"
	"tg_bot/model"
	"tg_bot/service"
	"time"
)

type createShiftRequest struct {
	TelegramID   int64   `json:"telegram_id"`
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time"`
	BreakMinutes int     `json:"break_minutes"`
	HourlyRate   float64 `json:"hourly_rate"`
}

func main() {
	if err := config.Load(); err != nil {
		log.Fatal("ошибка загрузки .env:", err)
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatal("ошибка подключения к базе данных:", err)
	}
	defer db.Close()

	log.Println("Database connected!")

	shiftRepo := repository.NewShiftRepository(db)
	shiftService := service.NewShiftService(shiftRepo)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	http.Handle(
		"/api/shifts",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				getShiftsHandler(shiftService)(w, r)
			case http.MethodPost:
				createShiftHandler(shiftService)(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}),
	)

	http.Handle(
		"/api/shifts/{id}",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				getShiftByIDHandler(shiftService)(w, r)
			case http.MethodPut:
				updateShiftHandler(shiftService)(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}),
	)

	log.Println("API запущен на :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func getShiftsHandler(shiftService *service.ShiftService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("GET /api/shifts")
		telegramIDText := r.URL.Query().Get("telegram_id")

		if telegramIDText == "" {
			http.Error(w, "telegram_id is required", http.StatusBadRequest)
			return
		}

		telegramID, err := strconv.ParseInt(telegramIDText, 10, 64)
		if err != nil {
			http.Error(w, "invalid telegram_id", http.StatusBadRequest)
			return
		}

		shifts, err := shiftService.GetShifts(
			r.Context(),
			telegramID,
		)
		if err != nil {
			http.Error(w, "failed to get shifts", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(shifts); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func getShiftByIDHandler(shiftService *service.ShiftService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		telegramIDText := r.URL.Query().Get("telegram_id")
		if telegramIDText == "" {
			http.Error(w, "telegram_id is required", http.StatusBadRequest)
			return
		}

		telegramID, err := strconv.ParseInt(telegramIDText, 10, 64)
		if err != nil {
			http.Error(w, "invalid telegram_id", http.StatusBadRequest)
			return
		}

		shiftIDText := r.PathValue("id")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil {
			http.Error(w, "invalid shift id", http.StatusBadRequest)
			return
		}

		shift, err := shiftService.GetShiftByID(
			r.Context(),
			shiftID,
			telegramID,
		)
		if err != nil {
			http.Error(w, "shift not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(shift); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func updateShiftHandler(shiftService *service.ShiftService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		telegramIDText := r.URL.Query().Get("telegram_id")

		if telegramIDText == "" {
			http.Error(w, "telegram_id is required", http.StatusBadRequest)
			return
		}

		telegramID, err := strconv.ParseInt(telegramIDText, 10, 64)
		if err != nil {
			http.Error(w, "invalid telegram_id", http.StatusBadRequest)
			return
		}

		shiftID, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid shift id", http.StatusBadRequest)
			return
		}

		var req createShiftRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		startTime, err := time.Parse("2006-01-02T15:04:05", req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time", http.StatusBadRequest)
			return
		}

		endTime, err := time.Parse("2006-01-02T15:04:05", req.EndTime)
		if err != nil {
			http.Error(w, "invalid end_time", http.StatusBadRequest)
			return
		}

		shift := model.Shift{
			ID:         shiftID,
			TelegramID: telegramID,
			StartTime:  startTime,
			EndTime:    endTime,
			Break:      time.Duration(req.BreakMinutes) * time.Minute,
			HourlyRate: req.HourlyRate,
		}

		if err := shiftService.UpdateShift(
			r.Context(),
			shiftID,
			shift,
		); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(shift); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func createShiftHandler(shiftService *service.ShiftService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createShiftRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		startTime, err := time.Parse("2006-01-02T15:04:05", req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time", http.StatusBadRequest)
			return
		}

		endTime, err := time.Parse("2006-01-02T15:04:05", req.EndTime)
		if err != nil {
			http.Error(w, "invalid end_time", http.StatusBadRequest)
			return
		}

		shift := model.Shift{
			TelegramID: req.TelegramID,
			StartTime:  startTime,
			EndTime:    endTime,
			Break:      time.Duration(req.BreakMinutes) * time.Minute,
			HourlyRate: req.HourlyRate,
		}

		id, err := shiftService.CreateShift(
			r.Context(),
			shift,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		json.NewEncoder(w).Encode(map[string]int{
			"id": id,
		})
	}
}
