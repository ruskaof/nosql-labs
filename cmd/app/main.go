package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	appPort, portPresent := os.LookupEnv("APP_PORT")

	if !portPresent {
		log.Fatal("Environment variable APP_PORT is not present")
	}

	http.HandleFunc("/health", healthHandler)
	http.ListenAndServe("0.0.0.0:"+appPort, nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("{\"status\":\"ok\"}"))
}
