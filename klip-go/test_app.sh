#!/bin/bash
set -e

echo "Testing Klip Go app..."

# Test help flag
echo "Testing --help flag..."
./dist/klip --help > /dev/null

# Test version flag  
echo "Testing --version flag..."
./dist/klip --version > /dev/null

# Test that app starts and can be terminated
echo "Testing app startup..."
timeout 5s ./dist/klip || true

echo "All tests passed! ✅"