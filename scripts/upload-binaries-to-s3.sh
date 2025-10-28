#!/bin/bash
# Upload all platform binaries to S3 bucket
set -e

BUCKET_NAME="hubble-install"
BIN_DIR="bin"

if [ ! -d "$BIN_DIR" ]; then
    echo "❌ Error: bin/ directory not found. Run ./scripts/build.sh first."
    exit 1
fi

echo "📦 Uploading binaries to S3..."
echo ""

# Upload each binary
for binary in ${BIN_DIR}/*; do
    filename=$(basename "$binary")
    echo "  Uploading: $filename"
    aws s3 cp "$binary" "s3://${BUCKET_NAME}/${filename}" \
        --content-type "application/octet-stream" \
        --cache-control "max-age=300"
done

echo ""
echo "✓ All binaries uploaded!"
echo ""
echo "🌍 Public URLs (after setting bucket policy):"
for binary in ${BIN_DIR}/*; do
    filename=$(basename "$binary")
    echo "   https://${BUCKET_NAME}.s3.amazonaws.com/${filename}"
done
echo ""
echo "Note: Binaries are private by default. Only install.sh is public."

