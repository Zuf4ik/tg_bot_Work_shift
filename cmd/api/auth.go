package main

import (
	"crypto/subtle"
	"net/http"
)

// apiKeyAuth проверяет API-ключ перед выполнением запроса.
func apiKeyAuth(next http.Handler, apiKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			http.Error(
				w,
				"API key is not configured",
				http.StatusInternalServerError,
			)
			return
		}

		expected := "Bearer " + apiKey
		actual := r.Header.Get("Authorization")

		if subtle.ConstantTimeCompare(
			[]byte(actual),
			[]byte(expected),
		) != 1 {
			http.Error(
				w,
				"Unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
