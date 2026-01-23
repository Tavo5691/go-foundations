package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTKey      string
	Port        string
}

func Load() *Config {
	_ = godotenv.Load()

	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatal("JWT_KEY environment variable is required")
	}

	port := os.Getenv("PORT")

	return &Config{DatabaseURL: databaseUrl, JWTKey: jwtKey, Port: port}
}
