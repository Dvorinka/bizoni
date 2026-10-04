package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---- futsalliga.cz scraper ----
// Secondary data source used when the primary FAČR API is unavailable.
// Scrapes the official 2. Futsal liga - východ site (eSports/eSportsmedia CMS)
// and produces the same Combined JSON shape the frontend expects.

const (
	flBase     = "https://2fl-v.futsalliga.cz"
	flClubLogo = "/img/logo.png"
)

func flSeason() string {
	if s := os.Getenv("FL_SEASON"); s != "" {
		return s
	}
	return "2027" // 2026/2027
}

func flTeamID() string {
	if s := os.Getenv("FL_TEAM_ID"); s != "" {
		return s
	}
	return "60" // FC Bizoni Uherské Hradiště
}

var (
	flH2Re       = regexp.MustCompile(`(?s)<h2 class="delta[^"]*"[^>]*>.*?</h2>`)
	flDateRe     = regexp.MustCompile(`(\d{1,2})\.\s*(\d{1,2})\.\s*(\d{4})`)
	flRowRe      = regexp.MustCompile(`(?s)<a href="/match/(\d+)"[^>]*>\s*<div class="delta mb-0 pl-md-3 pl-lg-8">\s*([0-9:]+)\s*</div>`)
	flTeamRe     = regexp.MustCompile(`(?s)<div class="match-team">(.*?)</div>`)
	flImgRe      = regexp.MustCompile(`(?:data-src|src)="(https?://[^"]+)"`)
	flScoreRe    = regexp.MustCompile(`(?s)match-results[^>]*>\s*([^<]*?)\s*</div>`)
	flScoreNumRe = regexp.MustCompile(`(\d+)\s*:\s*(\d+)`)
	flVenueRe    = regexp.MustCompile(`(?s)<div class="match-place">\s*([^<]*?)\s*</div>`)
	flTagRe      = regexp.MustCompile(`<[^>]+>`)
	flWsRe       = regexp.MustCompile(`\s+`)
	flTableRe    = regexp.MustCompile(`(?s)<table[^>]*standingsTable[^>]*>(.*?)</table>`)
	flTrRe       = regexp.MustCompile(`(?s)<tr>(.*?)</tr>`)
	flTdRe       = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
)

// wire structs mirroring the FAČR API JSON shape; marshaled then unmarshaled
// into ClubDetail/ClubTable (which use anonymous nested structs)
type flWireMatch struct {
	DateTime    string `json:"date_time"`
	Home        string `json:"home"`
	HomeID      string `json:"home_id"`
	HomeLogoURL string `json:"home_logo_url"`
	Away        string `json:"away"`
	AwayID      string `json:"away_id"`
	AwayLogoURL string `json:"away_logo_url"`
	Score       string `json:"score"`
	Venue       string `json:"venue"`
	MatchID     string `json:"match_id"`
	ReportURL   string `json:"report_url"`
	FacrLink    string `json:"facr_link"`
}

type flWireCompetition struct {
	ID          string        `json:"id"`
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	TeamCount   string        `json:"team_count"`
	MatchesLink string        `json:"matches_link"`
	Matches     []flWireMatch `json:"matches"`
}

type flWireDetail struct {
	Name         string              `json:"name"`
	ClubID       string              `json:"club_id"`
	ClubType     string              `json:"club_type"`
	URL          string              `json:"url"`
	LogoURL      string              `json:"logo_url"`
	Address      string              `json:"address"`
	Category     string              `json:"category"`
	Competitions []flWireCompetition `json:"competitions"`
}

type flWireRow struct {
	Rank     string `json:"rank"`
	Team     string `json:"team"`
	TeamID   string `json:"team_id"`
	TeamLogo string `json:"team_logo_url"`
	Played   string `json:"played"`
	Wins     string `json:"wins"`
	Draws    string `json:"draws"`
	Losses   string `json:"losses"`
	Score    string `json:"score"`
	Points   string `json:"points"`
}

type flWireTableComp struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	TeamCount   string `json:"team_count"`
	MatchesLink string `json:"matches_link"`
	Table       struct {
		Overall []flWireRow `json:"overall"`
	} `json:"table"`
}

type flWireTable struct {
	Name         string            `json:"name"`
	ClubID       string            `json:"club_id"`
	ClubType     string            `json:"club_type"`
	LogoURL      string            `json:"logo_url"`
	Competitions []flWireTableComp `json:"competitions"`
}

func flClean(s string) string {
	return strings.TrimSpace(flWsRe.ReplaceAllString(flTagRe.ReplaceAllString(s, ""), " "))
}

func flIsBizoni(name string) bool {
	return strings.Contains(strings.ToLower(name), "bizoni")
}

func flTeamLogo(content string) string {
	if m := flImgRe.FindStringSubmatch(content); m != nil {
		return m[1]
	}
	return ""
}

func fetchText(ctx context.Context, client *http.Client, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (bizoni-site-scraper)")
	req.Header.Set("Accept", "text/html")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", u, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// flParseMatches parses the team-filtered match list page.
// The page groups matches under <h2> date headers ("neděle 4. 10. 2026");
// each row starts with a time anchor to /match/<id>.
func flParseMatches(doc string) []flWireMatch {
	var matches []flWireMatch
	teamID := flTeamID()
	headers := flH2Re.FindAllStringIndex(doc, -1)
	for i, loc := range headers {
		headText := flTagRe.ReplaceAllString(doc[loc[0]:loc[1]], "")
		dm := flDateRe.FindStringSubmatch(headText)
		if dm == nil {
			continue
		}
		day, _ := strconv.Atoi(dm[1])
		mon, _ := strconv.Atoi(dm[2])
		yr, _ := strconv.Atoi(dm[3])
		secEnd := len(doc)
		if i+1 < len(headers) {
			secEnd = headers[i+1][0]
		}
		section := doc[loc[1]:secEnd]
		rows := flRowRe.FindAllStringSubmatchIndex(section, -1)
		for j, rloc := range rows {
			matchID := section[rloc[2]:rloc[3]]
			kickoff := section[rloc[4]:rloc[5]]
			segEnd := len(section)
			if j+1 < len(rows) {
				segEnd = rows[j+1][0]
			}
			seg := section[rloc[1]:segEnd]

			teams := flTeamRe.FindAllStringSubmatch(seg, -1)
			if len(teams) < 2 {
				continue
			}
			home := flClean(teams[0][1])
			away := flClean(teams[1][1])
			homeLogo := flTeamLogo(teams[0][1])
			awayLogo := flTeamLogo(teams[1][1])
			if flIsBizoni(home) {
				homeLogo = flClubLogo
			}
			if flIsBizoni(away) {
				awayLogo = flClubLogo
			}

			score := ""
			if sm := flScoreRe.FindStringSubmatch(seg); sm != nil {
				if sc := flScoreNumRe.FindStringSubmatch(sm[1]); sc != nil {
					score = sc[1] + ":" + sc[2]
				}
			}

			venue := ""
			if vm := flVenueRe.FindStringSubmatch(seg); vm != nil {
				venue = flClean(vm[1])
			}

			m := flWireMatch{
				DateTime:    fmt.Sprintf("%02d.%02d.%04d %s", day, mon, yr, kickoff),
				Home:        home,
				HomeLogoURL: homeLogo,
				Away:        away,
				AwayLogoURL: awayLogo,
				Score:       score,
				Venue:       venue,
				MatchID:     matchID,
				ReportURL:   fmt.Sprintf("%s/match/%s", flBase, matchID),
				FacrLink:    fmt.Sprintf("%s/match/%s", flBase, matchID),
			}
			if flIsBizoni(home) {
				m.HomeID = teamID
			}
			if flIsBizoni(away) {
				m.AwayID = teamID
			}
			matches = append(matches, m)
		}
	}
	return matches
}

// flParseStandings parses the standings table (rank, team, Z, V, R, P, skóre, body).
func flParseStandings(doc string) []flWireRow {
	var rows []flWireRow
	tm := flTableRe.FindStringSubmatch(doc)
	if tm == nil {
		return rows
	}
	for _, tr := range flTrRe.FindAllStringSubmatch(tm[1], -1) {
		cells := flTdRe.FindAllStringSubmatch(tr[1], -1)
		if len(cells) < 8 {
			continue
		}
		team := flClean(cells[1][1])
		row := flWireRow{
			Rank:   flClean(cells[0][1]),
			Team:   team,
			Played: flClean(cells[2][1]),
			Wins:   flClean(cells[3][1]),
			Draws:  flClean(cells[4][1]),
			Losses: flClean(cells[5][1]),
			Score:  flClean(cells[6][1]),
			Points: flClean(cells[7][1]),
		}
		if flIsBizoni(team) {
			row.TeamID = flTeamID()
			row.TeamLogo = flClubLogo
		} else {
			row.TeamLogo = flTeamLogo(cells[1][1])
		}
		rows = append(rows, row)
	}
	return rows
}

func flCompName() string {
	y, err := strconv.Atoi(flSeason())
	if err != nil || y <= 0 {
		return "2. Futsal liga - východ"
	}
	return fmt.Sprintf("2. Futsal liga - východ %d/%d", y-1, y)
}

// scrapeFutsalLiga fetches Bizoni matches + league standings from
// 2fl-v.futsalliga.cz and returns a Combined payload equivalent to what the
// FAČR/flashscore APIs produce.
func scrapeFutsalLiga(ctx context.Context, client *http.Client) (Combined, error) {
	season := flSeason()
	teamID := flTeamID()

	mDoc, err := fetchText(ctx, client, fmt.Sprintf("%s/matches/%s?team=%s", flBase, season, teamID))
	if err != nil {
		return Combined{}, fmt.Errorf("futsalliga matches: %w", err)
	}
	sDoc, err := fetchText(ctx, client, fmt.Sprintf("%s/standings/%s", flBase, season))
	if err != nil {
		return Combined{}, fmt.Errorf("futsalliga standings: %w", err)
	}

	matches := flParseMatches(mDoc)
	if len(matches) == 0 {
		return Combined{}, fmt.Errorf("futsalliga: parsed 0 matches")
	}
	tableRows := flParseStandings(sDoc)

	clubName := "FC Bizoni Uherské Hradiště"
	matchesLink := fmt.Sprintf("%s/matches/%s?team=%s", flBase, season, teamID)
	teamCount := strconv.Itoa(len(tableRows))

	wd := flWireDetail{
		Name:     clubName,
		ClubID:   teamID,
		ClubType: "futsal",
		URL:      fmt.Sprintf("%s/team/%s", flBase, teamID),
		LogoURL:  flClubLogo,
		Category: "Futsal",
		Competitions: []flWireCompetition{{
			ID:          season,
			Code:        "2FL-V",
			Name:        flCompName(),
			TeamCount:   teamCount,
			MatchesLink: matchesLink,
			Matches:     matches,
		}},
	}
	wt := flWireTable{
		Name:     clubName,
		ClubID:   teamID,
		ClubType: "futsal",
		LogoURL:  flClubLogo,
	}
	wt.Competitions = append(wt.Competitions, flWireTableComp{
		ID:          season,
		Code:        "2FL-V",
		Name:        flCompName(),
		TeamCount:   teamCount,
		MatchesLink: matchesLink,
	})
	wt.Competitions[0].Table.Overall = tableRows

	var detail ClubDetail
	var table ClubTable
	db, err := json.Marshal(wd)
	if err != nil {
		return Combined{}, fmt.Errorf("futsalliga marshal detail: %w", err)
	}
	if err := json.Unmarshal(db, &detail); err != nil {
		return Combined{}, fmt.Errorf("futsalliga unmarshal detail: %w", err)
	}
	tb, err := json.Marshal(wt)
	if err != nil {
		return Combined{}, fmt.Errorf("futsalliga marshal table: %w", err)
	}
	if err := json.Unmarshal(tb, &table); err != nil {
		return Combined{}, fmt.Errorf("futsalliga unmarshal table: %w", err)
	}

	return Combined{
		FetchedAt:  time.Now(),
		ClubDetail: detail,
		ClubTable:  table,
	}, nil
}
