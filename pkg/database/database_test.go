package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/matthew-burke/gametime-monitor/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) *testDB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("gametime_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := New(connStr)
	require.NoError(t, err)

	err = MigrateUp(db.DB)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close())
		assert.NoError(t, pgContainer.Terminate(ctx))
	})

	return &testDB{db: db, repo: NewRepository(db)}
}

type testDB struct {
	db   interface{ Close() error }
	repo *Repository
}

func TestNew_Success(t *testing.T) {
	tdb := setupTestDB(t)
	assert.NotNil(t, tdb.repo)
}

func TestNew_BadURL(t *testing.T) {
	_, err := New("postgres://bad:bad@127.0.0.1:1/bad?sslmode=disable&connect_timeout=1")
	assert.Error(t, err)
}

func TestIsScheduleProcessed_NotProcessed(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	processed, err := tdb.repo.IsScheduleProcessed(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, processed)
}

func TestMarkScheduleProcessed(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	err := tdb.repo.MarkScheduleProcessed(ctx, "sheet123")
	require.NoError(t, err)

	processed, err := tdb.repo.IsScheduleProcessed(ctx, "sheet123")
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestMarkScheduleProcessed_Idempotent(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	require.NoError(t, tdb.repo.MarkScheduleProcessed(ctx, "sheet123"))
	require.NoError(t, tdb.repo.MarkScheduleProcessed(ctx, "sheet123"))

	processed, err := tdb.repo.IsScheduleProcessed(ctx, "sheet123")
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestFindLocationByName(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	loc, err := tdb.repo.FindLocationByName(ctx, "Fenton")
	require.NoError(t, err)
	assert.Equal(t, "Fenton", loc.Name)
	assert.Equal(t, "945 Larkin Williams Rd, Fenton, MO 63026", loc.Address)
}

func TestFindLocationByName_NotFound(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	_, err := tdb.repo.FindLocationByName(ctx, "Nonexistent")
	assert.Error(t, err)
}

func TestUpsertGame_Insert(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/Chicago")
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)

	game := model.Game{
		HomeTeam: "Stl Bears Bell",
		AwayTeam: "Gamers Blue",
		DateTime: time.Date(2026, 4, 4, 9, 0, 0, 0, loc),
		Location: "Fenton", FieldNumber: "2",
	}

	err := tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate)
	require.NoError(t, err)

	// Verify it was inserted with correct location_id
	var count int
	err = tdb.repo.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM games WHERE home_team = 'Stl Bears Bell'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var locationName string
	err = tdb.repo.db.QueryRowContext(ctx,
		"SELECT l.name FROM games g JOIN locations l ON g.location_id = l.id WHERE g.home_team = 'Stl Bears Bell'").Scan(&locationName)
	require.NoError(t, err)
	assert.Equal(t, "Fenton", locationName)
}

func TestUpsertGame_UnknownLocation(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/Chicago")
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)

	game := model.Game{
		HomeTeam: "Stl Bears Bell",
		AwayTeam: "Gamers Blue",
		DateTime: time.Date(2026, 4, 4, 9, 0, 0, 0, loc),
		Location: "Unknown Park", FieldNumber: "1",
	}

	err := tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate)
	assert.Error(t, err)
}

func TestUpsertGame_UpdateOnConflict(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/Chicago")
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)

	game := model.Game{
		HomeTeam: "Stl Bears Bell",
		AwayTeam: "Gamers Blue",
		DateTime: time.Date(2026, 4, 4, 9, 0, 0, 0, loc),
		Location: "Fenton", FieldNumber: "2",
	}
	require.NoError(t, tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate))

	// Update the game time and location
	game.DateTime = time.Date(2026, 4, 4, 10, 30, 0, 0, loc)
	game.Location = "Woodlands"
	game.FieldNumber = "4"
	require.NoError(t, tdb.repo.UpsertGame(ctx, game, "sheet456", tournamentDate))

	// Verify only one row and it's updated
	var count int
	err := tdb.repo.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM games WHERE home_team = 'Stl Bears Bell' AND away_team = 'Gamers Blue'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var locationName, fieldNumber, spreadsheetID string
	err = tdb.repo.db.QueryRowContext(ctx,
		"SELECT l.name, g.field_number, g.spreadsheet_id FROM games g JOIN locations l ON g.location_id = l.id WHERE g.home_team = 'Stl Bears Bell' AND g.away_team = 'Gamers Blue'").
		Scan(&locationName, &fieldNumber, &spreadsheetID)
	require.NoError(t, err)
	assert.Equal(t, "Woodlands", locationName)
	assert.Equal(t, "4", fieldNumber)
	assert.Equal(t, "sheet456", spreadsheetID)
}

func TestUpsertGame_WithScores(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/Chicago")
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)

	homeScore := 5
	awayScore := 3
	game := model.Game{
		HomeTeam:  "Stl Bears Bell",
		AwayTeam:  "Gamers Blue",
		DateTime:  time.Date(2026, 4, 4, 9, 0, 0, 0, loc),
		Location: "Fenton", FieldNumber: "2",
		HomeScore: &homeScore,
		AwayScore: &awayScore,
	}

	require.NoError(t, tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate))

	var hs, as int
	err := tdb.repo.db.QueryRowContext(ctx,
		"SELECT home_score, away_score FROM games WHERE home_team = 'Stl Bears Bell'").
		Scan(&hs, &as)
	require.NoError(t, err)
	assert.Equal(t, 5, hs)
	assert.Equal(t, 3, as)
}

func TestUpsertGame_UpdateScores(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/Chicago")
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)

	// Insert without scores
	game := model.Game{
		HomeTeam: "Stl Bears Bell",
		AwayTeam: "Gamers Blue",
		DateTime: time.Date(2026, 4, 4, 9, 0, 0, 0, loc),
		Location: "Fenton", FieldNumber: "2",
	}
	require.NoError(t, tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate))

	// Update with scores
	homeScore := 7
	awayScore := 2
	game.HomeScore = &homeScore
	game.AwayScore = &awayScore
	require.NoError(t, tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate))

	var hs, as int
	err := tdb.repo.db.QueryRowContext(ctx,
		"SELECT home_score, away_score FROM games WHERE home_team = 'Stl Bears Bell'").
		Scan(&hs, &as)
	require.NoError(t, err)
	assert.Equal(t, 7, hs)
	assert.Equal(t, 2, as)
}

func TestTriggers_CreatedAt(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/Chicago")
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)

	game := model.Game{
		HomeTeam: "Stl Bears Bell",
		AwayTeam: "Gamers Blue",
		DateTime: time.Date(2026, 4, 4, 9, 0, 0, 0, loc),
		Location: "Fenton", FieldNumber: "2",
	}
	before := time.Now().Add(-1 * time.Second)
	require.NoError(t, tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate))

	var createdAt time.Time
	err := tdb.repo.db.QueryRowContext(ctx,
		"SELECT created_at FROM games WHERE home_team = 'Stl Bears Bell'").Scan(&createdAt)
	require.NoError(t, err)
	assert.True(t, createdAt.After(before), "created_at should be set by trigger")
}

func TestTriggers_UpdatedAt(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/Chicago")
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)

	game := model.Game{
		HomeTeam: "Stl Bears Bell",
		AwayTeam: "Gamers Blue",
		DateTime: time.Date(2026, 4, 4, 9, 0, 0, 0, loc),
		Location: "Fenton", FieldNumber: "2",
	}
	require.NoError(t, tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate))

	var createdAt time.Time
	err := tdb.repo.db.QueryRowContext(ctx,
		"SELECT created_at FROM games WHERE home_team = 'Stl Bears Bell'").Scan(&createdAt)
	require.NoError(t, err)

	// Small delay to ensure updated_at differs
	time.Sleep(10 * time.Millisecond)

	// Trigger an update
	game.Location = "Woodlands"
	game.FieldNumber = "9"
	require.NoError(t, tdb.repo.UpsertGame(ctx, game, "sheet123", tournamentDate))

	var updatedAt time.Time
	err = tdb.repo.db.QueryRowContext(ctx,
		"SELECT updated_at FROM games WHERE home_team = 'Stl Bears Bell'").Scan(&updatedAt)
	require.NoError(t, err)
	assert.True(t, updatedAt.After(createdAt),
		fmt.Sprintf("updated_at (%v) should be after created_at (%v)", updatedAt, createdAt))
}

func TestNewRepository(t *testing.T) {
	tdb := setupTestDB(t)
	assert.NotNil(t, tdb.repo)
}

func TestMigrateUp_BadConnection(t *testing.T) {
	// sql.DB that can't actually execute queries
	db, err := New("postgres://bad:bad@127.0.0.1:1/bad?sslmode=disable&connect_timeout=1")
	if err != nil {
		// Connection failed, which is expected
		assert.Error(t, err)
		return
	}
	defer func() { assert.NoError(t, db.Close()) }()

	err = MigrateUp(db.DB)
	assert.Error(t, err)
}

func TestMigrateUp_Idempotent(t *testing.T) {
	tdb := setupTestDB(t)

	// setupTestDB already ran migrations; run them again to verify idempotency
	err := MigrateUp(tdb.repo.DB().DB)
	assert.NoError(t, err)
}

func TestMigrateDown(t *testing.T) {
	tdb := setupTestDB(t)
	sqlDB := tdb.repo.DB().DB

	// Should revert one version
	err := MigrateDown(sqlDB)
	assert.NoError(t, err)

	version, _, err := MigrateVersion(sqlDB)
	require.NoError(t, err)
	assert.Equal(t, uint(3), version)
}

func TestMigrateVersion(t *testing.T) {
	tdb := setupTestDB(t)
	sqlDB := tdb.repo.DB().DB

	version, dirty, err := MigrateVersion(sqlDB)
	require.NoError(t, err)
	assert.Equal(t, uint(4), version)
	assert.False(t, dirty)
}
