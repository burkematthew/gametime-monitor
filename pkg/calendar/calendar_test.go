package calendar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/matthew-burke/gametime-monitor/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func cst() *time.Location {
	loc, _ := time.LoadLocation("America/Chicago")
	return loc
}

func testGame() model.Game {
	return model.Game{
		HomeTeam:    "Stl Bears Bell",
		AwayTeam:    "Gamers Blue",
		DateTime:    time.Date(2026, 4, 4, 9, 0, 0, 0, cst()),
		Location:    "Fenton",
		FieldNumber: "2",
		Address:     "945 Larkin Williams Rd, Fenton, MO 63026",
	}
}

func setupTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	svc, err := calendar.NewService(t.Context(), option.WithHTTPClient(srv.Client()), option.WithEndpoint(srv.URL))
	require.NoError(t, err)

	return NewWithService(svc, "test-calendar-id")
}

func TestNewWithService(t *testing.T) {
	svc, err := calendar.NewService(t.Context(), option.WithHTTPClient(http.DefaultClient))
	require.NoError(t, err)

	client := NewWithService(svc, "cal-id")
	assert.NotNil(t, client)
	assert.Equal(t, "cal-id", client.calendarID)
}

func TestNew_MissingCredentials(t *testing.T) {
	_, err := New("/nonexistent/path.json", "cal-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading credentials")
}

func TestNew_ValidCredentials(t *testing.T) {
	// A valid-format service account JSON (non-functional keys)
	creds := `{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "key123",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nDUMMY-KEY-FOR-TESTING-ONLY-NOT-A-REAL-KEY\n-----END RSA PRIVATE KEY-----\n",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "123456789",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token"
	}`

	tmpFile, err := os.CreateTemp("", "creds-*.json")
	require.NoError(t, err)
	defer func() { require.NoError(t, os.Remove(tmpFile.Name())) }()

	_, _ = tmpFile.WriteString(creds)
	require.NoError(t, tmpFile.Close())

	client, err := New(tmpFile.Name(), "cal-id")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_InvalidCredentials(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "creds-*.json")
	require.NoError(t, err)
	defer func() { require.NoError(t, os.Remove(tmpFile.Name())) }()

	_, _ = tmpFile.WriteString(`{"not": "valid credentials"}`)
	require.NoError(t, tmpFile.Close())

	_, err = New(tmpFile.Name(), "cal-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing credentials")
}

func TestCreateGameEvent_Success(t *testing.T) {
	game := testGame()
	insertCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// eventExists check — return empty list
			_ = json.NewEncoder(w).Encode(&calendar.Events{Items: []*calendar.Event{}})
			return
		}
		if r.Method == "POST" {
			insertCalled = true
			_ = json.NewEncoder(w).Encode(&calendar.Event{Id: "new-event-id"})
			return
		}
	})

	client := setupTestClient(t, mux)

	err := client.CreateGameEvent(game, "Stl Bears Bell")
	require.NoError(t, err)
	assert.True(t, insertCalled, "insert should have been called")
}

func TestCreateGameEvent_SkipsDuplicate(t *testing.T) {
	game := testGame()
	insertCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Return an existing event with the same summary
			events := &calendar.Events{
				Items: []*calendar.Event{
					{Summary: "Stl Bears Bell vs Gamers Blue"},
				},
			}
			_ = json.NewEncoder(w).Encode(events)
			return
		}
		if r.Method == "POST" {
			insertCalled = true
			_ = json.NewEncoder(w).Encode(&calendar.Event{Id: "id"})
			return
		}
	})

	client := setupTestClient(t, mux)

	err := client.CreateGameEvent(game, "Stl Bears Bell")
	require.NoError(t, err)
	assert.False(t, insertCalled, "insert should NOT have been called for duplicate")
}

func TestCreateGameEvent_ListErrorStillInserts(t *testing.T) {
	game := testGame()
	insertCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Method == "POST" {
			insertCalled = true
			_ = json.NewEncoder(w).Encode(&calendar.Event{Id: "id"})
			return
		}
	})

	client := setupTestClient(t, mux)

	err := client.CreateGameEvent(game, "Stl Bears Bell")
	require.NoError(t, err)
	assert.True(t, insertCalled, "should still attempt insert when list fails")
}

func TestCreateGameEvent_InsertError(t *testing.T) {
	game := testGame()

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(&calendar.Events{Items: []*calendar.Event{}})
			return
		}
		if r.Method == "POST" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	})

	client := setupTestClient(t, mux)

	err := client.CreateGameEvent(game, "Stl Bears Bell")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating event")
}

func TestCreateGameEvent_EventFields(t *testing.T) {
	game := testGame()

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(&calendar.Events{Items: []*calendar.Event{}})
			return
		}
		if r.Method == "POST" {
			var event calendar.Event
			_ = json.NewDecoder(r.Body).Decode(&event)

			assert.Equal(t, "Stl Bears Bell vs Gamers Blue", event.Summary)
			assert.Equal(t, "945 Larkin Williams Rd, Fenton, MO 63026", event.Location)
			assert.Contains(t, event.Description, "10U Tournament Game")
			assert.Equal(t, "America/Chicago", event.Start.TimeZone)
			assert.Equal(t, "America/Chicago", event.End.TimeZone)

			// Verify 90-minute duration
			start, _ := time.Parse(time.RFC3339, event.Start.DateTime)
			end, _ := time.Parse(time.RFC3339, event.End.DateTime)
			assert.Equal(t, 90*time.Minute, end.Sub(start))

			_ = json.NewEncoder(w).Encode(&calendar.Event{Id: "id"})
			return
		}
	})

	client := setupTestClient(t, mux)
	require.NoError(t, client.CreateGameEvent(game, "Stl Bears Bell"))
}

func TestGameDuration(t *testing.T) {
	assert.Equal(t, 90*time.Minute, GameDuration)
}

func TestEventExists_NoEvents(t *testing.T) {
	game := testGame()

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(&calendar.Events{Items: []*calendar.Event{}})
	})

	client := setupTestClient(t, mux)
	exists, err := client.eventExists(game, "Stl Bears Bell")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestEventExists_MatchingEvent(t *testing.T) {
	game := testGame()

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		events := &calendar.Events{
			Items: []*calendar.Event{
				{Summary: "Stl Bears Bell vs Gamers Blue"},
			},
		}
		_ = json.NewEncoder(w).Encode(events)
	})

	client := setupTestClient(t, mux)
	exists, err := client.eventExists(game, "Stl Bears Bell")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestEventExists_DifferentEvent(t *testing.T) {
	game := testGame()

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		events := &calendar.Events{
			Items: []*calendar.Event{
				{Summary: "Something Else"},
			},
		}
		_ = json.NewEncoder(w).Encode(events)
	})

	client := setupTestClient(t, mux)
	exists, err := client.eventExists(game, "Stl Bears Bell")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestEventExists_APIError(t *testing.T) {
	game := testGame()

	mux := http.NewServeMux()
	mux.HandleFunc("/calendars/test-calendar-id/events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	client := setupTestClient(t, mux)
	_, err := client.eventExists(game, "Stl Bears Bell")
	assert.Error(t, err)
}
