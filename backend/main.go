package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"
)

const (
	clubID   = "441d3783-06aa-436a-b438-359300ee0371"
	clubType = "futsal"
	baseURL  = "https://facr.tdvorak.dev"
)

type ClubDetail struct {
	Name         string `json:"name"`
	ClubID       string `json:"club_id"`
	ClubType     string `json:"club_type"`
	URL          string `json:"url"`
	LogoURL      string `json:"logo_url"`
	Address      string `json:"address"`
	Category     string `json:"category"`
	Competitions []struct {
		ID          string `json:"id"`
		Code        string `json:"code"`
		Name        string `json:"name"`
		TeamCount   string `json:"team_count"`
		MatchesLink string `json:"matches_link"`
		Matches     []struct {
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
		} `json:"matches"`
	} `json:"competitions"`
}

func dataPath() string {
	if p := os.Getenv("DATA_PATH"); p != "" {
		return p
	}
	return "/app/data/club.json"
}

type ClubTable struct {
	Name         string `json:"name"`
	ClubID       string `json:"club_id"`
	ClubType     string `json:"club_type"`
	LogoURL      string `json:"logo_url"`
	Competitions []struct {
		ID          string `json:"id"`
		Code        string `json:"code"`
		Name        string `json:"name"`
		TeamCount   string `json:"team_count"`
		MatchesLink string `json:"matches_link"`
		Table       struct {
			Overall []struct {
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
			} `json:"overall"`
		} `json:"table"`
	} `json:"competitions"`
}

type Combined struct {
	FetchedAt  time.Time  `json:"fetched_at"`
	ClubDetail ClubDetail `json:"club_detail"`
	ClubTable  ClubTable  `json:"club_table"`
}

type cache struct {
	mu   sync.RWMutex
	data Combined
}

var c cache

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// initial fetch
	if err := refresh(ctx); err != nil {
		log.Printf("initial refresh error: %v", err)
	}

	// scheduler
	go scheduler(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/data/club.json", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			c.mu.RLock()
			defer c.mu.RUnlock()
			_ = json.NewEncoder(w).Encode(c.data)
		case http.MethodDelete:
			// delete on-disk file and clear in-memory cache
			path := dataPath()
			_ = os.Remove(path)
			c.mu.Lock()
			c.data = Combined{}
			c.mu.Unlock()
			// trigger immediate refresh so next GET has fresh data
			if err := refresh(r.Context()); err != nil {
				log.Printf("manual refresh after delete failed: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/data/club.js", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		c.mu.RLock()
		payload, _ := json.Marshal(c.data)
		c.mu.RUnlock()
		w.Write([]byte("window.FACR_DATA="))
		w.Write(payload)
		w.Write([]byte(";"))
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	go func() {
		log.Println("backend listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	ctxShut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxShut)
}

func okCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func scheduler(ctx context.Context) {
	// default 30m; during match window (±2h around any match today), 2m
	for {
		var d time.Duration = 30 * time.Minute
		if withinMatchWindow() {
			d = 2 * time.Minute
		}
		select {
		case <-time.After(d):
			if err := refresh(ctx); err != nil {
				log.Printf("refresh error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func withinMatchWindow() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	loc, err := time.LoadLocation("Europe/Prague")
	if err != nil {
		// Fallback to local/UTC if tzdata is missing to avoid panic
		loc = time.Local
	}
	now := time.Now().In(loc)
	for _, comp := range c.data.ClubDetail.Competitions {
		for _, m := range comp.Matches {
			// date format in API example: "12.08.2023 18:00" or "12.08.2023 18:00"
			t, err := time.ParseInLocation("02.01.2006 15:04", m.DateTime, loc)
			if err != nil {
				continue
			}
			if absDuration(now.Sub(t)) <= 2*time.Hour {
				return true
			}
		}
	}
	return false
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func refresh(ctx context.Context) error {
	client := &http.Client{Timeout: 30 * time.Second}
	urlDetail := fmt.Sprintf("%s/club/%s/%s", baseURL, clubType, clubID)
	urlTable := fmt.Sprintf("%s/club/%s/%s/table", baseURL, clubType, clubID)

	var detail ClubDetail
	if err := getJSON(ctx, client, urlDetail, &detail); err != nil {
		return fmt.Errorf("detail: %w", err)
	}
	var table ClubTable
	if err := getJSON(ctx, client, urlTable, &table); err != nil {
		return fmt.Errorf("table: %w", err)
	}

	// Override or inject facr_link based on match_id
	for i := range detail.Competitions {
		for j := range detail.Competitions[i].Matches {
			mid := detail.Competitions[i].Matches[j].MatchID
			if mid != "" {
				detail.Competitions[i].Matches[j].FacrLink = fmt.Sprintf("https://www.fotbal.cz/futsal/zapasy/futsal/%s", mid)
			}
			// Override logo URLs for our club in match details
			if detail.Competitions[i].Matches[j].HomeID == clubID {
				detail.Competitions[i].Matches[j].HomeLogoURL = "/img/logo.png"
			}
			if detail.Competitions[i].Matches[j].AwayID == clubID {
				detail.Competitions[i].Matches[j].AwayLogoURL = "/img/logo.png"
			}
		}
	}

	// Override logo URLs for our club in the table standings
	for i := range table.Competitions {
		for j := range table.Competitions[i].Table.Overall {
			if table.Competitions[i].Table.Overall[j].TeamID == clubID {
				table.Competitions[i].Table.Overall[j].TeamLogo = "/img/logo.png"
			}
		}
	}

	c.mu.Lock()
	c.data = Combined{
		FetchedAt:  time.Now(),
		ClubDetail: detail,
		ClubTable:  table,
	}
	c.mu.Unlock()

	// persist to disk for control/deletion
	if err := writeDiskJSON(c.data); err != nil {
		log.Printf("warn: write disk json: %v", err)
	}
	log.Printf("refreshed data: comps=%d", len(detail.Competitions))
	return nil
}

func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("status %d: %s", resp.StatusCode, bytes.TrimSpace(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func writeDiskJSON(d Combined) error {
	path := dataPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	// On Windows, Rename over existing file may fail; remove target first.
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
