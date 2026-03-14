#!/bin/bash

# 🚀 Bizoni Remote Blog Setup - One Command Script
# Run this to setup remote blog management

set -e

echo "🚀 Bizoni Remote Blog Setup"
echo "=========================="

# Configuration - UPDATE THESE
SERVER_USER="your_username"          # Your SSH username
SERVER_HOST="your_server.com"       # Your server domain/IP
SERVER_BLOG_DIR="/var/www/bizoni/blog"  # Blog directory on server
LOCAL_PROJECT_DIR="/home/tdvorak/Desktop/HTML_Projekty/bizoni"  # Your local project

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if local blogs exist
check_local_blogs() {
    print_status "Checking local blog files..."
    
    if [ -d "$LOCAL_PROJECT_DIR/blog" ]; then
        blog_count=$(find "$LOCAL_PROJECT_DIR/blog" -name "*.html" -type f 2>/dev/null | wc -l)
        if [ "$blog_count" -gt 0 ]; then
            print_warning "Found $blog_count local blog files"
            read -p "Remove local blog files? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                rm -rf "$LOCAL_PROJECT_DIR/blog"
                print_success "Local blog files removed"
            else
                print_warning "Skipping local blog removal"
            fi
        else
            print_success "No local blog files found"
        fi
    else
        print_success "No local blog directory found"
    fi
}

# Create Ubuntu management script
create_server_script() {
    print_status "Creating server management script..."
    
    cat > /tmp/ubuntu-blog-manager.sh << 'EOF'
#!/bin/bash

# Ubuntu Server Blog Management Script
set -e

BLOG_DIR="/var/www/bizoni/blog"
BACKUP_DIR="/var/backups/bizoni-blogs"

# Create backup
create_backup() {
    echo "📦 Creating backup..."
    mkdir -p "$BACKUP_DIR"
    timestamp=$(date +%Y%m%d_%H%M%S)
    tar -czf "$BACKUP_DIR/blogs_backup_$timestamp.tar.gz" -C "$BLOG_DIR" . 2>/dev/null || echo "No files to backup"
    echo "✅ Backup created"
}

# Generate slug from title
generate_slug() {
    local title="$1"
    echo "$title" | tr '[:upper:]' '[:lower:]' | \
        sed 's/á/a/g; s/ä/a/g; s/č/c/g; s/ď/d/g; s/é/e/g; s/ě/e/g; s/í/i/g; s/ľ/l/g; s/ň/n/g; s/ó/o/g; s/ö/o/g; s/ô/o/g; s/ř/r/g; s/š/s/g; s/ť/t/g; s/ú/u/g; s/ů/u/g; s/ý/y/g; s/ž/z/g' | \
        sed 's/Á/a/g; s/Ä/a/g; s/Č/c/g; s/Ď/d/g; s/É/e/g; s/Ě/e/g; s/Í/i/g; s/Ľ/l/g; s/Ň/n/g; s/Ó/o/g; s/Ö/o/g; s/Ô/o/g; s/Ř/r/g; s/Š/s/g; s/Ť/t/g; s/Ú/u/g; s/Ů/u/g; s/Ý/y/g; s/Ž/z/g' | \
        sed 's/[^a-z0-9\s-]//g' | \
        sed 's/[\s-]\+/ -/g' | \
        sed 's/^-\|-$//g'
}

# Add slug to blog
add_slug_to_blog() {
    local blog_id="$1"
    local blog_file="$BLOG_DIR/$blog_id.html"
    
    if [ ! -f "$blog_file" ]; then
        echo "❌ Blog not found: $blog_id.html"
        return 1
    fi
    
    # Check if slug already exists
    if grep -q '<meta name="slug"' "$blog_file"; then
        echo "ℹ️  $blog_id already has slug"
        return 0
    fi
    
    # Extract title
    title=$(grep -o '<h1[^>]*class="[^"]*lte-header[^"]*"[^>]*>.*</h1>' "$blog_file" 2>/dev/null | sed 's/<[^>]*>//g' | xargs || echo "")
    
    if [ -z "$title" ]; then
        echo "❌ Could not extract title from $blog_id"
        return 1
    fi
    
    # Generate slug
    slug=$(generate_slug "$title")
    
    # Check uniqueness
    counter=2
    original_slug="$slug"
    while [ -f "$BLOG_DIR/$slug.html" ]; do
        slug="$original_slug-$counter"
        ((counter++))
    done
    
    echo "📝 $blog_id: $title → $slug"
    
    # Add slug meta tag
    sed -i "s|</head>|<meta name=\"slug\" content=\"$slug\">\n</head>|" "$blog_file"
    
    # Create slug file
    cp "$blog_file" "$BLOG_DIR/$slug.html"
    
    echo "✅ Added slug: $slug.html"
}

# Migrate all blogs
migrate_all() {
    echo "🔄 Migrating all blogs..."
    
    if [ ! -d "$BLOG_DIR" ]; then
        echo "❌ Blog directory not found: $BLOG_DIR"
        exit 1
    fi
    
    create_backup
    
    count=0
    for blog_file in "$BLOG_DIR"/*.html; do
        if [ -f "$blog_file" ]; then
            filename=$(basename "$blog_file")
            if [[ "$filename" =~ ^([0-9]{4})\.html$ ]]; then
                blog_id="${BASH_REMATCH[1]}"
                add_slug_to_blog "$blog_id"
                ((count++))
            fi
        fi
    done
    
    echo "✅ Migration completed! Processed $count blogs"
}

# List blogs
list_blogs() {
    echo "📋 Blogs on server:"
    if [ -d "$BLOG_DIR" ]; then
        for file in "$BLOG_DIR"/*.html; do
            if [ -f "$file" ]; then
                filename=$(basename "$file")
                if [[ "$filename" =~ ^[0-9]{4}\.html$ ]]; then
                    echo "  📄 $filename (numeric)"
                elif [[ "$filename" =~ ^[a-z0-9-]+\.html$ ]]; then
                    echo "  🔗 $filename (slug)"
                fi
            fi
        done
    else
        echo "  ❌ Blog directory not found"
    fi
}

case "${1:-}" in
    "migrate")
        migrate_all
        ;;
    "list")
        list_blogs
        ;;
    "backup")
        create_backup
        ;;
    *)
        echo "Ubuntu Blog Manager"
        echo "=================="
        echo "Usage: $0 <command>"
        echo ""
        echo "Commands:"
        echo "  migrate  - Add slugs to all blogs"
        echo "  list     - List all blogs"
        echo "  backup   - Create backup"
        echo ""
        echo "Example: $0 migrate"
        exit 1
        ;;
esac
EOF

    chmod +x /tmp/ubuntu-blog-manager.sh
    print_success "Server script created"
}

# Deploy to server
deploy_to_server() {
    print_status "Deploying to server..."
    
    # Upload server script
    scp /tmp/ubuntu-blog-manager.sh "$SERVER_USER@$SERVER_HOST:/tmp/"
    
    # Move script to server location and make executable
    ssh "$SERVER_USER@$SERVER_HOST" "sudo mv /tmp/ubuntu-blog-manager.sh /usr/local/bin/blog-manager && sudo chmod +x /usr/local/bin/blog-manager"
    
    print_success "Server script deployed to /usr/local/bin/blog-manager"
}

# Test server connection
test_connection() {
    print_status "Testing server connection..."
    
    if ssh "$SERVER_USER@$SERVER_HOST" "echo 'Connection successful'" 2>/dev/null; then
        print_success "Server connection OK"
    else
        print_error "Cannot connect to server. Please check:"
        echo "  - Username: $SERVER_USER"
        echo "  - Host: $SERVER_HOST"
        echo "  - SSH key or password setup"
        exit 1
    fi
}

# Run migration on server
run_migration() {
    print_status "Running migration on server..."
    
    ssh "$SERVER_USER@$SERVER_HOST" "sudo blog-manager migrate"
    
    print_success "Migration completed on server"
}

# Show results
show_results() {
    print_status "Showing results..."
    
    echo ""
    ssh "$SERVER_USER@$SERVER_HOST" "blog-manager list"
    echo ""
    print_success "Setup completed!"
    echo ""
    echo "🌐 Your blogs now have clean URLs:"
    echo "   Old: /blog/0030.html"
    echo "   New: /blog/jdeme-do-finale"
    echo ""
    echo "🔧 Server commands you can use:"
    echo "   ssh $SERVER_USER@$SERVER_HOST 'blog-manager list'"
    echo "   ssh $SERVER_USER@$SERVER_HOST 'blog-manager backup'"
    echo ""
}

# Main execution
main() {
    echo ""
    print_warning "Please update the configuration in this script:"
    echo "   SERVER_USER=\"$SERVER_USER\""
    echo "   SERVER_HOST=\"$SERVER_HOST\""
    echo "   SERVER_BLOG_DIR=\"$SERVER_BLOG_DIR\""
    echo ""
    read -p "Continue with current settings? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Please edit the script and run again"
        exit 1
    fi
    
    check_local_blogs
    test_connection
    create_server_script
    deploy_to_server
    run_migration
    show_results
}

# Run main function
main "$@"
