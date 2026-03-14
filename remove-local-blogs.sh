#!/bin/bash

# Remove local blog files and keep only remote blogs
# Run this in your bizoni directory

echo "🗑️  Removing local blog files..."

# Remove all blog files
rm -rf blog/

echo "✅ Local blog files removed!"
echo ""
echo "📝 Next steps:"
echo "1. Deploy updated backend to server"
echo "2. Backend will now work with remote blogs only"
echo "3. Admin interface will connect to server API"
echo ""
echo "🌐 Your live blogs will remain untouched on the server"
