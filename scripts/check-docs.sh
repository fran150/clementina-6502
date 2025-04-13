#!/bin/bash
# Script to check for undocumented exported symbols in Go code

echo "Checking for undocumented exported symbols..."

# Find all Go files (excluding tests and vendor)
go_files=$(find . -name "*.go" | grep -v "_test.go" | grep -v "/vendor/" | grep -v "/build/")

# Check for exported symbols without documentation
undocumented=0
for file in $go_files; do
  # Get exported symbols without documentation
  missing=$(grep -E "^[[:space:]]*(func|type|const|var)[[:space:]]+[A-Z][a-zA-Z0-9_]*" "$file" | 
           grep -v -B1 "\/\/" | 
           grep -v "^--$" | 
           grep -v "_test.go")
  
  if [ ! -z "$missing" ]; then
    echo -e "\033[0;31mMissing documentation in $file:\033[0m"
    echo "$missing"
    echo ""
    undocumented=1
  fi
done

if [ $undocumented -eq 1 ]; then
  echo -e "\033[0;31m❌ Some exported symbols are missing documentation\033[0m"
  exit 1
else
  echo -e "\033[0;32m✅ All exported symbols are documented\033[0m"
fi
