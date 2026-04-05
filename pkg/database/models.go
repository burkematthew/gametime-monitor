package database

import (
	"time"

	"github.com/uptrace/bun"
)

type LocationRecord struct {
	bun.BaseModel `bun:"table:locations"`

	ID      int64  `bun:"id,pk,autoincrement"`
	Name    string `bun:"name,notnull,unique"`
	Address string `bun:"address,notnull"`
}

type GameRecord struct {
	bun.BaseModel `bun:"table:games"`

	ID             int64     `bun:"id,pk,autoincrement"`
	SpreadsheetID  string    `bun:"spreadsheet_id,notnull"`
	TournamentDate time.Time `bun:"tournament_date,notnull,type:date"`
	HomeTeam       string    `bun:"home_team,notnull"`
	AwayTeam       string    `bun:"away_team,notnull"`
	GameTime       time.Time `bun:"game_time,notnull"`
	LocationID     int64     `bun:"location_id,notnull"`
	FieldNumber    string    `bun:"field_number"`
	HomeScore      *int      `bun:"home_score"`
	AwayScore      *int      `bun:"away_score"`
	CreatedAt      time.Time `bun:"created_at"`
	UpdatedAt      time.Time `bun:"updated_at"`
}

type ProcessedSchedule struct {
	bun.BaseModel `bun:"table:processed_schedules"`

	SpreadsheetID string    `bun:"spreadsheet_id,pk"`
	ProcessedAt   time.Time `bun:"processed_at,default:current_timestamp"`
}
