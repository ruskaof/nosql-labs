package main

import (
	"log"
	"net/http"
	"nosql-labs/cmd/internal/config"
	"strconv"
)

func main() {
	config, err := config.InitConfig()

	if err != nil {
		log.Fatalf("Could not init application configuration %s", err.Error())
	}

	http.HandleFunc("/health", healthHandler)
	http.ListenAndServe(config.Host+":"+strconv.Itoa(config.Port), nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("{\"status\":\"ok\"}"))
}
