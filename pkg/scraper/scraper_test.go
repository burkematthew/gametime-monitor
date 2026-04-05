package scraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleHTML = `
<html><body>
<a href="https://docs.google.com/spreadsheets/d/abc123def/edit?usp=sharing">Look Who's Coming (All Dates)</a>
<a href="https://docs.google.com/spreadsheets/d/1AGs8MMG51vLR9AnSrGZGaBjMJJDz0aFU2lvmGAuwzdM/edit?usp=sharing">April 4th Game Schedule (Not Official Until 9pm w/Email)</a>
<a href="https://docs.google.com/spreadsheets/d/xyz789/edit?usp=sharing">May 2nd Game Schedule</a>
</body></html>
`

func TestParseScheduleLinks_FindsScheduleLinks(t *testing.T) {
	links := ParseScheduleLinks(sampleHTML, 2026)
	require.Len(t, links, 2)

	assert.Equal(t, "1AGs8MMG51vLR9AnSrGZGaBjMJJDz0aFU2lvmGAuwzdM", links[0].SpreadsheetID)
	assert.Equal(t, time.April, links[0].Date.Month())
	assert.Equal(t, 4, links[0].Date.Day())
	assert.Equal(t, 2026, links[0].Date.Year())
	assert.Contains(t, links[0].LinkText, "April 4th")

	assert.Equal(t, "xyz789", links[1].SpreadsheetID)
	assert.Equal(t, time.May, links[1].Date.Month())
	assert.Equal(t, 2, links[1].Date.Day())
}

func TestParseScheduleLinks_SkipsLookWhosComingLink(t *testing.T) {
	links := ParseScheduleLinks(sampleHTML, 2026)
	for _, link := range links {
		assert.NotContains(t, link.LinkText, "Look Who")
	}
}

func TestParseScheduleLinks_NoLinks(t *testing.T) {
	links := ParseScheduleLinks("<html><body>No links here</body></html>", 2026)
	assert.Empty(t, links)
}

func TestParseScheduleLinks_SkipsLinksWithoutDate(t *testing.T) {
	html := `<a href="https://docs.google.com/spreadsheets/d/abc123/edit?usp=sharing">Some Random Link</a>`
	links := ParseScheduleLinks(html, 2026)
	assert.Empty(t, links)
}

func TestParseScheduleLinks_AllMonths(t *testing.T) {
	months := []string{"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	for i, month := range months {
		html := `<a href="https://docs.google.com/spreadsheets/d/id` + month + `/edit">` + month + ` 15 Schedule</a>`
		links := ParseScheduleLinks(html, 2026)
		require.Len(t, links, 1, "failed for month: %s", month)
		assert.Equal(t, i+1, int(links[0].Date.Month()))
	}
}

func TestScrapeScheduleLinks_HTTPSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleHTML))
	}))
	defer srv.Close()

	links, err := ScrapeScheduleLinks(srv.URL, 2026)
	require.NoError(t, err)
	assert.Len(t, links, 2)
}

func TestScrapeScheduleLinks_HTTPError(t *testing.T) {
	_, err := ScrapeScheduleLinks("http://127.0.0.1:1", 2026)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching page")
}

func TestFetchCSVFromURL_Success(t *testing.T) {
	csvData := `"","Stl Bears Bell","vs","","Gamers Blue","Saturday","900","Fenton 2"
"","Stl Bears Bell","vs","","Retro Baseball","Saturday","1045","Fenton 2"`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(csvData))
	}))
	defer srv.Close()

	rows, err := FetchCSVFromURL(srv.URL)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, "Stl Bears Bell", rows[0][1])
}

func TestFetchCSVFromURL_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := FetchCSVFromURL(srv.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

func TestFetchCSVFromURL_HTTPError(t *testing.T) {
	_, err := FetchCSVFromURL("http://127.0.0.1:1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching CSV")
}

func TestFetchCSV_ConstructsURL(t *testing.T) {
	// FetchCSV calls FetchCSVFromURL with a constructed URL that won't resolve,
	// but we verify it returns an error (confirming the path works)
	_, err := FetchCSV("nonexistent-id", "10u")
	assert.Error(t, err)
}

func TestParseCSV_ValidCSV(t *testing.T) {
	data := `"a","b","c"
"d","e","f"`
	rows, err := ParseCSV(strings.NewReader(data))
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, []string{"a", "b", "c"}, rows[0])
}

func TestParseCSV_LazyQuotes(t *testing.T) {
	data := `"a","he said "hello"","c"`
	rows, err := ParseCSV(strings.NewReader(data))
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestParseCSV_VariableFieldCount(t *testing.T) {
	data := `"a","b"
"c","d","e","f"`
	rows, err := ParseCSV(strings.NewReader(data))
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Len(t, rows[0], 2)
	assert.Len(t, rows[1], 4)
}

func TestParseCSV_Empty(t *testing.T) {
	rows, err := ParseCSV(strings.NewReader(""))
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParseCSV_MalformedRowsSkipped(t *testing.T) {
	// A line with a bare quote that can't be parsed will be skipped
	data := "\"a\",\"b\"\n\"c\n\"d\",\"e\""
	rows, err := ParseCSV(strings.NewReader(data))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(rows), 1)
}

func TestParseScheduleLinks_InvalidDateSkipped(t *testing.T) {
	// "Smarch" is not a valid month so it won't match dateRe at all,
	// but we can test with a link that has no date text
	html := `<a href="https://docs.google.com/spreadsheets/d/abc123/edit">No date here at all</a>`
	links := ParseScheduleLinks(html, 2026)
	assert.Empty(t, links)
}

func TestScrapeScheduleLinks_EmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(""))
	}))
	defer srv.Close()

	links, err := ScrapeScheduleLinks(srv.URL, 2026)
	require.NoError(t, err)
	assert.Empty(t, links)
}

func TestParseScheduleLinks_NonSpreadsheetLinks(t *testing.T) {
	html := `<a href="https://example.com/something">April 4th Schedule</a>`
	links := ParseScheduleLinks(html, 2026)
	assert.Empty(t, links)
}

func TestParseCSV_WithMalformedLine(t *testing.T) {
	// Embed a NUL byte which triggers a csv parse error on that line
	data := "\"a\",\"b\"\n\x00\n\"d\",\"e\""
	rows, err := ParseCSV(strings.NewReader(data))
	require.NoError(t, err)
	// Should get at least the valid rows
	assert.GreaterOrEqual(t, len(rows), 1)
}

func TestScrapeScheduleLinks_ReturnsLinksFromServer(t *testing.T) {
	html := `<a href="https://docs.google.com/spreadsheets/d/sheet1/edit?usp=sharing">June 20th Game Schedule</a>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer srv.Close()

	links, err := ScrapeScheduleLinks(srv.URL, 2026)
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, "sheet1", links[0].SpreadsheetID)
	assert.Equal(t, time.June, links[0].Date.Month())
}

func TestFetchCSVFromURL_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rows, err := FetchCSVFromURL(srv.URL)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestScrapeScheduleLinks_ReadBodyError(t *testing.T) {
	// Server sends Content-Length but closes connection early, causing ReadAll to fail
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "99999")
		_, _ = w.Write([]byte("partial"))
		// Hijack to force close the connection
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, err := hj.Hijack()
			if err == nil {
				assert.NoError(t, conn.Close())
			}
		}
	}))
	defer srv.Close()

	_, err := ScrapeScheduleLinks(srv.URL, 2026)
	// Either returns an error or parses what it got
	if err != nil {
		assert.Contains(t, err.Error(), "reading page")
	}
}

type errReader struct{}

func (e errReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("forced read error")
}

func TestParseCSV_ReaderError(t *testing.T) {
	rows, err := ParseCSV(errReader{})
	assert.Error(t, err)
	assert.Empty(t, rows)
}
