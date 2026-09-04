package config

import (
	"os"

	"github.com/joho/godotenv"
)

type (
	Container struct {
		App  *App
		HTTP *HTTP
		DB   *DB
		JWT  *JWT
	}

	App struct {
		Name string
		Env  string
	}

	HTTP struct {
		Host           string
		Port           string
		AllowedOrigins string
	}

	DB struct {
		Host     string
		Port     string
		Name     string
		User     string
		Password string
	}

	JWT struct {
		AccessTokenSecret    string
		RefreshTokenSecret   string
		AccessTokenDuration  string
		RefreshTokenDuration string
	}
)

func New() (*Container, error) {
	_ = godotenv.Load()

	App := &App{
		Name: os.Getenv("APP_NAME"),
		Env:  os.Getenv("APP_ENV"),
	}

	HTTP := &HTTP{
		Host:           os.Getenv("HTTP_HOST"),
		Port:           os.Getenv("HTTP_PORT"),
		AllowedOrigins: os.Getenv("HTTP_ALLOWED_ORIGINS"),
	}

	DB := &DB{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Name:     os.Getenv("DB_NAME"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
	}

	JWT := &JWT{
		AccessTokenSecret:    os.Getenv("ACCESS_TOKEN_SECRET"),
		RefreshTokenSecret:   os.Getenv("REFRESH_TOKEN_SECRET"),
		AccessTokenDuration:  os.Getenv("ACCESS_TOKEN_DURATION"),
		RefreshTokenDuration: os.Getenv("REFRESH_TOKEN_DURATION"),
	}

	return &Container{
		App:  App,
		HTTP: HTTP,
		DB:   DB,
		JWT:  JWT,
	}, nil
}
