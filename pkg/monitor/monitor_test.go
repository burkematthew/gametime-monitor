package monitor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matthew-burke/gametime-monitor/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockScraper struct {
	links    []model.ScheduleLink
	linksErr error
	rows     [][]string
	rowsErr  error
}

func (m *mockScraper) ScrapeScheduleLinks(_ string, _ int) ([]model.ScheduleLink, error) {
	return m.links, m.linksErr
}

func (m *mockScraper) FetchCSV(_, _ string) ([][]string, error) {
	return m.rows, m.rowsErr
}

type mockRepo struct {
	processed       map[string]bool
	upsertedGames   []model.Game
	markedProcessed []string
	locations       map[string]*model.Location
	upsertErr       error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		processed: make(map[string]bool),
		locations: map[string]*model.Location{
			"Fenton":    {Name: "Fenton", Address: "945 Larkin Williams Rd, Fenton, MO 63026"},
			"Woodlands": {Name: "Woodlands", Address: "1 Woodlands Parkway, St Peters, MO 63376"},
		},
	}
}

func (m *mockRepo) IsScheduleProcessed(_ context.Context, id string) (bool, error) {
	return m.processed[id], nil
}

func (m *mockRepo) MarkScheduleProcessed(_ context.Context, id string) error {
	m.markedProcessed = append(m.markedProcessed, id)
	return nil
}

func (m *mockRepo) FindLocationByName(_ context.Context, name string) (*model.Location, error) {
	loc, ok := m.locations[name]
	if !ok {
		return nil, errors.New("location not found")
	}
	return loc, nil
}

func (m *mockRepo) UpsertGame(_ context.Context, game model.Game, _ string, _ time.Time) error {
	m.upsertedGames = append(m.upsertedGames, game)
	return m.upsertErr
}

type mockCalendar struct {
	createdEvents []model.Game
	err           error
}

func (m *mockCalendar) CreateGameEvent(game model.Game, _ string) error {
	m.createdEvents = append(m.createdEvents, game)
	return m.err
}

func cst() *time.Location {
	loc, _ := time.LoadLocation("America/Chicago")
	return loc
}

func testRows() [][]string {
	return [][]string{
		{"", "Stl Bears Bell", "vs", "", "Gamers Blue", "Saturday", "900", "Fenton", "2"},
	}
}

func TestRun_ScrapeError(t *testing.T) {
	m := &Monitor{
		Scraper:  &mockScraper{linksErr: errors.New("network error")},
		Repo:     newMockRepo(),
		Calendar: &mockCalendar{},
		TeamName: "Stl Bears Bell",
		Year:     2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	assert.Error(t, err)
}

func TestRun_NoLinks(t *testing.T) {
	m := &Monitor{
		Scraper:  &mockScraper{},
		Repo:     newMockRepo(),
		Calendar: &mockCalendar{},
		TeamName: "Stl Bears Bell",
		Year:     2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	require.NoError(t, err)
}

func TestRun_SkipsAlreadyProcessed(t *testing.T) {
	repo := newMockRepo()
	repo.processed["sheet123"] = true

	m := &Monitor{
		Scraper: &mockScraper{
			links: []model.ScheduleLink{
				{SpreadsheetID: "sheet123", Date: time.Date(2026, 4, 4, 0, 0, 0, 0, cst()), LinkText: "April 4"},
			},
		},
		Repo:     repo,
		Calendar: &mockCalendar{},
		TeamName: "Stl Bears Bell",
		Year:     2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	require.NoError(t, err)
	assert.Empty(t, repo.upsertedGames)
}

func TestRun_ProcessesNewSchedule(t *testing.T) {
	repo := newMockRepo()
	cal := &mockCalendar{}

	m := &Monitor{
		Scraper: &mockScraper{
			links: []model.ScheduleLink{
				{SpreadsheetID: "sheet123", Date: time.Date(2026, 4, 4, 0, 0, 0, 0, cst()), LinkText: "April 4"},
			},
			rows: testRows(),
		},
		Repo:      repo,
		Calendar:  cal,
		TeamName:  "Stl Bears Bell",
		SheetName: "10u",
		Year:      2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	require.NoError(t, err)

	require.Len(t, repo.upsertedGames, 1)
	assert.Equal(t, "Stl Bears Bell", repo.upsertedGames[0].HomeTeam)
	assert.Equal(t, "Gamers Blue", repo.upsertedGames[0].AwayTeam)
	assert.Equal(t, "945 Larkin Williams Rd, Fenton, MO 63026", repo.upsertedGames[0].Address)

	require.Len(t, cal.createdEvents, 1)
	assert.Equal(t, "945 Larkin Williams Rd, Fenton, MO 63026", cal.createdEvents[0].Address)

	assert.Equal(t, []string{"sheet123"}, repo.markedProcessed)
}

func TestRun_NoGamesMarksProcessed(t *testing.T) {
	repo := newMockRepo()

	m := &Monitor{
		Scraper: &mockScraper{
			links: []model.ScheduleLink{
				{SpreadsheetID: "sheet123", Date: time.Date(2026, 4, 4, 0, 0, 0, 0, cst()), LinkText: "April 4"},
			},
			rows: [][]string{
				{"", "Other Team", "vs", "", "Another Team", "Saturday", "900", "Fenton", "2"},
			},
		},
		Repo:      repo,
		Calendar:  &mockCalendar{},
		TeamName:  "Stl Bears Bell",
		SheetName: "10u",
		Year:      2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, []string{"sheet123"}, repo.markedProcessed)
	assert.Empty(t, repo.upsertedGames)
}

func TestRun_FetchCSVError(t *testing.T) {
	repo := newMockRepo()

	m := &Monitor{
		Scraper: &mockScraper{
			links: []model.ScheduleLink{
				{SpreadsheetID: "sheet123", Date: time.Date(2026, 4, 4, 0, 0, 0, 0, cst()), LinkText: "April 4"},
			},
			rowsErr: errors.New("csv error"),
		},
		Repo:      repo,
		Calendar:  &mockCalendar{},
		TeamName:  "Stl Bears Bell",
		SheetName: "10u",
		Year:      2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	require.NoError(t, err) // Run doesn't return errors from individual links
	assert.Empty(t, repo.markedProcessed)
}

func TestRun_UpsertError_StillCreatesEvents(t *testing.T) {
	repo := newMockRepo()
	repo.upsertErr = errors.New("db error")
	cal := &mockCalendar{}

	m := &Monitor{
		Scraper: &mockScraper{
			links: []model.ScheduleLink{
				{SpreadsheetID: "sheet123", Date: time.Date(2026, 4, 4, 0, 0, 0, 0, cst()), LinkText: "April 4"},
			},
			rows: testRows(),
		},
		Repo:      repo,
		Calendar:  cal,
		TeamName:  "Stl Bears Bell",
		SheetName: "10u",
		Year:      2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	require.NoError(t, err)
	assert.Len(t, cal.createdEvents, 1)
}

func TestRun_UnknownLocation_StillProcesses(t *testing.T) {
	repo := newMockRepo()
	cal := &mockCalendar{}

	m := &Monitor{
		Scraper: &mockScraper{
			links: []model.ScheduleLink{
				{SpreadsheetID: "sheet123", Date: time.Date(2026, 4, 4, 0, 0, 0, 0, cst()), LinkText: "April 4"},
			},
			rows: [][]string{
				{"", "Stl Bears Bell", "vs", "", "Gamers Blue", "Saturday", "900", "Unknown Park", "1"},
			},
		},
		Repo:      repo,
		Calendar:  cal,
		TeamName:  "Stl Bears Bell",
		SheetName: "10u",
		Year:      2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Len(t, repo.upsertedGames, 1)
	assert.Equal(t, "", repo.upsertedGames[0].Address)
}

func TestRun_MultipleLinks(t *testing.T) {
	repo := newMockRepo()
	repo.processed["sheet111"] = true

	m := &Monitor{
		Scraper: &mockScraper{
			links: []model.ScheduleLink{
				{SpreadsheetID: "sheet111", Date: time.Date(2026, 4, 4, 0, 0, 0, 0, cst()), LinkText: "April 4"},
				{SpreadsheetID: "sheet222", Date: time.Date(2026, 4, 11, 0, 0, 0, 0, cst()), LinkText: "April 11"},
			},
			rows: testRows(),
		},
		Repo:      repo,
		Calendar:  &mockCalendar{},
		TeamName:  "Stl Bears Bell",
		SheetName: "10u",
		Year:      2026,
	}

	err := m.Run(context.Background(), "https://example.com")
	require.NoError(t, err)
	// Only sheet222 should be processed (sheet111 was already done)
	assert.Equal(t, []string{"sheet222"}, repo.markedProcessed)
	assert.Len(t, repo.upsertedGames, 1)
}
