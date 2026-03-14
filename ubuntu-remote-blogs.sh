#!/bin/bash

# Ubuntu Server Script: Remote Blog Management
# This script manages blogs on the remote server only

set -e

# Configuration - UPDATE THESE PATHS
SERVER_BLOG_DIR="/var/www/bizoni/blog"  # Path to blogs on your server
BACKUP_DIR="/var/backups/bizoni-blogs"  # Backup location
SITE_ROOT="/var/www/bizoni"            # Site root on server

echo "🚀 Bizoni Remote Blog Management Script"
echo "====================================="

# Function to create backup
create_backup() {
    echo "📦 Creating backup..."
    mkdir -p "$BACKUP_DIR"
    timestamp=$(date +%Y%m%d_%H%M%S)
    tar -czf "$BACKUP_DIR/blogs_backup_$timestamp.tar.gz" -C "$SERVER_BLOG_DIR" .
    echo "✅ Backup created: $BACKUP_DIR/blogs_backup_$timestamp.tar.gz"
}

# Function to list blogs
list_blogs() {
    echo "📋 Current blogs on server:"
    if [ -d "$SERVER_BLOG_DIR" ]; then
        ls -la "$SERVER_BLOG_DIR"/*.html | while read line; do
            filename=$(basename "$line")
            if [[ "$filename" =~ ^[0-9]{4}\.html$ ]]; then
                echo "  📄 $filename (numeric)"
            elif [[ "$filename" =~ ^[a-z0-9-]+\.html$ ]]; then
                echo "  🔗 $filename (slug)"
            fi
        done
    else
        echo "  ❌ Blog directory not found: $SERVER_BLOG_DIR"
    fi
}

# Function to add slug to existing blog
add_slug_to_blog() {
    local blog_id="$1"
    local blog_file="$SERVER_BLOG_DIR/$blog_id.html"
    
    if [ ! -f "$blog_file" ]; then
        echo "❌ Blog file not found: $blog_file"
        return 1
    fi
    
    echo "🔧 Adding slug to blog: $blog_id"
    
    # Extract title from blog file
    title=$(grep -o '<h1[^>]*class="[^"]*lte-header[^"]*"[^>]*>.*</h1>' "$blog_file" | sed 's/<[^>]*>//g' | xargs)
    
    if [ -z "$title" ]; then
        echo "❌ Could not extract title from $blog_file"
        return 1
    fi
    
    echo "📝 Title: $title"
    
    # Generate slug
    slug=$(echo "$title" | tr '[:upper:]' '[:lower:]' | \
        sed 's/á/a/g; s/ä/a/g; s/č/c/g; s/ď/d/g; s/é/e/g; s/ě/e/g; s/í/i/g; s/ľ/l/g; s/ň/n/g; s/ó/o/g; s/ö/o/g; s/ô/o/g; s/ř/r/g; s/š/s/g; s/ť/t/g; s/ú/u/g; s/ů/u/g; s/ý/y/g; s/ž/z/g' | \
        sed 's/Á/a/g; s/Ä/a/g; s/Č/c/g; s/Ď/d/g; s/É/e/g; s/Ě/e/g; s/Í/i/g; s/Ľ/l/g; s/Ň/n/g; s/Ó/o/g; s/Ö/o/g; s/Ô/o/g; s/Ř/r/g; s/Š/s/g; s/Ť/t/g; s/Ú/u/g; s/Ů/u/g; s/Ý/y/g; s/Ž/z/g' | \
        sed 's/[^a-z0-9\s-]//g' | \
        sed 's/[\s-]\+/ -/g' | \
        sed 's/^-\|-$//g')
    
    # Check if slug file already exists
    slug_file="$SERVER_BLOG_DIR/$slug.html"
    if [ -f "$slug_file" ]; then
        # Add number suffix
        i=2
        while [ -f "$SERVER_BLOG_DIR/${slug}-${i}.html" ]; do
            ((i++))
        done
        slug="${slug}-${i}"
        slug_file="$SERVER_BLOG_DIR/${slug}.html"
    fi
    
    echo "🔗 Generated slug: $slug"
    
    # Check if slug meta tag already exists
    if grep -q '<meta name="slug"' "$blog_file"; then
        echo "ℹ️  Slug meta tag already exists"
        return 0
    fi
    
    # Add slug meta tag before </head>
    sed -i "s|</head>|<meta name=\"slug\" content=\"$slug\">\n</head>|" "$blog_file"
    
    # Create slug file (copy of original)
    cp "$blog_file" "$slug_file"
    
    echo "✅ Slug added and slug file created: $slug.html"
}

# Function to migrate all blogs to slugs
migrate_all_blogs() {
    echo "🔄 Migrating all blogs to slugs..."
    
    if [ ! -d "$SERVER_BLOG_DIR" ]; then
        echo "❌ Blog directory not found: $SERVER_BLOG_DIR"
        echo "Please update SERVER_BLOG_DIR in this script"
        exit 1
    fi
    
    create_backup
    
    # Process all numeric blog files
    for blog_file in "$SERVER_BLOG_DIR"/*.html; do
        if [ -f "$blog_file" ]; then
            filename=$(basename "$blog_file")
            if [[ "$filename" =~ ^([0-9]{4})\.html$ ]]; then
                blog_id="${BASH_REMATCH[1]}"
                add_slug_to_blog "$blog_id"
                echo ""
            fi
        fi
    done
    
    echo "✅ Migration completed!"
}

# Function to show blog info
show_blog_info() {
    local blog_id="$1"
    local blog_file="$SERVER_BLOG_DIR/$blog_id.html"
    
    if [ ! -f "$blog_file" ]; then
        echo "❌ Blog file not found: $blog_file"
        return 1
    fi
    
    echo "📄 Blog Info for: $blog_id"
    echo "========================"
    
    # Extract title
    title=$(grep -o '<h1[^>]*class="[^"]*lte-header[^"]*"[^>]*>.*</h1>' "$blog_file" | sed 's/<[^>]*>//g' | xargs)
    echo "📝 Title: $title"
    
    # Extract slug
    slug=$(grep -o '<meta name="slug" content="[^"]*"' "$blog_file" | sed 's/.*content="\([^"]*\)".*/\1/')
    echo "🔗 Slug: $slug"
    
    # Extract categories
    categories=$(grep -o '<meta name="category" content="[^"]*"' "$blog_file" | sed 's/.*content="\([^"]*\)".*/\1/' | tr '\n' ', ')
    echo "🏷️  Categories: $categories"
    
    # File size
    size=$(du -h "$blog_file" | cut -f1)
    echo "📊 Size: $size"
    
    # Check if slug file exists
    if [ -n "$slug" ] && [ -f "$SERVER_BLOG_DIR/$slug.html" ]; then
        echo "✅ Slug file exists: $slug.html"
    else
        echo "❌ Slug file missing"
    fi
}

# Main menu
case "${1:-}" in
    "list")
        list_blogs
        ;;
    "migrate")
        migrate_all_blogs
        ;;
    "info")
        if [ -z "${2:-}" ]; then
            echo "Usage: $0 info <blog_id>"
            echo "Example: $0 info 0030"
            exit 1
        fi
        show_blog_info "$2"
        ;;
    "add-slug")
        if [ -z "${2:-}" ]; then
            echo "Usage: $0 add-slug <blog_id>"
            echo "Example: $0 add-slug 0030"
            exit 1
        fi
        add_slug_to_blog "$2"
        ;;
    "backup")
        create_backup
        ;;
    *)
        echo "Bizoni Remote Blog Management"
        echo "============================"
        echo ""
        echo "Usage: $0 <command> [options]"
        echo ""
        echo "Commands:"
        echo "  list                    - List all blogs on server"
        echo "  migrate                 - Migrate all blogs to use slugs"
        echo "  info <blog_id>          - Show info about specific blog"
        echo "  add-slug <blog_id>      - Add slug to specific blog"
        echo "  backup                  - Create backup of all blogs"
        echo ""
        echo "Examples:"
        echo "  $0 list"
        echo "  $0 migrate"
        echo "  $0 info 0030"
        echo "  $0 add-slug 0030"
        echo ""
        echo "⚠️  Make sure to update SERVER_BLOG_DIR path in this script!"
        exit 1
        ;;
esac
