package main

import (
	"context"
	"log"
	"net/http"
	"nosql-labs/cmd/internal/config"
	"nosql-labs/cmd/internal/db"
	"nosql-labs/cmd/internal/db/event"
	"nosql-labs/cmd/internal/db/session"
	"nosql-labs/cmd/internal/db/user"
	"nosql-labs/cmd/internal/handler"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI()))
	if err != nil {
		log.Fatalf("MongoDB connect: %v", err)
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping: %v", err)
	}

	database := mongoClient.Database(cfg.MongoDatabase)
	if err := db.EnsureIndexes(ctx, database); err != nil {
		log.Fatalf("MongoDB indexes: %v", err)
	}

	userStore := user.NewStore(database)
	eventStore := event.NewStore(database)

	h := handler.NewHttpHandler(cfg, store, userStore, eventStore)
	http.HandleFunc("/health", h.HealthHandler)
	http.HandleFunc("/session", h.SessionHandler)
	http.HandleFunc("/users", h.WithPostSessionRefresh(h.CreateUser))
	http.HandleFunc("/auth/login", h.WithPostSessionRefresh(h.Login))
	http.HandleFunc("/auth/logout", h.Logout)
	http.HandleFunc("/events", h.WithPostSessionRefresh(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListEvents(w, r)
		case http.MethodPost:
			h.CreateEvent(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	addr := cfg.Host + ":" + strconv.Itoa(cfg.Port)
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %s", err.Error())
	}
}
