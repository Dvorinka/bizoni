# Remote Blog Management - Complete Guide

## 🎯 Goal
Remove local blog files and work exclusively with remote server blogs.

## 📋 Options Available

### Option 1: Quick Remove Local Blogs
```bash
# Run this in your bizoni directory
./remove-local-blogs.sh
```

### Option 2: Ubuntu Server Management Script
```bash
# Upload to your Ubuntu server and run
./ubuntu-remote-blogs.sh migrate
```

### Option 3: Backend Configuration (Recommended)
Update backend to work with remote blogs only.

## 🚀 Recommended Deployment Steps

### Step 1: Remove Local Blogs
```bash
cd /home/tdvorak/Desktop/HTML_Projekty/bizoni
./remove-local-blogs.sh
```

### Step 2: Update Backend Configuration
The backend is now configured to work with remote blogs at `/var/www/bizoni/blog`.

You can set the remote path with environment variable:
```bash
export REMOTE_BLOG_DIR="/var/www/bizoni/blog"
```

### Step 3: Deploy Backend to Server
1. Build the updated backend
2. Deploy to your server
3. Set REMOTE_BLOG_DIR environment variable

### Step 4: Run Migration on Server
```bash
# On your Ubuntu server
./ubuntu-remote-blogs.sh migrate
```

## 📁 File Structure After Changes

### Local (Development)
```
bizoni/
├── backend/main.go          # Updated for remote blogs
├── admin/new.html           # Updated with new fields
├── js/admin-auth.js         # Login persistence
├── tools/migrate_slugs.go   # Migration tool
├── remove-local-blogs.sh    # Local cleanup script
└── ubuntu-remote-blogs.sh   # Server management script
```

### Server (Production)
```
/var/www/bizoni/
├── blog/
│   ├── 0000.html            # Original numeric files
│   ├── 0001.html
│   ├── jdeme-do-finale.html # New slug files
│   └── 1-zapas-final-score.html
├── img/blog/
│   ├── 0000.png
│   └── 0001.png
└── backend                  # Updated backend
```

## 🔧 Ubuntu Server Script Usage

### List All Blogs
```bash
./ubuntu-remote-blogs.sh list
```

### Migrate All Blogs to Slugs
```bash
./ubuntu-remote-blogs.sh migrate
```

### Show Blog Info
```bash
./ubuntu-remote-blogs.sh info 0030
```

### Add Slug to Specific Blog
```bash
./ubuntu-remote-blogs.sh add-slug 0030
```

### Create Backup
```bash
./ubuntu-remote-blogs.sh backup
```

## 🌐 URL Structure After Migration

### Before
- `/blog/0030.html`
- `/blog/0001.html`

### After
- `/blog/jdeme-do-finale` (clean URL)
- `/blog/1-zapas-final-score`
- `/blog/0030.html` (still works for backward compatibility)

## ⚠️ Important Notes

1. **Backup First**: Always create backup before migration
2. **Test Locally**: Test backend with REMOTE_BLOG_DIR set to local copy
3. **Deploy Gradually**: Deploy backend first, then run migration
4. **Environment Variables**: Use REMOTE_BLOG_DIR for flexibility

## 🔄 Environment Variables

Set these on your server:

```bash
# Path to remote blog directory
export REMOTE_BLOG_DIR="/var/www/bizoni/blog"

# Port for backend (if needed)
export PORT="8080"

# Static files path
export STATIC_PATH="/var/www/bizoni"
```

## 🚨 Troubleshooting

### Backend Can't Find Blogs
```bash
# Check if directory exists
ls -la /var/www/bizoni/blog

# Set correct path
export REMOTE_BLOG_DIR="/correct/path/to/blogs"
```

### Migration Script Fails
```bash
# Make script executable
chmod +x ubuntu-remote-blogs.sh

# Update paths in script
nano ubuntu-remote-blogs.sh
```

### Permission Issues
```bash
# Fix permissions on server
sudo chown -R www-data:www-data /var/www/bizoni/blog
sudo chmod -R 755 /var/www/bizoni/blog
```

## 📞 Next Steps

1. **Choose your option** (1, 2, or 3)
2. **Remove local blogs** with the provided script
3. **Deploy updated backend** to server
4. **Run migration** on server
5. **Test new URLs** and admin interface

Your blog system will then work entirely with remote server blogs! 🎉
