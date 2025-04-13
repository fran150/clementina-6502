#!/bin/bash
# Improved script to check for undocumented exported symbols in Go code

echo "Checking for undocumented exported symbols..."

# Find all Go files (excluding tests and vendor)
go_files=$(find . -name "*.go" | grep -v "_test.go" | grep -v "/vendor/" | grep -v "/build/")

# Check for exported symbols without documentation
undocumented=0
for file in $go_files; do
  # Get all exported symbols (types, functions, etc.)
  while IFS= read -r line; do
    # Skip empty lines
    [ -z "$line" ] && continue
    
    # Get line number and content
    line_num=$(echo "$line" | cut -d':' -f1)
    content=$(echo "$line" | cut -d':' -f2-)
    
    # Extract symbol name
    symbol=$(echo "$content" | sed -E 's/^[[:space:]]*(func|type|const|var)[[:space:]]+([A-Z][a-zA-Z0-9_]*).*/\2/')
    
    # Check for documentation above the symbol
    has_doc=0
    if [ "$line_num" -gt 1 ]; then
      # Check up to 5 lines above for comments
      for i in {1..5}; do
        prev_line=$((line_num - i))
        [ "$prev_line" -lt 1 ] && break
        
        prev_content=$(sed -n "${prev_line}p" "$file")
        if echo "$prev_content" | grep -q -E "^[[:space:]]*(//|\*|/\*)"; then
          has_doc=1
          break
        elif echo "$prev_content" | grep -q -E "^[[:space:]]*$"; then
          # Skip empty lines
          continue
        else
          # Found non-comment, non-empty line
          break
        fi
      done
    fi
    
    if [ $has_doc -eq 0 ]; then
      echo -e "\033[0;31mMissing documentation in $file:\033[0m"
      echo "$content"
      undocumented=1
    fi
  done < <(grep -n -E "^[[:space:]]*(func|type|const|var)[[:space:]]+[A-Z][a-zA-Z0-9_]*" "$file")
done

if [ $undocumented -eq 1 ]; then
  echo -e "\033[0;31m❌ Some exported symbols are missing documentation\033[0m"
  exit 1
else
  echo -e "\033[0;32m✅ All exported symbols are documented\033[0m"
fi
