package bootstrap

import (
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	DB  Database
	SMS SMS
	JWT JWT
	S3  S3
}

type S3 struct {
	Bucket, Region, Endpoint string
	UsePathStyle             bool
}

type JWT struct{ Secret, Issuer, Audience, AccessTokenTTL string }

type SMS struct{ Token string }

type Database struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     string
}

func NewEnv() *Env {
	godotenv.Load(".env")
	return &Env{
		S3:  S3{Bucket: os.Getenv("S3_BUCKET"), Region: envOrDefault("AWS_REGION", "us-east-1"), Endpoint: os.Getenv("S3_ENDPOINT"), UsePathStyle: os.Getenv("S3_USE_PATH_STYLE") == "true"},
		JWT: JWT{Secret: os.Getenv("JWT_SECRET"), Issuer: envOrDefault("JWT_ISSUER", "zedlancer"), Audience: envOrDefault("JWT_AUDIENCE", "zedlancer-api"), AccessTokenTTL: envOrDefault("JWT_ACCESS_TOKEN_TTL", "1h")},
		SMS: SMS{Token: os.Getenv("SMS_TOKEN")},
		DB: Database{
			Host:     os.Getenv("DB_HOST"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			Port:     os.Getenv("DB_PORT"),
		},
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
