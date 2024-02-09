#!/bin/bash

# Check if at least three arguments are provided
if [ $# -lt 4 ]; then
    echo "Usage: $0 <search_dir> <output_file> <tag1> [<tag2> ...]"
    exit 1
fi

# Get the search directory and output file from arguments
search_dir="$1"
output_file="$2"
suitename="$3"
testname="$4"

shift 2  # Remove the first two arguments

# find the chunk of postman collection wrt to the suitename and append it to ouptut file
find "$search_dir" -type f -name "*.json" | while IFS= read -r file; do
    jq `.item[] | select(.name=="$suitename" and (.item[].name=="$testname"))` "$file" >> "$output_file"
done

echo "Search complete. Results are saved in $output_file"