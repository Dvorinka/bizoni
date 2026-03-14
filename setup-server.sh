#!/bin/bash

# Simple Server Setup Script - Run this ON YOUR SERVER after git push
# Usage: ./setup-server.sh

set -e

echo "🚀 Bizoni Server Setup Script"
echo "============================"

# Configuration - UPDATE THESE IF NEEDED
BLOG_DIR="/var/www/bizoni/blog"
IMG_DIR="/var/www/bizoni/img/blog"
BACKUP_DIR="/var/backups/bizoni"

echo "📁 Blog directory: $BLOG_DIR"
echo "🖼️  Image directory: $IMG_DIR"
echo ""

# Check if directories exist
if [ ! -d "$BLOG_DIR" ]; then
    echo "❌ Blog directory not found: $BLOG_DIR"
    echo "Please update BLOG_DIR in this script"
    exit 1
fi

if [ ! -d "$IMG_DIR" ]; then
    echo "❌ Image directory not found: $IMG_DIR" 
    echo "Please update IMG_DIR in this script"
    exit 1
fi

echo "✅ Directories found"
echo ""

# Create backup
echo "📦 Creating backup..."
mkdir -p "$BACKUP_DIR"
timestamp=$(date +%Y%m%d_%H%M%S)
tar -czf "$BACKUP_DIR/blogs_backup_$timestamp.tar.gz" -C "$(dirname "$BLOG_DIR")" "$(basename "$BLOG_DIR")"
echo "✅ Backup created: $BACKUP_DIR/blogs_backup_$timestamp.tar.gz"
echo ""

# Count existing blogs
total_blogs=$(ls "$BLOG_DIR"/*.html 2>/dev/null | wc -l)
numeric_blogs=$(ls "$BLOG_DIR"/[0-9][0-9][0-9][0-9].html 2>/dev/null | wc -l)
slug_blogs=$(ls "$BLOG_DIR"/[a-z]*.html 2>/dev/null | wc -l)

echo "📊 Current blog status:"
echo "  Total blogs: $total_blogs"
echo "  Numeric files: $numeric_blogs" 
echo "  Slug files: $slug_blogs"
echo ""

# Function to generate slug from title
generate_slug() {
    local title="$1"
    echo "$title" | tr '[:upper:]' '[:lower:]' | \
        sed 's/á/a/g; s/ä/a/g; s/č/c/g; s/ď/d/g; s/é/e/g; s/ě/e/g; s/í/i/g; s/ľ/l/g; s/ň/n/g; s/ó/o/g; s/ö/o/g; s/ô/o/g; s/ř/r/g; s/š/s/g; s/ť/t/g; s/ú/u/g; s/ů/u/g; s/ý/y/g; s/ž/z/g' | \
        sed 's/Á/a/g; s/Ä/a/g; s/Č/c/g; s/Ď/d/g; s/É/e/g; s/Ě/e/g; s/Í/i/g; s/Ľ/l/g; s/Ň/n/g; s/Ó/o/g; s/Ö/o/g; s/Ô/o/g; s/Ř/r/g; s/Š/s/g; s/Ť/t/g; s/Ú/u/g; s/Ů/u/g; s/Ý/y/g; s/Ž/z/g' | \
        sed 's/[^a-z0-9\s-]//g' | \
        sed 's/[\s-]\+/ -/g' | \
        sed 's/^-\|-$//g'
}

# Function to extract title from HTML
extract_title() {
    local file="$1"
    grep -o '<h1[^>]*class="[^"]*lte-header[^"]*"[^>]*>.*</h1>' "$file" | sed 's/<[^>]*>//g' | xargs || echo ""
}

# Function to check if slug exists
slug_exists() {
    local slug="$1"
    [ -f "$BLOG_DIR/$slug.html" ]
}

# Process numeric blogs that don't have slugs yet
echo "🔄 Processing blogs without slugs..."
processed=0

for blog_file in "$BLOG_DIR"/[0-9][0-9][0-9][0-9].html; do
    if [ ! -f "$blog_file" ]; then
        continue
    fi
    
    filename=$(basename "$blog_file")
    blog_id="${filename%.html}"
    
    # Check if slug meta tag already exists
    if grep -q '<meta name="slug"' "$blog_file"; then
        echo "  ⏭️  Skipping $filename (slug already exists)"
        continue
    fi
    
    # Extract title
    title=$(extract_title "$blog_file")
    if [ -z "$title" ]; then
        echo "  ⚠️  Skipping $filename (no title found)"
        continue
    fi
    
    # Generate slug
    base_slug=$(generate_slug "$title")
    slug="$base_slug"
    
    # Make slug unique
    counter=2
    while slug_exists "$slug"; do
        slug="${base_slug}-${counter}"
        ((counter++))
    done
    
    echo "  📝 $filename: '$title' → '$slug'"
    
    # Add slug meta tag before </head>
    sed -i "s|</head>|<meta name=\"slug\" content=\"$slug\">\n</head>|" "$blog_file"
    
    # Create slug file (copy of original)
    cp "$blog_file" "$BLOG_DIR/$slug.html"
    
    ((processed++))
done

echo ""
echo "✅ Migration completed!"
echo "📊 Processed $processed blogs"
echo ""

# Show final status
total_blogs=$(ls "$BLOG_DIR"/*.html 2>/dev/null | wc -l)
numeric_blogs=$(ls "$BLOG_DIR"/[0-9][0-9][0-9][0-9].html 2>/dev/null | wc -l)
slug_blogs=$(ls "$BLOG_DIR"/[a-z]*.html 2>/dev/null | wc -l)

echo "📊 Final blog status:"
echo "  Total blogs: $total_blogs"
echo "  Numeric files: $numeric_blogs"
echo "  Slug files: $slug_blogs"
echo ""

# Show some example URLs
echo "🌐 Example URLs now available:"
echo "  /blog/$(ls "$BLOG_DIR"/[a-z]*.html 2>/dev/null | head -1 | xargs basename -s .html || echo 'jdeme-do-finale')"
echo "  /blog/$(ls "$BLOG_DIR"/[a-z]*.html 2>/dev/null | head -2 | tail -1 | xargs basename -s .html || echo '1-zapas-final-score')"
echo ""

echo "🎉 Setup complete! Your blogs now support:"
echo "  ✅ Clean URLs (slugs)"
echo "  ✅ SEO meta tags"
echo "  ✅ Backward compatibility"
echo "  ✅ New admin features"
echo ""
echo "📝 Next steps:"
echo "  1. Restart your backend service"
echo "  2. Test new URLs in browser"
echo "  3. Try admin interface with new features"
