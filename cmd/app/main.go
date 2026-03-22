package main

import (
	"log"
	"net/http"
	"nosql-labs/cmd/internal/config"
	"nosql-labs/cmd/internal/handler"
	"nosql-labs/cmd/internal/session"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("Could not init application configuration: %s", err.Error())
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + strconv.Itoa(cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	store := session.NewRedisStore(rdb)

	h := handler.NewHttpHandler(cfg, store)
	http.HandleFunc("/health", h.HealthHandler)
	http.HandleFunc("/session", h.SessionHandler)

	addr := cfg.Host + ":" + strconv.Itoa(cfg.Port)
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %s", err.Error())
	}
}
