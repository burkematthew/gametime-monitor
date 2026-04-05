package monitor

import (
	"context"
	"log"
	"time"

	"github.com/matthew-burke/gametime-monitor/pkg/model"
)

type ScheduleScraper interface {
	ScrapeScheduleLinks(pageURL string, year int) ([]model.ScheduleLink, error)
	FetchCSV(spreadsheetID, sheetName string) ([][]string, error)
}

type GameRepository interface {
	IsScheduleProcessed(ctx context.Context, spreadsheetID string) (bool, error)
	MarkScheduleProcessed(ctx context.Context, spreadsheetID string) error
	FindLocationByName(ctx context.Context, name string) (*model.Location, error)
	UpsertGame(ctx context.Context, game model.Game, spreadsheetID string, tournamentDate time.Time) error
}

type CalendarClient interface {
	CreateGameEvent(game model.Game, teamName string) error
}

type Monitor struct {
	Scraper  ScheduleScraper
	Repo     GameRepository
	Calendar CalendarClient
	TeamName string
	SheetName string
	Year     int
}

func (m *Monitor) Run(ctx context.Context, scheduleURL string) error {
	log.Println("Checking for new schedule links...")

	links, err := m.Scraper.ScrapeScheduleLinks(scheduleURL, m.Year)
	if err != nil {
		log.Printf("Error scraping schedule: %v", err)
		return err
	}

	log.Printf("Found %d schedule link(s)", len(links))

	for _, link := range links {
		if err := m.processLink(ctx, link); err != nil {
			log.Printf("Error processing link %s: %v", link.LinkText, err)
		}
	}

	log.Println("Check complete.")
	return nil
}

func (m *Monitor) processLink(ctx context.Context, link model.ScheduleLink) error {
	processed, err := m.Repo.IsScheduleProcessed(ctx, link.SpreadsheetID)
	if err != nil {
		return err
	}
	if processed {
		log.Printf("Already processed: %s (%s)", link.LinkText, link.SpreadsheetID[:8])
		return nil
	}

	log.Printf("New schedule found: %s (date: %s)", link.LinkText, link.Date.Format("Jan 2"))

	rows, err := m.Scraper.FetchCSV(link.SpreadsheetID, m.SheetName)
	if err != nil {
		return err
	}

	games := model.ParseGames(rows, m.TeamName, link.Date)
	if len(games) == 0 {
		log.Printf("No games found for %s in this schedule", m.TeamName)
		return m.Repo.MarkScheduleProcessed(ctx, link.SpreadsheetID)
	}

	log.Printf("Found %d game(s) for %s:", len(games), m.TeamName)
	for _, g := range games {
		log.Printf("  %s at %s (%s)", g.Summary(m.TeamName), g.DateTime.Format("Mon Jan 2 3:04 PM"), g.Location)
	}

	for i := range games {
		loc, err := m.Repo.FindLocationByName(ctx, games[i].Location)
		if err != nil {
			log.Printf("Error finding location %s: %v", games[i].Location, err)
		} else {
			games[i].Address = loc.Address
		}
		if err := m.Repo.UpsertGame(ctx, games[i], link.SpreadsheetID, link.Date); err != nil {
			log.Printf("Error inserting game: %v", err)
		}
		if err := m.Calendar.CreateGameEvent(games[i], m.TeamName); err != nil {
			log.Printf("Error creating calendar event: %v", err)
		}
	}

	return m.Repo.MarkScheduleProcessed(ctx, link.SpreadsheetID)
}
