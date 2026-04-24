package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var cassandraIdentifierPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,47}$`)

type ApplicationConfig struct {
	Port                 int
	Host                 string
	AppUserSessionTTL    int
	RedisHost            string
	RedisPort            int
	RedisPassword        string
	RedisDB              int
	MongoDatabase        string
	MongoHost            string
	MongoPort            int
	MongoUser            string
	MongoPassword        string
	MongoAuthSource      string
	AppLikeTTL           int
	CassandraHosts       []string
	CassandraPort        int
	CassandraUsername    string
	CassandraPassword    string
	CassandraKeyspace    string
	CassandraConsistency string
}

func InitConfig() (*ApplicationConfig, error) {
	appPortStr, portPresent := os.LookupEnv("APP_PORT")
	if !portPresent {
		return nil, errors.New("Env variable APP_PORT is not present")
	}
	appPort, err := strconv.Atoi(appPortStr)
	if err != nil {
		return nil, err
	}

	appHost, hostPresent := os.LookupEnv("APP_HOST")
	if !hostPresent {
		return nil, errors.New("Env variable APP_HOST is not present")
	}

	appUserSessionTTLStr, ttlPresent := os.LookupEnv("APP_USER_SESSION_TTL")
	if !ttlPresent {
		return nil, errors.New("Env variable APP_USER_SESSION_TTL is not present")
	}
	appUserSessionTTL, err := strconv.Atoi(appUserSessionTTLStr)
	if err != nil {
		return nil, err
	}
	if appUserSessionTTL <= 0 {
		return nil, errors.New("Env variable APP_USER_SESSION_TTL must be greater than 0")
	}
	appLikeTTL := 60
	if s := os.Getenv("APP_LIKE_TTL"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, errors.New("Env variable APP_LIKE_TTL must be an integer")
		}
		if v <= 0 {
			return nil, errors.New("Env variable APP_LIKE_TTL must be greater than 0")
		}
		appLikeTTL = v
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := 6379
	if s := os.Getenv("REDIS_PORT"); s != "" {
		if p, err := strconv.Atoi(s); err == nil {
			redisPort = p
		}
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0
	if s := os.Getenv("REDIS_DB"); s != "" {
		if d, err := strconv.Atoi(s); err == nil {
			redisDB = d
		}
	}

	mongoDatabase := os.Getenv("MONGODB_DATABSE")
	if mongoDatabase == "" {
		mongoDatabase = os.Getenv("MONGODB_DATABASE")
	}
	if mongoDatabase == "" {
		return nil, errors.New("Env variable MONGODB_DATABSE (or MONGODB_DATABASE) is not present")
	}
	mongoHost := os.Getenv("MONGODB_HOST")
	if mongoHost == "" {
		mongoHost = "localhost"
	}
	mongoPort := 27017
	if s := os.Getenv("MONGODB_PORT"); s != "" {
		if p, err := strconv.Atoi(s); err == nil {
			mongoPort = p
		}
	}
	mongoUser := os.Getenv("MONGODB_USER")
	mongoPassword := os.Getenv("MONGODB_PASSWORD")
	mongoAuthSource := os.Getenv("MONGODB_AUTH_SOURCE")
	if mongoAuthSource == "" {
		mongoAuthSource = "admin"
	}

	cassandraHostsRaw, cassandraHostsPresent := os.LookupEnv("CASSANDRA_HOSTS")
	if !cassandraHostsPresent || strings.TrimSpace(cassandraHostsRaw) == "" {
		return nil, errors.New("Env variable CASSANDRA_HOSTS is not present")
	}
	cassandraHosts := make([]string, 0)
	for _, host := range strings.Split(cassandraHostsRaw, ",") {
		host = strings.TrimSpace(host)
		if host != "" {
			cassandraHosts = append(cassandraHosts, host)
		}
	}
	if len(cassandraHosts) == 0 {
		return nil, errors.New("Env variable CASSANDRA_HOSTS must contain at least one host")
	}

	cassandraPortStr, cassandraPortPresent := os.LookupEnv("CASSANDRA_PORT")
	if !cassandraPortPresent {
		return nil, errors.New("Env variable CASSANDRA_PORT is not present")
	}
	cassandraPort, err := strconv.Atoi(cassandraPortStr)
	if err != nil {
		return nil, errors.New("Env variable CASSANDRA_PORT must be an integer")
	}
	cassandraUsername := os.Getenv("CASSANDRA_USERNAME")
	cassandraPassword := os.Getenv("CASSANDRA_PASSWORD")

	cassandraKeyspace := os.Getenv("CASSANDRA_KEYSPACE")
	if strings.TrimSpace(cassandraKeyspace) == "" {
		return nil, errors.New("Env variable CASSANDRA_KEYSPACE is not present")
	}
	if !cassandraIdentifierPattern.MatchString(cassandraKeyspace) {
		return nil, errors.New("Env variable CASSANDRA_KEYSPACE must match ^[A-Za-z][A-Za-z0-9_]{0,47}$")
	}
	cassandraConsistency := os.Getenv("CASSANDRA_CONSISTENCY")
	if strings.TrimSpace(cassandraConsistency) == "" {
		cassandraConsistency = "ONE"
	}

	return &ApplicationConfig{
		Port:                 appPort,
		Host:                 appHost,
		AppUserSessionTTL:    appUserSessionTTL,
		RedisHost:            redisHost,
		RedisPort:            redisPort,
		RedisPassword:        redisPassword,
		RedisDB:              redisDB,
		MongoDatabase:        mongoDatabase,
		MongoHost:            mongoHost,
		MongoPort:            mongoPort,
		MongoUser:            mongoUser,
		MongoPassword:        mongoPassword,
		MongoAuthSource:      mongoAuthSource,
		AppLikeTTL:           appLikeTTL,
		CassandraHosts:       cassandraHosts,
		CassandraPort:        cassandraPort,
		CassandraUsername:    cassandraUsername,
		CassandraPassword:    cassandraPassword,
		CassandraKeyspace:    cassandraKeyspace,
		CassandraConsistency: cassandraConsistency,
	}, nil
}

func (c *ApplicationConfig) MongoURI() string {
	host := fmt.Sprintf("%s:%d", c.MongoHost, c.MongoPort)
	db := c.MongoDatabase
	if c.MongoUser == "" && c.MongoPassword == "" {
		return "mongodb://" + host + "/" + db
	}
	user := url.UserPassword(c.MongoUser, c.MongoPassword)
	return fmt.Sprintf("mongodb://%s@%s/%s?authSource=%s", user.String(), host, url.PathEscape(db), url.QueryEscape(c.MongoAuthSource))
}
