package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuth(t *testing.T) {
	const apiKey = "test-secret-key"

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "Правильный ключ",
			header:     "Bearer test-secret-key",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "Неправильный ключ",
			header:     "Bearer wrong-key",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
		{
			name:       "Ключ отсутствует",
			header:     "",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false

			next := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					called = true
					w.WriteHeader(http.StatusOK)
				},
			)

			handler := apiKeyAuth(next, apiKey)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/shifts",
				nil,
			)
			req.Header.Set("Authorization", tt.header)

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf(
					"ожидался статус %d, получен %d",
					tt.wantStatus,
					rec.Code,
				)
			}

			if called != tt.wantCalled {
				t.Errorf(
					"вызов обработчика: ожидалось %t, получено %t",
					tt.wantCalled,
					called,
				)
			}
		})
	}
}
