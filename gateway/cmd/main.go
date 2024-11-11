package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gateway/handlers"
	"net/http"
	"os"
)

type HttpResponseStruct struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

func main() {
	handler := handlers.UserHandler()

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Got request: %s %s, Body: %v\n", r.Method, r.RequestURI, r.Body)

		data, _ := handler.GetUsers(context.Background(), "test-1")
		jsonData, err := json.Marshal(HttpResponseStruct{Success: true, Data: data})

		if err != nil {
			http.Error(w, "Error converting data to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	})

	err := http.ListenAndServe(":5000", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
