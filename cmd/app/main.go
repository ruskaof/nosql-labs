package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"nosql-labs/cmd/internal/config"
	"nosql-labs/cmd/internal/db"
	"nosql-labs/cmd/internal/db/event"
	"nosql-labs/cmd/internal/db/session"
	"nosql-labs/cmd/internal/db/user"
	"nosql-labs/cmd/internal/handler"
	"nosql-labs/cmd/internal/reaction"
	"strconv"
	"strings"
	"time"

	"github.com/gocql/gocql"
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
	cassandraSession, err := newCassandraSession(cfg, false)
	if err != nil {
		log.Fatalf("Cassandra connect: %v", err)
	}
	if err := cassandraSession.Query(
		fmt.Sprintf(
			"CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}",
			cfg.CassandraKeyspace,
		),
	).WithContext(ctx).Exec(); err != nil {
		log.Fatalf("Cassandra keyspace init: %v", err)
	}
	cassandraSession.Close()

	cassandraSession, err = newCassandraSession(cfg, true)
	if err != nil {
		log.Fatalf("Cassandra connect with keyspace: %v", err)
	}
	defer cassandraSession.Close()

	reactionStore := reaction.NewCassandraStore(cassandraSession, cfg.CassandraKeyspace)
	if err := reactionStore.InitSchema(ctx); err != nil {
		log.Fatalf("Cassandra schema init: %v", err)
	}
	reactionCache := reaction.NewCache(rdb)
	reactionService := reaction.NewService(
		reactionStore,
		reactionCache,
		time.Duration(cfg.AppLikeTTL)*time.Second,
		eventStore,
	)

	h := handler.NewHttpHandler(cfg, store, userStore, eventStore, reactionService)
	http.HandleFunc("/health", h.HealthHandler)
	http.HandleFunc("/session", h.SessionHandler)
	http.HandleFunc("/users", h.WithPostSessionRefresh(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListUsers(w, r)
		case http.MethodPost:
			h.CreateUser(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/users/")
		if path == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.HasSuffix(path, "/events") {
			id := strings.TrimSuffix(path, "/events")
			id = strings.TrimSuffix(id, "/")
			h.ListEventsByUserID(w, r, id)
			return
		}
		h.GetUserByID(w, r, strings.TrimSuffix(path, "/"))
	})
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
	http.HandleFunc("/events/", h.WithPostSessionRefresh(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/events/")
		path = strings.TrimSuffix(path, "/")
		if path == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.HasSuffix(path, "/like") {
			id := strings.TrimSuffix(path, "/like")
			id = strings.TrimSuffix(id, "/")
			h.PutEventLike(w, r, id)
			return
		}
		if strings.HasSuffix(path, "/dislike") {
			id := strings.TrimSuffix(path, "/dislike")
			id = strings.TrimSuffix(id, "/")
			h.PutEventDislike(w, r, id)
			return
		}
		id := path
		switch r.Method {
		case http.MethodGet:
			h.GetEventByID(w, r, id)
		case http.MethodPatch:
			h.PatchEventByID(w, r, id)
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

func newCassandraSession(cfg *config.ApplicationConfig, withKeyspace bool) (*gocql.Session, error) {
	cluster := gocql.NewCluster(cfg.CassandraHosts...)
	cluster.Port = cfg.CassandraPort
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cfg.CassandraUsername,
		Password: cfg.CassandraPassword,
	}
	consistency := gocql.One
	switch strings.ToUpper(cfg.CassandraConsistency) {
	case "ANY":
		consistency = gocql.Any
	case "ONE":
		consistency = gocql.One
	case "TWO":
		consistency = gocql.Two
	case "THREE":
		consistency = gocql.Three
	case "QUORUM":
		consistency = gocql.Quorum
	case "ALL":
		consistency = gocql.All
	case "LOCAL_QUORUM":
		consistency = gocql.LocalQuorum
	}
	cluster.Consistency = consistency
	if withKeyspace {
		cluster.Keyspace = cfg.CassandraKeyspace
	}
	return cluster.CreateSession()
}
