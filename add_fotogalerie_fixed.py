import os
import re

def update_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as file:
        content = file.read()
    
    # Pattern to find the Kontakt menu item and the following closing tags
    pattern = r'(<li id="menu-item-\d+" class="menu-item menu-item-type-post_type menu-item-object-page">\s*<a href="\.\./kontakt\.html">\s*<span>Kontakt<\/span>\s*<\/a>\s*<\/li>\s*)(<\/ul>)'
    
    # Replacement string that includes the new Fotogalerie item
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
    zapasy_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'zapasy')
    
    # Process all HTML files in the zapasy directory
    for filename in os.listdir(zapasy_dir):
        if filename.endswith('.html'):
            file_path = os.path.join(zapasy_dir, filename)
            try:
                updated = update_file(file_path)
                status = "Updated" if updated else "No changes needed"
                print(f"{filename}: {status}")
            except Exception as e:
                print(f"Error processing {filename}: {str(e)}")

if __name__ == "__main__":
    main()
