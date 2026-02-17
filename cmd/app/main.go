package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/health", healthHandler)
	http.ListenAndServe("0.0.0.0:8080", nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("{\"status\":\"ok\"}"))
}
