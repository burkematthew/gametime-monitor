package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnvOrDefault_EnvSet(t *testing.T) {
	t.Setenv("TEST_CONFIG_VAR", "from_env")
	assert.Equal(t, "from_env", GetEnvOrDefault("TEST_CONFIG_VAR", "default"))
}

func TestGetEnvOrDefault_EnvNotSet(t *testing.T) {
	assert.Equal(t, "default", GetEnvOrDefault("TEST_CONFIG_UNSET_VAR", "default"))
}

func TestGetEnvOrDefault_EnvEmpty(t *testing.T) {
	t.Setenv("TEST_CONFIG_EMPTY", "")
	assert.Equal(t, "fallback", GetEnvOrDefault("TEST_CONFIG_EMPTY", "fallback"))
}

func TestReadSecretFrom_FileExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "my_secret"), []byte("  secret_value  \n"), 0600))

	assert.Equal(t, "secret_value", ReadSecretFrom(dir, "my_secret"))
}

func TestReadSecretFrom_FileNotFound(t *testing.T) {
	assert.Equal(t, "", ReadSecretFrom("/nonexistent", "missing"))
}

func TestReadSecretFrom_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "padded"), []byte("\n  trimmed  \n"), 0600))

	assert.Equal(t, "trimmed", ReadSecretFrom(dir, "padded"))
}

func TestGetSecretOrEnv_SecretMissing_UsesEnv(t *testing.T) {
	t.Setenv("TEST_SECRET_ENV", "env_value")
	assert.Equal(t, "env_value", GetSecretOrEnv("nonexistent_secret", "TEST_SECRET_ENV", "default"))
}

func TestGetSecretOrEnv_BothMissing_UsesFallback(t *testing.T) {
	assert.Equal(t, "fallback", GetSecretOrEnv("nonexistent", "TEST_SECRET_UNSET", "fallback"))
}

func TestBuildDatabaseURL_Defaults(t *testing.T) {
	// Clear any env vars that might be set
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")

	url := BuildDatabaseURL()
	assert.Equal(t, "postgres://gametime:gametime@localhost:5432/gametime?sslmode=disable", url)
}

func TestBuildDatabaseURL_CustomValues(t *testing.T) {
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "admin")
	t.Setenv("DB_PASSWORD", "s3cret")
	t.Setenv("DB_NAME", "mydb")

	url := BuildDatabaseURL()
	assert.Equal(t, "postgres://admin:s3cret@db.example.com:5433/mydb?sslmode=disable", url)
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("CALENDAR_ID", "")
	t.Setenv("TEAM_NAME", "")
	t.Setenv("SCHEDULE_URL", "")
	t.Setenv("SHEET_NAME", "")

	cfg := Load()
	assert.Equal(t, "/run/secrets/gcp_key", cfg.CredentialsPath)
	assert.Equal(t, "", cfg.CalendarID)
	assert.Equal(t, "", cfg.TeamName)
	assert.Equal(t, "", cfg.ScheduleURL)
	assert.Equal(t, "10u", cfg.SheetName)
	assert.NotZero(t, cfg.Year)
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/path/to/creds.json")
	t.Setenv("CALENDAR_ID", "cal123")
	t.Setenv("TEAM_NAME", "Bears")
	t.Setenv("SCHEDULE_URL", "https://example.com")
	t.Setenv("SHEET_NAME", "12u")

	cfg := Load()
	assert.Equal(t, "/path/to/creds.json", cfg.CredentialsPath)
	assert.Equal(t, "cal123", cfg.CalendarID)
	assert.Equal(t, "Bears", cfg.TeamName)
	assert.Equal(t, "https://example.com", cfg.ScheduleURL)
	assert.Equal(t, "12u", cfg.SheetName)
}
