package database

import (
	"context"
	"fmt"
	"time"

	"github.com/matthew-burke/gametime-monitor/pkg/model"
	"github.com/uptrace/bun"
)

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *bun.DB {
	return r.db
}

func (r *Repository) IsScheduleProcessed(ctx context.Context, spreadsheetID string) (bool, error) {
	return r.db.NewSelect().
		Model((*ProcessedSchedule)(nil)).
		Where("spreadsheet_id = ?", spreadsheetID).
		Exists(ctx)
}

func (r *Repository) MarkScheduleProcessed(ctx context.Context, spreadsheetID string) error {
	record := &ProcessedSchedule{SpreadsheetID: spreadsheetID}
	_, err := r.db.NewInsert().
		Model(record).
		On("CONFLICT (spreadsheet_id) DO NOTHING").
		Exec(ctx)
	return err
}

func (r *Repository) FindLocationByName(ctx context.Context, name string) (*model.Location, error) {
	rec := new(LocationRecord)
	err := r.db.NewSelect().
		Model(rec).
		Where("name = ?", name).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding location %q: %w", name, err)
	}
	return &model.Location{Name: rec.Name, Address: rec.Address}, nil
}

func (r *Repository) UpsertGame(ctx context.Context, game model.Game, spreadsheetID string, tournamentDate time.Time) error {
	rec := new(LocationRecord)
	err := r.db.NewSelect().
		Model(rec).
		Where("name = ?", game.Location).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("looking up location for game: %w", err)
	}

	record := &GameRecord{
		SpreadsheetID:  spreadsheetID,
		TournamentDate: tournamentDate,
		HomeTeam:       game.HomeTeam,
		AwayTeam:       game.AwayTeam,
		GameTime:       game.DateTime,
		LocationID:     rec.ID,
		FieldNumber:    game.FieldNumber,
		HomeScore:      game.HomeScore,
		AwayScore:      game.AwayScore,
	}
	_, err = r.db.NewInsert().
		Model(record).
		On("CONFLICT (home_team, away_team, tournament_date) DO UPDATE").
		Set("game_time = EXCLUDED.game_time").
		Set("location_id = EXCLUDED.location_id").
		Set("field_number = EXCLUDED.field_number").
		Set("spreadsheet_id = EXCLUDED.spreadsheet_id").
		Set("home_score = EXCLUDED.home_score").
		Set("away_score = EXCLUDED.away_score").
		Exec(ctx)
	return err
}
