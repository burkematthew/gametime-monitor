package calendar

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/matthew-burke/gametime-monitor/pkg/model"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const GameDuration = 90 * time.Minute

type Client struct {
	svc        *calendar.Service
	calendarID string
}

func New(credentialsPath, calendarID string) (*Client, error) {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	conf, err := google.JWTConfigFromJSON(data, calendar.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	ctx := context.Background()
	httpClient := conf.Client(ctx)

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("creating calendar service: %w", err)
	}

	return &Client{svc: svc, calendarID: calendarID}, nil
}

func NewWithService(svc *calendar.Service, calendarID string) *Client {
	return &Client{svc: svc, calendarID: calendarID}
}

func (c *Client) CreateGameEvent(game model.Game, teamName string) error {
	exists, err := c.eventExists(game, teamName)
	if err != nil {
		log.Printf("Warning: could not check for duplicate event: %v", err)
	}
	if exists {
		log.Printf("Event already exists: %s at %s", game.Summary(teamName), game.DateTime.Format("Mon 3:04 PM"))
		return nil
	}

	summary := game.Summary(teamName)
	start := game.DateTime
	end := start.Add(GameDuration)

	locationDisplay := game.Location
	if game.FieldNumber != "" {
		locationDisplay = fmt.Sprintf("%s Field %s", game.Location, game.FieldNumber)
	}

	eventLocation := locationDisplay
	if game.Address != "" {
		eventLocation = game.Address
	}

	event := &calendar.Event{
		Summary:  summary,
		Location: eventLocation,
		Start: &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: "America/Chicago",
		},
		End: &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: "America/Chicago",
		},
		Description: fmt.Sprintf("10U Tournament Game\n%s", locationDisplay),
	}

	_, err = c.svc.Events.Insert(c.calendarID, event).Do()
	if err != nil {
		return fmt.Errorf("creating event: %w", err)
	}

	log.Printf("Created event: %s at %s (%s)", summary, start.Format("Mon Jan 2 3:04 PM"), locationDisplay)
	return nil
}

func (c *Client) eventExists(game model.Game, teamName string) (bool, error) {
	start := game.DateTime
	end := start.Add(GameDuration)

	events, err := c.svc.Events.List(c.calendarID).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		Do()
	if err != nil {
		return false, err
	}

	summary := game.Summary(teamName)
	for _, e := range events.Items {
		if e.Summary == summary {
			return true, nil
		}
	}

	return false, nil
}
