package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Location struct {
	Name    string
	Address string
}

type Game struct {
	HomeTeam    string
	AwayTeam    string
	DateTime    time.Time
	Location    string
	FieldNumber string
	Address     string
	HomeScore   *int
	AwayScore   *int
}

func (g Game) Opponent(team string) string {
	if strings.EqualFold(strings.TrimSpace(g.HomeTeam), strings.TrimSpace(team)) {
		return g.AwayTeam
	}
	return g.HomeTeam
}

func (g Game) Summary(team string) string {
	return fmt.Sprintf("%s vs %s", team, g.Opponent(team))
}

func ParseGames(rows [][]string, targetTeam string, tournamentDate time.Time) []Game {
	var games []Game
	target := strings.ToLower(strings.TrimSpace(targetTeam))

	for _, row := range rows {
		row = trimAll(row)
		if len(row) < 9 {
			continue
		}
		if row[2] != "vs" {
			continue
		}

		home := row[1]
		away := row[4]

		if !strings.EqualFold(strings.TrimSpace(strings.ToLower(home)), target) &&
			!strings.EqualFold(strings.TrimSpace(strings.ToLower(away)), target) {
			continue
		}

		dayName := strings.TrimSpace(row[5])
		gameDate := tournamentDate
		if strings.EqualFold(dayName, "Sunday") {
			gameDate = gameDate.AddDate(0, 0, 1)
		}

		gameTime := parseGameTime(row[6], gameDate)
		location := row[7]
		fieldNumber := row[8]

		game := Game{
			HomeTeam:    home,
			AwayTeam:    away,
			DateTime:    gameTime,
			Location:    location,
			FieldNumber: fieldNumber,
		}
		if score, err := strconv.Atoi(row[0]); err == nil {
			game.HomeScore = &score
		}
		if score, err := strconv.Atoi(row[3]); err == nil {
			game.AwayScore = &score
		}
		games = append(games, game)
	}

	return games
}

func parseGameTime(timeStr string, baseDate time.Time) time.Time {
	timeStr = strings.TrimSpace(timeStr)
	var hour, min int

	switch len(timeStr) {
	case 3:
		hour, _ = strconv.Atoi(timeStr[:1])
		min, _ = strconv.Atoi(timeStr[1:])
	case 4:
		hour, _ = strconv.Atoi(timeStr[:2])
		min, _ = strconv.Atoi(timeStr[2:])
	}

	// Games don't start before 7 AM, so low hours mean PM
	if hour < 7 {
		hour += 12
	}

	loc, _ := time.LoadLocation("America/Chicago")
	return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
		hour, min, 0, 0, loc)
}

func trimAll(fields []string) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = strings.TrimSpace(f)
	}
	return out
}
