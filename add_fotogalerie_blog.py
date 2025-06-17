import os
import re

def update_blog_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as file:
        content = file.read()
    
    # Pattern to find the Kontakt menu item and the following closing tags
    pattern = r'(<li id="menu-item-13613" class="menu-item menu-item-type-post_type menu-item-object-page">\s*<a href="\.\./kontakt\.html">\s*<span>Kontakt<\/span>\s*<\/a>\s*<\/li>\s*)(<\/ul>)'
    
    # Fotogalerie menu item to be inserted
    fotogalerie_item = '''                  <li id="menu-item-20758" class="menu-item menu-item-type-custom">
                    <a target="_blank" href="https://eu.zonerama.com/Fcbizoni/1419417">
                      <span>Fotogalerie</span>
                    </a>
                  </li>
                '''
    
    # Perform the replacement
    new_content = re.sub(pattern, r'\1' + fotogalerie_item + r'\2', content, flags=re.DOTALL)
    
    # Write the updated content back to the file
    if new_content != content:
        with open(file_path, 'w', encoding='utf-8') as file:
            file.write(new_content)
        return True
    return False

def main():
    blog_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'blog')
    
    # Process blog files from 0014.html to 0030.html
    for i in range(14, 31):
        filename = f"{i:04d}.html"
        file_path = os.path.join(blog_dir, filename)
        if os.path.exists(file_path):
            try:
                updated = update_blog_file(file_path)
                status = "Updated" if updated else "No changes needed"
                print(f"{filename}: {status}")
            except Exception as e:
                print(f"Error processing {filename}: {str(e)}")
        else:
            print(f"File not found: {filename}")

if __name__ == "__main__":
    main()
