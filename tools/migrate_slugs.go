package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./migrate_slugs.go <site_root>")
		fmt.Println("Example: ./migrate_slugs.go /home/tdvorak/Desktop/HTML_Projekty/bizoni")
		os.Exit(1)
	}

	siteRoot := os.Args[1]
	blogDir := filepath.Join(siteRoot, "blog")

	// Read all blog files
	entries, err := os.ReadDir(blogDir)
	if err != nil {
		log.Fatalf("Failed to read blog directory: %v", err)
	}

	// Pattern to match numeric blog files
	numericPattern := regexp.MustCompile(`^(\d{4})\.html$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		matches := numericPattern.FindStringSubmatch(filename)
		if len(matches) != 2 {
			continue // Skip non-numeric files
		}

		blogPath := filepath.Join(blogDir, filename)

		// Read the blog file
		content, err := os.ReadFile(blogPath)
		if err != nil {
			log.Printf("Failed to read %s: %v", filename, err)
			continue
		}

		contentStr := string(content)

		// Check if slug already exists
		slugPattern := regexp.MustCompile(`(?is)<meta name="slug" content="([^"]+)"`)
		if slugPattern.MatchString(contentStr) {
			log.Printf("Skipping %s - slug already exists", filename)
			continue
		}

		// Extract title
		titlePattern := regexp.MustCompile(`(?is)<h1[^>]*class="[^"]*\blte-header\b[^"]*"[^>]*>(.*?)</h1>`)
		titleMatches := titlePattern.FindStringSubmatch(contentStr)
		if len(titleMatches) < 2 {
			log.Printf("Skipping %s - could not find title", filename)
			continue
		}

		title := titleMatches[1]
		// Remove any HTML tags from title
		title = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(title, "")
		title = strings.TrimSpace(title)

		// Generate slug
		slug := generateSlug(title)
		slug = ensureUniqueSlug(siteRoot, slug)

		// Find where to insert the slug meta tag (before </head>)
		headPattern := regexp.MustCompile(`(?is)</head>`)
		if !headPattern.MatchString(contentStr) {
			log.Printf("Skipping %s - could not find </head> tag", filename)
			continue
		}

		// Insert slug meta tag
		slugMeta := fmt.Sprintf(`<meta name="slug" content="%s">`, slug)
		newContent := headPattern.ReplaceAllString(contentStr, slugMeta+"\n</head>")

		// Write the updated content to both files
		err = os.WriteFile(blogPath, []byte(newContent), 0644)
		if err != nil {
			log.Printf("Failed to write updated %s: %v", filename, err)
			continue
		}

		// Create slug-based file
		slugPath := filepath.Join(blogDir, slug+".html")
		err = os.WriteFile(slugPath, []byte(newContent), 0644)
		if err != nil {
			log.Printf("Failed to create slug file %s: %v", slugPath, err)
			continue
		}

		log.Printf("Processed %s: title='%s' -> slug='%s'", filename, title, slug)
	}

	log.Println("Migration completed!")
}

// generateSlug creates a URL-friendly slug from a title
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	// Replace Czech characters with their ASCII equivalents
	replacements := map[string]string{
		"á": "a", "ä": "a", "č": "c", "ď": "d", "é": "e", "ě": "e", "í": "i", "ľ": "l",
		"ň": "n", "ó": "o", "ö": "o", "ô": "o", "ř": "r", "š": "s", "ť": "t", "ú": "u",
		"ů": "u", "ý": "y", "ž": "z",
		"Á": "a", "Ä": "a", "Č": "c", "Ď": "d", "É": "e", "Ě": "e", "Í": "i", "Ľ": "l",
		"Ň": "n", "Ó": "o", "Ö": "o", "Ô": "o", "Ř": "r", "Š": "s", "Ť": "t", "Ú": "u",
		"Ů": "u", "Ý": "y", "Ž": "z",
	}
	for czech, ascii := range replacements {
		slug = strings.ReplaceAll(slug, czech, ascii)
	}
	// Remove any character that isn't alphanumeric, space, or hyphen
	re := regexp.MustCompile(`[^a-z0-9\s-]`)
	slug = re.ReplaceAllString(slug, "")
	// Replace spaces and multiple hyphens with a single hyphen
	re = regexp.MustCompile(`[\s-]+`)
	slug = re.ReplaceAllString(slug, "-")
	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")
	return slug
}

// ensureUniqueSlug ensures the slug is unique by appending a number if needed
func ensureUniqueSlug(siteRoot, baseSlug string) string {
	blogDir := filepath.Join(siteRoot, "blog")
	entries, err := os.ReadDir(blogDir)
	if err != nil {
		return baseSlug
	}
	existingSlugs := make(map[string]bool)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		// Extract slug from filename if it follows the new pattern
		name := strings.TrimSuffix(e.Name(), ".html")
		// Check if it's a slug-based filename (contains letters, not just numbers)
		if regexp.MustCompile(`[a-z]`).MatchString(name) {
			existingSlugs[name] = true
		}
	}
	if !existingSlugs[baseSlug] {
		return baseSlug
	}
	// Try baseSlug-2, baseSlug-3, etc.
	for i := 2; i < 100; i++ {
		testSlug := fmt.Sprintf("%s-%d", baseSlug, i)
		if !existingSlugs[testSlug] {
			return testSlug
		}
	}
	// Fallback to timestamp
	return fmt.Sprintf("%s-%d", baseSlug, 1234567890)
}
