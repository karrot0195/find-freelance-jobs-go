#!/bin/bash

rm -rf .promt
mkdir -p .promt

# Copy all .go files (or any files) flattened with path encoded in name
find ./cmd ./internal -type f -name "*.go" | while read -r file; do
    # Replace / with _ to create a flat filename
    flatname=$(echo "$file" | sed 's|^\./||; s|/|_|g')
    cp "$file" ".promt/$flatname"
done

tree > .promt/tree.txt