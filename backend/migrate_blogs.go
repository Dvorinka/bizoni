//go:build ignore

// Migration script to clean up duplicate blog files
// Run with: go run migrate_blogs.go <blog_directory>
//
// This script:
// 1. Scans all blog HTML files
// 2. Groups numeric files with their slug counterparts
// 3. Removes orphan slug files (slugs without matching numeric files)
// 4. Reports duplicates for manual review

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run migrate_blogs.go <blog_directory>")
		fmt.Println("Example: go run migrate_blogs.go ../blog")
		os.Exit(1)
	}

	blogDir := os.Args[1]

	// Check directory exists
	if _, err := os.Stat(blogDir); os.IsNotExist(err) {
		fmt.Printf("Error: blog directory not found: %s\n", blogDir)
		os.Exit(1)
	}

	// Read all files
	entries, err := os.ReadDir(blogDir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	numericRe := regexp.MustCompile(`^\d{4}\.html$`)
	slugRe := regexp.MustCompile(`^[a-z0-9-]+\.html$`)

	// Map: numeric ID -> slug (extracted from file content)
	numericToSlug := make(map[string]string)
	// Map: slug -> numeric ID (extracted by matching content)
	slugToNumeric := make(map[string]string)
	// List of orphan slug files (no matching numeric file)
	orphanSlugs := []string{}
	// List of numeric files
	numericFiles := []string{}
	// List of slug files
	slugFiles := []string{}

	// First pass: categorize files
	for _, e := range entries {
		name := e.Name()
		if numericRe.MatchString(name) {
			numericFiles = append(numericFiles, name)
		} else if slugRe.MatchString(name) {
			slugFiles = append(slugFiles, name)
		}
	}

	fmt.Printf("Found %d numeric files and %d slug files\n", len(numericFiles), len(slugFiles))

	// Extract slugs from numeric files
	for _, name := range numericFiles {
		path := filepath.Join(blogDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		slug := extractSlugFromContent(string(content))
		id := strings.TrimSuffix(name, ".html")
		if slug != "" {
			numericToSlug[id] = slug
		}
	}

	// Check slug files for matches
	for _, name := range slugFiles {
		slug := strings.TrimSuffix(name, ".html")
		foundMatch := false

		// Check if any numeric file has this slug
		for numericID, numericSlug := range numericToSlug {
			if numericSlug == slug {
				slugToNumeric[slug] = numericID
				foundMatch = true
				break
			}
		}

		if !foundMatch {
			orphanSlugs = append(orphanSlugs, name)
		}
	}

	// Report findings
	fmt.Println("\n=== Blog Migration Report ===")
	fmt.Printf("\nNumeric files with slugs:\n")
	for id, slug := range numericToSlug {
		fmt.Printf("  %s -> %s\n", id, slug)
	}

	fmt.Printf("\nSlug files with matching numeric:\n")
	for slug, id := range slugToNumeric {
		fmt.Printf("  %s.html -> %s.html\n", slug, id)
	}

	if len(orphanSlugs) > 0 {
		fmt.Printf("\nOrphan slug files (no matching numeric file):\n")
		for _, name := range orphanSlugs {
			fmt.Printf("  %s\n", name)
		}

		fmt.Printf("\nRemove %d orphan slug files? (y/n): ", len(orphanSlugs))
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) == "y" {
			for _, name := range orphanSlugs {
				path := filepath.Join(blogDir, name)
				if err := os.Remove(path); err != nil {
					fmt.Printf("  Error removing %s: %v\n", name, err)
				} else {
					fmt.Printf("  Removed: %s\n", name)
				}
			}
			fmt.Println("Migration complete!")
		} else {
			fmt.Println("Migration cancelled.")
		}
	} else {
		fmt.Println("\nNo orphan slug files found. Blog directory is clean.")
	}
}

func extractSlugFromContent(htmlContent string) string {
	re := regexp.MustCompile(`(?is)<meta name="slug" content="([^"]+)"`)
	m := re.FindStringSubmatch(htmlContent)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}
