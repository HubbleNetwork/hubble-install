#!/bin/bash
# Upload install.sh and binaries to S3 bucket
# Note: Bucket policy should already be configured to make all files public
set -e

BUCKET_NAME="hubble-install"
BIN_DIR="bin"

# Check if binaries exist
if [ ! -d "$BIN_DIR" ]; then
    echo "❌ Error: bin/ directory not found. Run ./scripts/build.sh first."
    exit 1
fi

echo "🚀 Uploading to S3 bucket: ${BUCKET_NAME}"
echo ""

# Upload install.sh
echo "📦 Uploading install.sh..."
aws s3 cp scripts/install.sh s3://${BUCKET_NAME}/install.sh \
  --content-type "text/x-shellscript" \
  --cache-control "max-age=300"
echo "✓ install.sh uploaded"
echo ""

# Upload install.ps1 (Windows)
echo "📦 Uploading install.ps1..."
aws s3 cp scripts/install.ps1 s3://${BUCKET_NAME}/install.ps1 \
  --content-type "text/plain" \
  --cache-control "max-age=300"
echo "✓ install.ps1 uploaded"
echo ""

# Upload binaries
echo "📦 Uploading binaries..."
for binary in ${BIN_DIR}/*; do
    filename=$(basename "$binary")
    echo "  → $filename"
    aws s3 cp "$binary" "s3://${BUCKET_NAME}/${filename}" \
        --content-type "application/octet-stream" \
        --cache-control "max-age=300"
done
echo "✓ All binaries uploaded"
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Upload complete!"
echo ""
echo "🌍 Public URLs:"
echo ""
echo "  Installer scripts:"
echo "    macOS/Linux: https://${BUCKET_NAME}.s3.amazonaws.com/install.sh"
echo "    Windows:     https://${BUCKET_NAME}.s3.amazonaws.com/install.ps1"
echo ""
echo "  Binaries:"
for binary in ${BIN_DIR}/*; do
    filename=$(basename "$binary")
    echo "    https://${BUCKET_NAME}.s3.amazonaws.com/${filename}"
done
echo ""
echo "Test installation:"
echo "  macOS/Linux:"
echo "    curl -fsSL https://${BUCKET_NAME}.s3.amazonaws.com/install.sh | bash"
echo "  Windows (PowerShell as Admin):"
echo "    iex \"& { \$(irm https://${BUCKET_NAME}.s3.amazonaws.com/install.ps1) }\""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

