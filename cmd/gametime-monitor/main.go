package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/matthew-burke/gametime-monitor/pkg/calendar"
	"github.com/matthew-burke/gametime-monitor/pkg/config"
	"github.com/matthew-burke/gametime-monitor/pkg/database"
	"github.com/matthew-burke/gametime-monitor/pkg/model"
	"github.com/matthew-burke/gametime-monitor/pkg/monitor"
	"github.com/matthew-burke/gametime-monitor/pkg/scraper"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		cmdServe()
	case "run":
		cmdRun()
	case "migrate":
		cmdMigrate()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: gametime-monitor <command>

Commands:
  serve       Start the scheduler (checks every hour on Tue/Wed)
  run         Run a single check and exit
  migrate     Manage database migrations

Migrate subcommands:
  migrate up        Apply all pending migrations
  migrate down      Revert all migrations
  migrate version   Show current migration version
`)
}

func cmdServe() {
	cfg := config.Load()
	requireCredentials(cfg)

	mon, cleanup := initMonitor(cfg)
	defer cleanup()

	log.Printf("Starting gametime-monitor (checking every hour on Tue/Wed)")
	log.Printf("Team: %s | Sheet: %s", cfg.TeamName, cfg.SheetName)

	now := time.Now()
	if now.Weekday() == time.Tuesday || now.Weekday() == time.Wednesday {
		if err := mon.Run(context.Background(), cfg.ScheduleURL); err != nil {
			log.Printf("Error during run: %v", err)
		}
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		if now.Weekday() == time.Tuesday || now.Weekday() == time.Wednesday {
			if err := mon.Run(context.Background(), cfg.ScheduleURL); err != nil {
				log.Printf("Error during run: %v", err)
			}
		}
	}
}

func cmdRun() {
	cfg := config.Load()
	requireCredentials(cfg)

	mon, cleanup := initMonitor(cfg)
	defer cleanup()

	if err := mon.Run(context.Background(), cfg.ScheduleURL); err != nil {
		log.Fatalf("Run failed: %v", err)
	}
}

func cmdMigrate() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: gametime-monitor migrate <up|down|version>\n")
		os.Exit(1)
	}

	cfg := config.Load()
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	sqlDB := db.DB

	switch os.Args[2] {
	case "up":
		if err := database.MigrateUp(sqlDB); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("Migrations applied successfully.")
	case "down":
		if err := database.MigrateDown(sqlDB); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Println("Migration reverted one version.")
	case "version":
		version, dirty, err := database.MigrateVersion(sqlDB)
		if err != nil {
			log.Fatalf("Failed to get migration version: %v", err)
		}
		fmt.Printf("Version: %d, Dirty: %v\n", version, dirty)
	default:
		fmt.Fprintf(os.Stderr, "Unknown migrate command: %s\nUsage: gametime-monitor migrate <up|down|version>\n", os.Args[2])
		os.Exit(1)
	}
}

func requireCredentials(cfg config.Config) {
	if _, err := os.Stat(cfg.CredentialsPath); err != nil {
		log.Fatalf("Google credentials not found at %s: set GOOGLE_APPLICATION_CREDENTIALS or mount as a Docker secret", cfg.CredentialsPath)
	}
}

func initMonitor(cfg config.Config) (*monitor.Monitor, func()) {
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	repo := database.NewRepository(db)

	cal, err := calendar.New(cfg.CredentialsPath, cfg.CalendarID)
	if err != nil {
		log.Fatalf("Failed to initialize Google Calendar: %v", err)
	}

	mon := &monitor.Monitor{
		Scraper:   &scraperAdapter{},
		Repo:      repo,
		Calendar:  cal,
		TeamName:  cfg.TeamName,
		SheetName: cfg.SheetName,
		Year:      cfg.Year,
	}

	return mon, func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}

// scraperAdapter wraps the scraper package functions to satisfy the monitor.ScheduleScraper interface.
type scraperAdapter struct{}

func (s *scraperAdapter) ScrapeScheduleLinks(pageURL string, year int) ([]model.ScheduleLink, error) {
	return scraper.ScrapeScheduleLinks(pageURL, year)
}

func (s *scraperAdapter) FetchCSV(spreadsheetID, sheetName string) ([][]string, error) {
	return scraper.FetchCSV(spreadsheetID, sheetName)
}
