package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	_ "image/jpeg"
)

const (
	clubID   = "441d3783-06aa-436a-b438-359300ee0371"
	clubType = "futsal"
	baseURL  = "https://facr.tdvorak.dev"
)

// Paths
func dataPath() string {
	if p := os.Getenv("DATA_PATH"); p != "" {
		return p
	}
	// Default: Docker volume path (writable)
	return "/app/data/club.json"
}

// ---------------- Image normalization for blog thumbnails ----------------
// Target dimensions for blog images
const (
    blogImgW = 1600
    blogImgH = 969
)

// avgLuma computes average luminance of an image (0..255)
func avgLuma(img image.Image) float64 {
    b := img.Bounds()
    if b.Empty() { return 0 }
    var sum uint64
    var n uint64
    for y := b.Min.Y; y < b.Max.Y; y += 4 { // sample every 4th row for speed
        for x := b.Min.X; x < b.Max.X; x += 4 { // sample every 4th column
            r,g,bv,_ := img.At(x,y).RGBA()
            r8 := float64(r>>8)
            g8 := float64(g>>8)
            b8 := float64(bv>>8)
            // Rec. 601 approximate luma
            l := 0.299*r8 + 0.587*g8 + 0.114*b8
            if l < 0 { l = 0 }
            if l > 255 { l = 255 }
            sum += uint64(l)
            n++
        }
    }
    if n == 0 { return 0 }
    return float64(sum)/float64(n)
}

// fitWithin returns destination size that fits source into max size, without upscaling
func fitWithin(sw, sh, mw, mh int) (int, int) {
    if sw <= 0 || sh <= 0 { return 0, 0 }
    if sw <= mw && sh <= mh {
        return sw, sh
    }
    wr := float64(mw) / float64(sw)
    hr := float64(mh) / float64(sh)
    r := wr
    if hr < wr { r = hr }
    dw := int(float64(sw) * r)
    dh := int(float64(sh) * r)
    if dw < 1 { dw = 1 }
    if dh < 1 { dh = 1 }
    return dw, dh
}

// scaleNearest performs nearest-neighbor downscaling from src to a new RGBA of size (dw, dh)
func scaleNearest(src image.Image, dw, dh int) *image.RGBA {
    dst := image.NewRGBA(image.Rect(0,0,dw,dh))
    sb := src.Bounds()
    sw := sb.Dx()
    sh := sb.Dy()
    for y := 0; y < dh; y++ {
        sy := sb.Min.Y + int(float64(y)*float64(sh)/float64(dh))
        for x := 0; x < dw; x++ {
            sx := sb.Min.X + int(float64(x)*float64(sw)/float64(dw))
            dst.Set(x, y, src.At(sx, sy))
        }
    }
    return dst
}

// normalizeBlogImage decodes any supported image (PNG/JPEG) and writes a 1600x969 PNG with letterboxing (black/white)
func normalizeBlogImage(r io.Reader, outPath string) error {
    img, _, err := image.Decode(r)
    if err != nil {
        return fmt.Errorf("decode image: %w", err)
    }
    // Choose background based on average luminance
    l := avgLuma(img)
    bg := color.Black
    if l > 160 { // bright image -> white bg; tweak threshold as needed
        bg = color.White
    }
    // Compute fitted size (no upscaling)
    srcB := img.Bounds()
    dw, dh := fitWithin(srcB.Dx(), srcB.Dy(), blogImgW, blogImgH)
    var scaled image.Image
    if dw == srcB.Dx() && dh == srcB.Dy() {
        scaled = img
    } else {
        scaled = scaleNearest(img, dw, dh)
    }
    // Compose centered on canvas
    canvas := image.NewRGBA(image.Rect(0,0,blogImgW,blogImgH))
    draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C:bg}, image.Point{}, draw.Src)
    offX := (blogImgW - dw) / 2
    offY := (blogImgH - dh) / 2
    draw.Draw(canvas, image.Rect(offX, offY, offX+dw, offY+dh), scaled, scaled.Bounds().Min, draw.Over)
    // Write PNG atomically
    if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil { return err }
    tmp := outPath + ".tmp"
    f, err := os.Create(tmp)
    if err != nil { return err }
    enc := png.Encoder{CompressionLevel: png.BestSpeed}
    if err := enc.Encode(f, canvas); err != nil {
        f.Close(); _ = os.Remove(tmp); return fmt.Errorf("encode png: %w", err)
    }
    f.Close()
    _ = os.Remove(outPath)
    if err := os.Rename(tmp, outPath); err != nil { return err }
    return nil
}

func staticPath() string {
	if p := os.Getenv("STATIC_PATH"); p != "" {
		return p
	}
	// Default: use current working directory when running locally
	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		// If CWD contains index.html, assume it's the site root
		if _, err := os.Stat(filepath.Join(cwd, "index.html")); err == nil {
			return cwd
		}
		// Otherwise, try parent directory (common when running from ./backend)
		parent := filepath.Dir(cwd)
		if parent != "" {
			if _, err := os.Stat(filepath.Join(parent, "index.html")); err == nil {
				return parent
			}
		}
		// Fallback to CWD even if index.html not found
		return cwd
	}
	// Fallback to container default if CWD is unavailable
	return "/app/site"
}

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

// ---------------- Admin Basic Auth ----------------
func adminCreds() (string, string) {
	u := os.Getenv("ADMIN_USER")
	p := os.Getenv("ADMIN_PASS")
	if u == "" {
		u = "info@tdvorak.dev"
	}
	if p == "" {
		p = "%8s3Yad*!b3*t"
	}
	return u, p
}

func checkBasicAuth(r *http.Request) bool {
	user, pass := adminCreds()
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}
	b64 := strings.TrimPrefix(auth, "Basic ")
	dec, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return false
	}
	parts := strings.SplitN(string(dec), ":", 2)
	if len(parts) != 2 {
		return false
	}
	return parts[0] == user && parts[1] == pass
}

func requireBasicAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Basic realm=admin")
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkBasicAuth(r) {
			requireBasicAuth(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Compute next blog numeric ID by scanning blog/*.html
func nextBlogID(siteRoot string) (int, error) {
	blogDir := filepath.Join(siteRoot, "blog")
	entries, err := os.ReadDir(blogDir)
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile(`^(\d{4})\.html$`)
	max := -1
	for _, e := range entries {
		m := re.FindStringSubmatch(e.Name())
		if len(m) == 2 {
			if n, err := strconv.Atoi(m[1]); err == nil && n > max {
				max = n
			}
		}
	}
	if max < 0 {
		return 0, fmt.Errorf("no posts found to derive id")
	}
	return max + 1, nil
}

// Minimal HTML escaper for title
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// ---------------- YouTube: periodic refresh and persistence ----------------
func videosScheduler(ctx context.Context) {
	// Refresh once a day
	for {
		select {
		case <-time.After(24 * time.Hour):
			if err := refreshVideos(ctx); err != nil {
				log.Printf("videos refresh error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func refreshVideos(ctx context.Context) error {
	client := &http.Client{Timeout: 30 * time.Second}
	base := "https://youtube.tdvorak.dev"
	ch := ytChannel()
	u := base + "/channel_videos?channel=" + url.QueryEscape(ch)
	var resp YTChannelResp
	if err := getJSON(ctx, client, u, &resp); err != nil {
		return fmt.Errorf("yt get: %w", err)
	}
	items := resp.Videos
	if len(items) > 5 {
		items = items[:5]
	}
	vc.mu.Lock()
	vc.data.FetchedAt = time.Now()
	vc.data.Channel = resp.Channel
	vc.data.Items = items
	vc.mu.Unlock()
	if err := writeVideosJSON(); err != nil {
		log.Printf("warn: write videos json: %v", err)
	}
	return nil
}

func writeVideosJSON() error {
	path := videosPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	vc.mu.RLock()
	payload := struct {
		FetchedAt time.Time `json:"fetched_at"`
		Channel   string    `json:"channel"`
		Items     []YTVideo `json:"items"`
	}{FetchedAt: vc.data.FetchedAt, Channel: vc.data.Channel, Items: vc.data.Items}
	vc.mu.RUnlock()
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	// Also mirror to static site path for fast fetch by frontend
	// Attempt best-effort; log warnings but do not fail the main write
	// Target files: <STATIC_PATH>/data/videos.json and <STATIC_PATH>/data/video.json
	func() {
		defer func() { _ = recover() }()
		sp := staticPath()
		if sp == "" {
			return
		}
		dataDir := filepath.Join(sp, "data")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Printf("warn: mkdir static data dir: %v", err)
			return
		}
		// Write videos.json
		dest1 := filepath.Join(dataDir, "videos.json")
		if err := os.WriteFile(dest1, b, 0644); err != nil {
			log.Printf("warn: write static videos.json: %v", err)
		}
		// Write video.json (alias) to match frontend expectation
		dest2 := filepath.Join(dataDir, "video.json")
		if err := os.WriteFile(dest2, b, 0644); err != nil {
			log.Printf("warn: write static video.json: %v", err)
		}
	}()
	return nil
}

func videosPath() string {
	if p := os.Getenv("VIDEOS_PATH"); p != "" {
		return p
	}
	// Default: Docker volume path (writable)
	return "/app/data/videos.json"
}

func ytChannel() string {
	if c := os.Getenv("YT_CHANNEL"); c != "" {
		return c
	}
	// Default YouTube channel
	return "@FCBizoniUH"
}

// BlogItem represents a simple blog card item for the homepage
type BlogItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Image       string    `json:"image"`
	MTime       time.Time `json:"mtime"`
	Categories  []string  `json:"categories,omitempty"`
}

func extractCategories(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	s := string(b)
	// match <meta name="category" content="Category">
	re := regexp.MustCompile(`(?is)<meta name="category" content="([^"]+)"`)
	matches := re.FindAllStringSubmatch(s, -1)
	categories := make([]string, len(matches))
	for i, match := range matches {
		categories[i] = match[1]
	}
	return categories
}

// listLatestBlogs scans the blog and image folders under the provided site root and returns the latest N posts
func listLatestBlogs(siteRoot string, limit int) ([]BlogItem, error) {
	blogDir := filepath.Join(siteRoot, "blog")
	imgDir := filepath.Join(siteRoot, "img", "blog")
	entries, err := os.ReadDir(blogDir)
	if err != nil {
		return nil, fmt.Errorf("readdir blog: %w", err)
	}
	re := regexp.MustCompile(`^(\d{4})\.html$`)
	var items []BlogItem
	for _, e := range entries {
		name := e.Name()
		if !re.MatchString(name) {
			continue
		}
		id := strings.TrimSuffix(name, ".html")
		// Title and categories extraction from blog HTML
		blogPath := filepath.Join(blogDir, name)
		title := extractTitle(blogPath)
		cats := extractCategories(blogPath)
		// Determine mod time - prefer image modtime if exists, else html
		mtime := time.Time{}
		htmlInfo, err1 := os.Stat(filepath.Join(blogDir, name))
		if err1 == nil {
			mtime = htmlInfo.ModTime()
		}
		if imgInfo, err2 := os.Stat(filepath.Join(imgDir, id+".png")); err2 == nil {
			// If image is newer, use that as a proxy for recency
			if imgInfo.ModTime().After(mtime) {
				mtime = imgInfo.ModTime()
			}
		}
		items = append(items, BlogItem{
			ID:          id,
			Title:       title,
			Link:        "/blog/" + id + ".html",
			Image:       "/img/blog/" + id + ".png",
			MTime:       mtime,
			Categories:  cats,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		// Descending by mod time, fallback to numeric ID desc
		if !items[i].MTime.Equal(items[j].MTime) {
			return items[i].MTime.After(items[j].MTime)
		}
		ii, _ := strconv.Atoi(items[i].ID)
		jj, _ := strconv.Atoi(items[j].ID)
		return ii > jj
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// extractTitle finds the first <h1>...</h1> and returns its inner text (very simple, best-effort)
func extractTitle(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := string(b)
	// match <h1 ...>Title</h1>
	re := regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`) // non-greedy
	m := re.FindStringSubmatch(s)
	if len(m) >= 2 {
		// strip HTML tags inside, if any
		inner := m[1]
		// remove any nested tags crudely
		inner = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(inner, "")
		inner = strings.TrimSpace(inner)
		return inner
	}
	return ""
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

// ---- YouTube videos cache ----
type YTVideo struct {
	VideoID       string `json:"video_id"`
	Title         string `json:"title"`
	Length        string `json:"length"`
	ThumbnailURL  string `json:"thumbnail_url"`
	ViewsText     string `json:"views_text"`
	Views         int    `json:"views"`
	PublishedText string `json:"published_text"`
	PublishedDate string `json:"published_date"`
}

type YTChannelResp struct {
	Channel         string    `json:"channel"`
	ChannelURL      string    `json:"channel_url"`
	SubscribersText string    `json:"subscribers_text"`
	Subscribers     int       `json:"subscribers"`
	Videos          []YTVideo `json:"videos"`
}

type videosCache struct {
	mu   sync.RWMutex
	data struct {
		FetchedAt time.Time `json:"fetched_at"`
		Channel   string    `json:"channel"`
		Items     []YTVideo `json:"items"`
	}
}

var vc videosCache

// simple in-memory rate limiter for manual videos refresh
type rateLimiter struct {
	mu   sync.Mutex
	hits []time.Time
}

func (rl *rateLimiter) Allow(now time.Time, limit int, per time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	// drop timestamps older than window
	cutoff := now.Add(-per)
	i := 0
	for _, t := range rl.hits {
		if t.After(cutoff) {
			rl.hits[i] = t
			i++
		}
	}
	rl.hits = rl.hits[:i]
	if len(rl.hits) >= limit {
		return false
	}
	rl.hits = append(rl.hits, now)
	return true
}

var videosPostLimiter rateLimiter

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
	go videosScheduler(ctx)

	// Initial videos fetch on startup to warm cache
	if err := refreshVideos(ctx); err != nil {
		log.Printf("initial videos refresh error: %v", err)
	}

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

	// Blog API: latest N posts from filesystem
	mux.HandleFunc("/api/blog/latest", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		limit := 5
		if q := r.URL.Query().Get("limit"); q != "" {
			if n, err := strconv.Atoi(q); err == nil && n > 0 {
				limit = n
			}
		}
		items, err := listLatestBlogs(staticPath(), limit)
		if err != nil {
			log.Printf("listLatestBlogs error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(items)
	})

	// Videos API
	mux.HandleFunc("/api/videos/latest", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		switch r.Method {
		case http.MethodGet:
			vc.mu.RLock()
			items := vc.data.Items
			fetched := vc.data.FetchedAt
			channel := vc.data.Channel
			vc.mu.RUnlock()
			if len(items) == 0 {
				// lazy refresh if empty
				if err := refreshVideos(r.Context()); err != nil {
					log.Printf("refreshVideos error: %v", err)
				}
				vc.mu.RLock()
				items = vc.data.Items
				fetched = vc.data.FetchedAt
				channel = vc.data.Channel
				vc.mu.RUnlock()
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(struct {
				FetchedAt time.Time `json:"fetched_at"`
				Channel   string    `json:"channel"`
				Items     []YTVideo `json:"items"`
			}{FetchedAt: fetched, Channel: channel, Items: items})
		case http.MethodPost:
			// rate limit: 5 requests per minute for manual refresh
			if !videosPostLimiter.Allow(time.Now(), 5, time.Minute) {
				http.Error(w, "rate limit: max 5 refresh per minute", http.StatusTooManyRequests)
				return
			}
			if err := refreshVideos(r.Context()); err != nil {
				log.Printf("manual refreshVideos error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Serve raw persisted videos json for debugging/preview
	mux.HandleFunc("/data/videos.json", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		b, err := os.ReadFile(videosPath())
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(b)
	})

	// Blog creation API (admin)
	mux.HandleFunc("/api/blog/new", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		if !checkBasicAuth(r) {
			requireBasicAuth(w)
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Expect multipart form with: title, content (HTML), image (png), categories (comma-separated)
		if err := r.ParseMultipartForm(20 << 20); err != nil { // 20MB
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		title := strings.TrimSpace(r.FormValue("title"))
		htmlContent := strings.TrimSpace(r.FormValue("content"))
		catsRaw := strings.TrimSpace(r.FormValue("categories"))
		var cats []string
		if catsRaw != "" {
			for _, p := range strings.Split(catsRaw, ",") {
				c := strings.TrimSpace(p)
				if c != "" {
					cats = append(cats, c)
				}
			}
		}
		if title == "" || htmlContent == "" {
			http.Error(w, "missing title or content", http.StatusBadRequest)
			return
		}
		f, fh, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "missing image", http.StatusBadRequest)
			return
		}
		defer f.Close()
		// Accept PNG/JPG/JPEG; always store normalized PNG 1600x969
		name := strings.ToLower(fh.Filename)
		if !(strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg")) {
			http.Error(w, "image must be .png, .jpg, or .jpeg", http.StatusBadRequest)
			return
		}
		site := staticPath()
		// Determine next ID
		nid, err := nextBlogID(site)
		if err != nil {
			http.Error(w, "failed to compute next id", http.StatusInternalServerError)
			return
		}
		idStr := fmt.Sprintf("%04d", nid)
		// Write image (normalize to 1600x969 with letterboxing)
		imgDir := filepath.Join(site, "img", "blog")
		if err := os.MkdirAll(imgDir, 0755); err != nil {
			http.Error(w, "storage error: img dir", http.StatusInternalServerError)
			return
		}
		imgPath := filepath.Join(imgDir, idStr+".png")
		if err := normalizeBlogImage(f, imgPath); err != nil {
			http.Error(w, "image processing failed", http.StatusInternalServerError)
			return
		}
		// Read template and replace
		tplPath := filepath.Join(site, "blog", "0030.html")
		tplBytes, err := os.ReadFile(tplPath)
		if err != nil {
			http.Error(w, "template not found", http.StatusInternalServerError)
			return
		}
		s := string(tplBytes)
		// Replace H1 title inside page header (match when lte-header is among multiple classes)
		reH1 := regexp.MustCompile(`(?is)<h1[^>]*class="[^"]*\blte-header\b[^"]*"[^>]*>.*?</h1>`)
		s = reH1.ReplaceAllString(s, "<h1 class=\"lte-header\">"+htmlEscape(title)+"</h1>")
		// Replace main hero image to point to new id
		reImg := regexp.MustCompile(`(?is)src=\"\.\./img/blog/\d{4}\.png\"`)
		s = reImg.ReplaceAllString(s, "src=\"../img/blog/"+idStr+".png\"")
		// Replace post top image similarly if found with different quoting
		reImg2 := regexp.MustCompile(`(?is)src=\"\.\./img/blog/\d{4}\.png\"`)
		s = reImg2.ReplaceAllString(s, "src=\"../img/blog/"+idStr+".png\"")
		// Replace the main content inside <div class="text lte-text-page clearfix"> ... </div>
		reContent := regexp.MustCompile(`(?is)<div class="text lte-text-page clearfix">[\s\S]*?</div>`)
		s = reContent.ReplaceAllString(s, "<div class=\"text lte-text-page clearfix\">\n"+htmlContent+"\n</div>")
		// Inject categories as <meta name="category" content="..."> before </head>
		if len(cats) > 0 {
			var meta string
			for _, c := range cats {
				meta += "<meta name=\"category\" content=\"" + htmlEscape(c) + "\">\n"
			}
			reHead := regexp.MustCompile(`(?is)</head>`) 
			s = reHead.ReplaceAllString(s, meta+"</head>")
		}
		// Write new blog html
		blogDir := filepath.Join(site, "blog")
		if err := os.MkdirAll(blogDir, 0755); err != nil {
			http.Error(w, "storage error: blog dir", http.StatusInternalServerError)
			return
		}
		htmlPath := filepath.Join(blogDir, idStr+".html")
		if err := os.WriteFile(htmlPath, []byte(s), 0644); err != nil {
			http.Error(w, "cannot write blog (is STATIC_PATH read-only?)", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      idStr,
			"link":    "/blog/" + idStr + ".html",
			"image":   "/img/blog/" + idStr + ".png",
			"message": "created",
		})
	})

	// Blog fetch (admin): returns title, content html, image for editing
	mux.HandleFunc("/api/blog/get", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		if !checkBasicAuth(r) {
			requireBasicAuth(w)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		re := regexp.MustCompile(`^\d{4}$`)
		if !re.MatchString(id) {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		path := filepath.Join(staticPath(), "blog", id+".html")
		b, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		s := string(b)
		reH1 := regexp.MustCompile(`(?is)<h1[^>]*class="[^"]*\blte-header\b[^"]*"[^>]*>(.*?)</h1>`)
		m := reH1.FindStringSubmatch(s)
		title := ""
		if len(m) >= 2 {
			// strip HTML tags inside, if any
			inner := m[1]
			// remove any nested tags crudely
			inner = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(inner, "")
			inner = strings.TrimSpace(inner)
			title = inner
		}
		reContent := regexp.MustCompile(`(?is)<div class="text lte-text-page clearfix">([\s\S]*?)</div>`)
		mc := reContent.FindStringSubmatch(s)
		content := ""
		if len(mc) >= 2 {
			content = strings.TrimSpace(mc[1])
		}
		cats := extractCategories(path)
		img := "/img/blog/" + id + ".png"
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "title": title, "content_html": content, "image": img, "categories": cats})
	})

	// Blog edit (admin): update title/content and optionally replace image
	mux.HandleFunc("/api/blog/edit", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		if !checkBasicAuth(r) {
			requireBasicAuth(w)
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Expect multipart form with: title, content (HTML), image (png), categories (comma-separated)
		if err := r.ParseMultipartForm(25 << 20); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		id := strings.TrimSpace(r.FormValue("id"))
		re := regexp.MustCompile(`^\d{4}$`)
		if !re.MatchString(id) {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		title := strings.TrimSpace(r.FormValue("title"))
		htmlContent := strings.TrimSpace(r.FormValue("content"))
		catsRaw := strings.TrimSpace(r.FormValue("categories"))
		var cats []string
		if catsRaw != "" {
			for _, p := range strings.Split(catsRaw, ",") {
				c := strings.TrimSpace(p)
				if c != "" {
					cats = append(cats, c)
				}
			}
		}
		if title == "" || htmlContent == "" {
			http.Error(w, "missing title or content", http.StatusBadRequest)
			return
		}
		site := staticPath()
		if f, fh, err := r.FormFile("image"); err == nil {
			defer f.Close()
			// Accept PNG/JPG/JPEG; always store normalized PNG 1600x969
			name := strings.ToLower(fh.Filename)
			if !(strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg")) {
				http.Error(w, "image must be .png, .jpg, or .jpeg", http.StatusBadRequest)
				return
			}
			imgPath := filepath.Join(site, "img", "blog", id+".png")
			if err := os.MkdirAll(filepath.Dir(imgPath), 0755); err != nil {
				http.Error(w, "storage error", http.StatusInternalServerError)
				return
			}
			if err := normalizeBlogImage(f, imgPath); err != nil {
				http.Error(w, "image processing failed", http.StatusInternalServerError)
				return
			}
		}
		hPath := filepath.Join(site, "blog", id+".html")
		b, err := os.ReadFile(hPath)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		s := string(b)
		reH1 := regexp.MustCompile(`(?is)<h1[^>]*class="lte-header"[^>]*>.*?</h1>`)
		s = reH1.ReplaceAllString(s, "<h1 class=\"lte-header\">"+htmlEscape(title)+"</h1>")
		reContent := regexp.MustCompile(`(?is)<div class="text lte-text-page clearfix">[\s\S]*?</div>`)
		s = reContent.ReplaceAllString(s, "<div class=\"text lte-text-page clearfix\">\n"+htmlContent+"\n</div>")
		// Replace categories meta tags
		reMeta := regexp.MustCompile(`(?is)<meta name="category" content="[^"]*"\s*/?>\s*`)
		s = reMeta.ReplaceAllString(s, "")
		if len(cats) > 0 {
			var meta string
			for _, c := range cats {
				meta += "<meta name=\"category\" content=\"" + htmlEscape(c) + "\">\n"
			}
			reHead := regexp.MustCompile(`(?is)</head>`) 
			s = reHead.ReplaceAllString(s, meta+"</head>")
		}
		if err := os.WriteFile(hPath, []byte(s), 0644); err != nil {
			http.Error(w, "cannot write", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// Blog delete (admin)
	mux.HandleFunc("/api/blog/delete", func(w http.ResponseWriter, r *http.Request) {
		okCORS(w)
		if !checkBasicAuth(r) {
			requireBasicAuth(w)
			return
		}
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		re := regexp.MustCompile(`^\d{4}$`)
		if !re.MatchString(id) {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		site := staticPath()
		_ = os.Remove(filepath.Join(site, "blog", id+".html"))
		_ = os.Remove(filepath.Join(site, "img", "blog", id+".png"))
		w.WriteHeader(http.StatusNoContent)
	})

    // Static file server for the frontend
    sp := staticPath()
    log.Printf("serving static from: %s", sp)
    fs := http.FileServer(http.Dir(sp))
    // Serve common asset prefixes explicitly
    mux.Handle("/img/", fs)
    mux.Handle("/css/", fs)
    mux.Handle("/js/", fs)
    // Protect /admin/ with Basic Auth
    mux.Handle("/admin/", basicAuth(http.StripPrefix("/admin/", http.FileServer(http.Dir(filepath.Join(staticPath(), "admin"))))))
    mux.Handle("/blog/", fs)
    mux.Handle("/zapasy/", fs)
    // Fallback: serve index.html at root, otherwise delegate to static file server
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/" || r.URL.Path == "/index.html" {
            http.ServeFile(w, r, filepath.Join(sp, "index.html"))
            return
        }
        fs.ServeHTTP(w, r)
    })

    port := os.Getenv("PORT")
    if port == "" { port = "8080" }
    srv := &http.Server{
        Addr:    ":" + port,
        Handler: mux,
    }
    go func() {
        log.Printf("server listening on :%s", port)
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
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
