package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	CredentialsPath string
	CalendarID      string
	DatabaseURL     string
	TeamName        string
	ScheduleURL     string
	SheetName       string
	Year            int
}

func Load() Config {
	return Config{
		CredentialsPath: GetEnvOrDefault("GOOGLE_APPLICATION_CREDENTIALS", "/run/secrets/gcp_key"),
		CalendarID:      GetEnvOrDefault("CALENDAR_ID", ""),
		DatabaseURL:     BuildDatabaseURL(),
		TeamName:        GetEnvOrDefault("TEAM_NAME", ""),
		ScheduleURL:     GetEnvOrDefault("SCHEDULE_URL", ""),
		SheetName:       GetEnvOrDefault("SHEET_NAME", "10u"),
		Year:            time.Now().Year(),
	}
}

func GetEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ReadSecret(name string) string {
	return ReadSecretFrom("/run/secrets", name)
}

func ReadSecretFrom(dir, name string) string {
	path := fmt.Sprintf("%s/%s", dir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func GetSecretOrEnv(secretName, envKey, fallback string) string {
	if v := ReadSecret(secretName); v != "" {
		return v
	}
	return GetEnvOrDefault(envKey, fallback)
}

func BuildDatabaseURL() string {
	dbHost := GetEnvOrDefault("DB_HOST", "localhost")
	dbPort := GetEnvOrDefault("DB_PORT", "5432")
	dbUser := GetEnvOrDefault("DB_USER", "gametime")
	dbPassword := GetSecretOrEnv("db_password", "DB_PASSWORD", "gametime")
	dbName := GetEnvOrDefault("DB_NAME", "gametime")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
}
