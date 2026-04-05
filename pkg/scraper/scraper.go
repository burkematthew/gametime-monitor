package scraper

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"log"

	"github.com/matthew-burke/gametime-monitor/pkg/model"
)

var (
	linkRe           = regexp.MustCompile(`<a[^>]+href="(https://docs\.google\.com/spreadsheets/d/([a-zA-Z0-9_-]+)/edit[^"]*)"[^>]*>([^<]+)</a>`)
	dateRe           = regexp.MustCompile(`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})`)
	lookWhosComingRe = regexp.MustCompile(`(?i)look\s+who`)
)

func ScrapeScheduleLinks(pageURL string, year int) ([]model.ScheduleLink, error) {
	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading page: %w", err)
	}

	return ParseScheduleLinks(string(body), year), nil
}

func ParseScheduleLinks(html string, year int) []model.ScheduleLink {
	matches := linkRe.FindAllStringSubmatch(html, -1)

	var links []model.ScheduleLink
	for _, m := range matches {
		fullURL := m[1]
		spreadsheetID := m[2]
		linkText := m[3]

		if lookWhosComingRe.MatchString(linkText) {
			continue
		}

		dateMatch := dateRe.FindStringSubmatch(linkText)
		if dateMatch == nil {
			continue
		}

		monthStr := dateMatch[1]
		dayStr := dateMatch[2]

		dateStr := fmt.Sprintf("%s %s, %d", monthStr, dayStr, year)
		loc, _ := time.LoadLocation("America/Chicago")
		t, err := time.ParseInLocation("January 2, 2006", dateStr, loc)
		if err != nil {
			continue
		}

		links = append(links, model.ScheduleLink{
			SpreadsheetID: spreadsheetID,
			Date:          t,
			URL:           fullURL,
			LinkText:      linkText,
		})
	}

	return links
}

func FetchCSV(spreadsheetID, sheetName string) ([][]string, error) {
	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/gviz/tq?tqx=out:csv&sheet=%s",
		spreadsheetID, sheetName)
	return FetchCSVFromURL(url)
}

func FetchCSVFromURL(url string) ([][]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching CSV: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("CSV fetch returned status %d", resp.StatusCode)
	}

	return ParseCSV(resp.Body)
}

func ParseCSV(r io.Reader) ([][]string, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	var rows [][]string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var parseErr *csv.ParseError
			if errors.As(err, &parseErr) {
				continue
			}
			return rows, fmt.Errorf("reading CSV: %w", err)
		}
		rows = append(rows, row)
	}

	return rows, nil
}
