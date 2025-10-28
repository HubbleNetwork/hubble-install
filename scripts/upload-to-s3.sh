#!/bin/bash
# Upload install.sh and binaries to S3 bucket
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

# Apply bucket policy
echo "🔒 Applying bucket policy (make all files public)..."
aws s3api put-bucket-policy --bucket ${BUCKET_NAME} --policy file://bucket-policy.json
echo "✓ Bucket policy applied"
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Upload complete!"
echo ""
echo "🌍 Public URLs:"
echo ""
echo "  Installer script:"
echo "    https://${BUCKET_NAME}.s3.amazonaws.com/install.sh"
echo ""
echo "  Binaries:"
for binary in ${BIN_DIR}/*; do
    filename=$(basename "$binary")
    echo "    https://${BUCKET_NAME}.s3.amazonaws.com/${filename}"
done
echo ""
echo "Test installation:"
echo "   curl -fsSL https://${BUCKET_NAME}.s3.amazonaws.com/install.sh | bash"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

