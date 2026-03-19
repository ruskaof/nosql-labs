package config

import (
	"errors"
	"os"
	"strconv"
)

type ApplicationConfig struct {
	Port              int
	Host              string
	AppUserSessionTTL int
	RedisHost         string
	RedisPort         int
	RedisPassword     string
	RedisDB           int
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

	return &ApplicationConfig{
		Port:              appPort,
		Host:              appHost,
		AppUserSessionTTL: appUserSessionTTL,
		RedisHost:         redisHost,
		RedisPort:         redisPort,
		RedisPassword:     redisPassword,
		RedisDB:           redisDB,
	}, nil
}
