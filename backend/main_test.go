package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"
)

// TestListLatestBlogsOrdering verifies that listLatestBlogs returns items
// sorted by numeric ID descending, regardless of file timestamps or order.
func TestListLatestBlogsOrdering(t *testing.T) {
	// Create a temp directory structure mimicking the remote server
	tmpDir := t.TempDir()
	blogDir := filepath.Join(tmpDir, "blog")
	imgDir := filepath.Join(tmpDir, "img", "blog")
	if err := os.MkdirAll(blogDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create numeric blog files with IDs spanning a wide range.
	// We intentionally create them in non-numeric order and touch
	// old IDs with newer timestamps to simulate a migration.
	files := []struct {
		id    string
		slug  string
		title string
	}{
		{"0031", "vstupujeme-spolecne-do-druhe-ligy", "VSTUPUJEME SPOLEČNĚ DO DRUHÉ LIGY!"},
		{"0032", "nova-mise-pred-nami", "NOVÁ MISE PŘED NÁMI!"},
		{"0033", "superpohar-divizi-je-zde", "SUPERPOHÁR DIVIZÍ JE ZDE!"},
		{"0034", "superpohar-je-nas", "SUPERPOHÁR JE NÁŠ!"},
		{"0035", "fotoreport-1", "FOTOREPORT"},
		{"0036", "regionalni-finale-je-tady", "REGIONÁLNÍ FINÁLE JE TADY!"},
		{"0037", "bizoni-slavi-postup", "BIZONI SLAVÍ POSTUP!"},
		{"0038", "fotoreport-2", "FOTOREPORT"},
		{"0039", "2-liga-je-tu", "2. LIGA JE TU!"},
		{"0040", "pred-startem-sezony-1", "PŘED STARTEM SEZONY"},
		{"0041", "pred-startem-sezony-2", "PŘED STARTEM SEZÓNY"},
		{"0042", "podpora-futsalu", "Podpora Futsalu"},
		{"0169", "stepan-stodulka-fanouskum-3", "Štěpán Stodůlka fanouškům: Budujeme klub, který bude dlouhodobě silný"},
		{"0170", "martin-prokes-fanouskum", "Martin Prokeš fanouškům: První futsalová sezóna přinesla cenné zkušenosti"},
		{"0171", "andrea-adamikova-fanouskum", "Andrea Adamíková fanouškům: Druhé místo je motivací do další práce"},
		{"0172", "stepan-stodulka-fanouskum-2", "Štěpán Stodůlka fanouškům: Bizonky jsou hrdou součástí našeho klubu"},
		{"0173", "martin-lapcik-fanouskum", "Martin Lapčík fanouškům: První rok bizoní mládeže nás všechny nadchl"},
		{"0174", "marek-stojaspal-fanouskum", "Marek Stojaspal fanouškům: Mládež položila pevné základy budoucnosti"},
		{"0175", "stepan-stodulka-fanouskum-1", "Štěpán Stodůlka fanouškům: Mládež ukázala velký potenciál"},
		{"0176", "dekujeme-my-jsme-tu-diky-vam", "DĚKUJEME, MY JSME TU DÍKY VÁM!"},
	}

	for _, f := range files {
		// Create numeric HTML file with slug meta tag
		htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta name="slug" content="%s">
</head>
<body>
<h1 class="lte-header">%s</h1>
<div class="text lte-text-page clearfix">content</div>
</body>
</html>`, f.slug, f.title)
		numericPath := filepath.Join(blogDir, f.id+".html")
		if err := os.WriteFile(numericPath, []byte(htmlContent), 0644); err != nil {
			t.Fatal(err)
		}
		// Create corresponding slug file (duplicate content)
		slugPath := filepath.Join(blogDir, f.slug+".html")
		if err := os.WriteFile(slugPath, []byte(htmlContent), 0644); err != nil {
			t.Fatal(err)
		}
		// Create a dummy image
		imgPath := filepath.Join(imgDir, f.id+".png")
		if err := os.WriteFile(imgPath, []byte("fake png"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Touch some old IDs with a newer timestamp to simulate post-migration
	// (this is the exact condition that broke the old sorting).
	// We sleep briefly between creation and touch so the timestamp is definitely newer.
	importTime := time.Now()
	for _, oldID := range []string{"0031", "0032", "0042"} {
		fpath := filepath.Join(blogDir, oldID+".html")
		if err := os.Chtimes(fpath, importTime, importTime); err != nil {
			// ignore errors on Chtimes
		}
	}

	// Call listLatestBlogs
	items, err := listLatestBlogs(tmpDir, 12)
	if err != nil {
		t.Fatalf("listLatestBlogs error: %v", err)
	}
	if len(items) != 12 {
		t.Fatalf("expected 12 items, got %d", len(items))
	}

	// Verify IDs are sorted descending (newest first)
	for i := 0; i < len(items)-1; i++ {
		curr, _ := strconv.Atoi(items[i].ID)
		next, _ := strconv.Atoi(items[i+1].ID)
		if curr <= next {
			t.Errorf("IDs not sorted descending at index %d: %s (%d) <= %s (%d)",
				i, items[i].ID, curr, items[i+1].ID, next)
		}
	}

	// Verify the first item is the newest (0176)
	if items[0].ID != "0176" {
		t.Errorf("expected first item ID to be 0176, got %s (title: %s)", items[0].ID, items[0].Title)
	}
	if items[1].ID != "0175" {
		t.Errorf("expected second item ID to be 0175, got %s", items[1].ID)
	}

	// Verify deduplication: total unique posts should be 20 (not 40)
	allItems, err := listLatestBlogs(tmpDir, 0)
	if err != nil {
		t.Fatalf("listLatestBlogs unlimited error: %v", err)
	}
	if len(allItems) != 20 {
		t.Errorf("expected 20 unique items after dedup, got %d", len(allItems))
	}

	// Verify all items are sorted descending
	sorted := sort.SliceIsSorted(allItems, func(i, j int) bool {
		ii, _ := strconv.Atoi(allItems[i].ID)
		jj, _ := strconv.Atoi(allItems[j].ID)
		return ii > jj
	})
	if !sorted {
		t.Error("allItems are not sorted by numeric ID descending")
	}
}

// TestListLatestBlogsNoSlug verifies numeric-only blogs still sort correctly.
func TestListLatestBlogsNoSlug(t *testing.T) {
	tmpDir := t.TempDir()
	blogDir := filepath.Join(tmpDir, "blog")
	imgDir := filepath.Join(tmpDir, "img", "blog")
	os.MkdirAll(blogDir, 0755)
	os.MkdirAll(imgDir, 0755)

	for _, id := range []string{"0001", "0005", "0010", "0002"} {
		content := fmt.Sprintf(`<html><head></head><body><h1 class="lte-header">Title %s</h1></body></html>`, id)
		os.WriteFile(filepath.Join(blogDir, id+".html"), []byte(content), 0644)
		os.WriteFile(filepath.Join(imgDir, id+".png"), []byte("png"), 0644)
	}

	items, err := listLatestBlogs(tmpDir, 0)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"0010", "0005", "0002", "0001"}
	if len(items) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(items))
	}
	for i, exp := range expected {
		if items[i].ID != exp {
			t.Errorf("index %d: expected %s, got %s", i, exp, items[i].ID)
		}
	}
}
