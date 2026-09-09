#!/usr/bin/env bash

# extracts .eml files from running container and processes them into anonymized test data

set -e

CONTAINER_ID=$(docker ps --filter "ancestor=nagomi-email-parser:latest" --format "{{.ID}}" | head -n1)

if [ -z "$CONTAINER_ID" ]; then
    echo "No running container found"
    exit 1
fi

TEMP_DIR="./temp_emails"
TESTDATA_DIR="./internal/email/rbc/testdata"

rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR" "$TESTDATA_DIR"

if docker cp "$CONTAINER_ID:/app/debug_emails/." "$TEMP_DIR/" 2>/dev/null; then
    EMAIL_COUNT=$(find "$TEMP_DIR" -name "*.eml" | wc -l)
    
    if [ "$EMAIL_COUNT" -eq 0 ]; then
        echo "No email files found"
        rm -rf "$TEMP_DIR"
        exit 0
    fi
    
    echo "Found $EMAIL_COUNT email file(s)"
    find "$TEMP_DIR" -name "*.eml" -exec mv {} "$TESTDATA_DIR/" \;
    
    if go run ./cmd/prepare-testdata/main.go; then
        echo "Processing completed"
        find "$TESTDATA_DIR" -name "*.decoded.eml" -exec basename {} \; | sort
    else
        echo "Failed to process emails"
        exit 1
    fi
else
    echo "No debug_emails directory found - set SAVE_EML=1 in container"
fi

rm -rf "$TEMP_DIR"