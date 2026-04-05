package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cst() *time.Location {
	loc, _ := time.LoadLocation("America/Chicago")
	return loc
}

func TestOpponent_HomeTeam(t *testing.T) {
	g := Game{HomeTeam: "Bears", AwayTeam: "Wolves"}
	assert.Equal(t, "Wolves", g.Opponent("Bears"))
}

func TestOpponent_AwayTeam(t *testing.T) {
	g := Game{HomeTeam: "Bears", AwayTeam: "Wolves"}
	assert.Equal(t, "Bears", g.Opponent("Wolves"))
}

func TestOpponent_CaseInsensitive(t *testing.T) {
	g := Game{HomeTeam: "Stl Bears Bell", AwayTeam: "Gamers Blue"}
	assert.Equal(t, "Gamers Blue", g.Opponent("stl bears bell"))
}

func TestOpponent_WithWhitespace(t *testing.T) {
	g := Game{HomeTeam: " Bears ", AwayTeam: "Wolves"}
	assert.Equal(t, "Wolves", g.Opponent(" Bears "))
}

func TestSummary_HomeTeam(t *testing.T) {
	g := Game{HomeTeam: "Bears", AwayTeam: "Wolves"}
	assert.Equal(t, "Bears vs Wolves", g.Summary("Bears"))
}

func TestSummary_AwayTeam(t *testing.T) {
	g := Game{HomeTeam: "Bears", AwayTeam: "Wolves"}
	assert.Equal(t, "Wolves vs Bears", g.Summary("Wolves"))
}

func TestParseGames_MatchingHomeTeam(t *testing.T) {
	rows := [][]string{
		{"", "Stl Bears Bell", "vs", "", "Gamers Blue", "Saturday", "900", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	require.Len(t, games, 1)
	assert.Equal(t, "Stl Bears Bell", games[0].HomeTeam)
	assert.Equal(t, "Gamers Blue", games[0].AwayTeam)
	assert.Equal(t, "Fenton", games[0].Location)
	assert.Equal(t, "2", games[0].FieldNumber)
	assert.Equal(t, 9, games[0].DateTime.Hour())
	assert.Equal(t, 0, games[0].DateTime.Minute())
}

func TestParseGames_MatchingAwayTeam(t *testing.T) {
	rows := [][]string{
		{"", "Gamers Blue", "vs", "", "Stl Bears Bell", "Saturday", "1045", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	require.Len(t, games, 1)
	assert.Equal(t, "Gamers Blue", games[0].HomeTeam)
	assert.Equal(t, "Stl Bears Bell", games[0].AwayTeam)
}

func TestParseGames_CaseInsensitiveMatch(t *testing.T) {
	rows := [][]string{
		{"", "stl bears bell", "vs", "", "Gamers Blue", "Saturday", "900", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	require.Len(t, games, 1)
}

func TestParseGames_NoMatch(t *testing.T) {
	rows := [][]string{
		{"", "Tribe", "vs", "", "Gamers Blue", "Saturday", "1230", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	assert.Empty(t, games)
}

func TestParseGames_SkipsShortRows(t *testing.T) {
	rows := [][]string{
		{"", "Bears", "vs", "", "Wolves"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Bears", tournamentDate)
	assert.Empty(t, games)
}

func TestParseGames_SkipsNonVsRows(t *testing.T) {
	rows := [][]string{
		{"Score", "10u Baseball", "", "W", "L", "T", "", "", ""},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "10u Baseball", tournamentDate)
	assert.Empty(t, games)
}

func TestParseGames_SundayAddsDay(t *testing.T) {
	rows := [][]string{
		{"", "Stl Bears Bell", "vs", "", "Gamers Blue", "Sunday", "900", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	require.Len(t, games, 1)
	assert.Equal(t, 5, games[0].DateTime.Day())
}

func TestParseGames_WithScores(t *testing.T) {
	rows := [][]string{
		{"5", "Stl Bears Bell", "vs", "3", "Gamers Blue", "Saturday", "900", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	require.Len(t, games, 1)
	require.NotNil(t, games[0].HomeScore)
	require.NotNil(t, games[0].AwayScore)
	assert.Equal(t, 5, *games[0].HomeScore)
	assert.Equal(t, 3, *games[0].AwayScore)
}

func TestParseGames_WithoutScores(t *testing.T) {
	rows := [][]string{
		{"", "Stl Bears Bell", "vs", "", "Gamers Blue", "Saturday", "900", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	require.Len(t, games, 1)
	assert.Nil(t, games[0].HomeScore)
	assert.Nil(t, games[0].AwayScore)
}

func TestParseGames_MultipleGames(t *testing.T) {
	rows := [][]string{
		{"", "Stl Bears Bell", "vs", "", "Gamers Blue", "Saturday", "900", "Fenton", "2"},
		{"", "Stl Bears Bell", "vs", "", "Retro Baseball", "Saturday", "1045", "Fenton", "2"},
		{"", "Tribe", "vs", "", "Gamers Blue", "Saturday", "1230", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	require.Len(t, games, 2)
}

func TestParseGames_WhitespaceInTeamNames(t *testing.T) {
	rows := [][]string{
		{"", " Stl Bears Bell ", "vs", "", " Gamers Blue ", "Saturday", "900", "Fenton", "2"},
	}
	tournamentDate := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	games := ParseGames(rows, "Stl Bears Bell", tournamentDate)
	require.Len(t, games, 1)
}

func TestParseGameTime_ThreeDigit(t *testing.T) {
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())
	result := parseGameTime("900", base)
	assert.Equal(t, 9, result.Hour())
	assert.Equal(t, 0, result.Minute())
}

func TestParseGameTime_FourDigit(t *testing.T) {
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())
	result := parseGameTime("1045", base)
	assert.Equal(t, 10, result.Hour())
	assert.Equal(t, 45, result.Minute())
}

func TestParseGameTime_PMConversion(t *testing.T) {
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())

	result := parseGameTime("215", base)
	assert.Equal(t, 14, result.Hour())
	assert.Equal(t, 15, result.Minute())
}

func TestParseGameTime_NoonStaysNoon(t *testing.T) {
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())
	result := parseGameTime("1200", base)
	assert.Equal(t, 12, result.Hour())
	assert.Equal(t, 0, result.Minute())
}

func TestParseGameTime_Whitespace(t *testing.T) {
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())
	result := parseGameTime("  900  ", base)
	assert.Equal(t, 9, result.Hour())
}

func TestParseGameTime_PreservesDate(t *testing.T) {
	base := time.Date(2026, 7, 11, 0, 0, 0, 0, cst())
	result := parseGameTime("900", base)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.July, result.Month())
	assert.Equal(t, 11, result.Day())
}

func TestParseGameTime_UsesChicagoTimezone(t *testing.T) {
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())
	result := parseGameTime("900", base)
	assert.Equal(t, "America/Chicago", result.Location().String())
}

func TestParseGameTime_InvalidLength(t *testing.T) {
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, cst())
	result := parseGameTime("12", base)
	// hour=0, min=0, hour < 7 so becomes 12:00
	assert.Equal(t, 12, result.Hour())
	assert.Equal(t, 0, result.Minute())
}

func TestTrimAll(t *testing.T) {
	input := []string{" foo ", "bar", " baz "}
	result := trimAll(input)
	assert.Equal(t, []string{"foo", "bar", "baz"}, result)
}

func TestTrimAll_Empty(t *testing.T) {
	result := trimAll([]string{})
	assert.Empty(t, result)
}
