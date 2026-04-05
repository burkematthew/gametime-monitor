package model

import "time"

type ScheduleLink struct {
	SpreadsheetID string
	Date          time.Time
	URL           string
	LinkText      string
}
