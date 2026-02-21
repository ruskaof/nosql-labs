package config

import (
	"os"
	"strconv"
)

type ApplicationConfig struct {
	Port int
	Host string
}

func InitConfig() (*ApplicationConfig, error) {
	appPortStr, portPresent := os.LookupEnv("APP_PORT")
	if !portPresent {
		return nil, &ApplicationConfigError{message: "Env variable APP_PORT is not present"}
	}
	appPort, err := strconv.Atoi(appPortStr)
	if err != nil {
		return nil, err
	}

	appHost, hostPresent := os.LookupEnv("APP_HOST")
	if !hostPresent {
		return nil, &ApplicationConfigError{message: "Env variable APP_HOST is not present"}
	}

	return &ApplicationConfig{
		Port: appPort,
		Host: appHost,
	}, nil
}

type ApplicationConfigError struct {
	message string
}

func (e *ApplicationConfigError) Error() string {
	return e.message
}
