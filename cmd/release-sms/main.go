package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"example.com/release-sms-registry/internal/infrai"
	"example.com/release-sms-registry/internal/releases"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := infrai.NewClient(key)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /release/sms-assets", func(w http.ResponseWriter, r *http.Request) {
		var input releases.ReleaseInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if err := releases.Validate(input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		result, err := releases.Register(r.Context(), client, input)
		if err != nil {
			status := http.StatusBadGateway
			var apiErr *infrai.APIError
			if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
				status = apiErr.HTTPStatus
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, result)
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("release SMS registry listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
