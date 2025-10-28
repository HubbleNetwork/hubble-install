#!/bin/bash
# Upload install.sh to S3 bucket
set -e

BUCKET_NAME="hubble-install"

echo "📦 Uploading install.sh to S3..."
aws s3 cp scripts/install.sh s3://${BUCKET_NAME}/install.sh \
  --content-type "text/x-shellscript" \
  --cache-control "max-age=300"

echo "✓ Upload complete!"
echo ""
echo "🔒 Applying bucket policy (make install.sh public)..."
aws s3api put-bucket-policy --bucket ${BUCKET_NAME} --policy file://bucket-policy.json

echo "✓ Bucket policy applied!"
echo ""
echo "🌍 Public URL:"
echo "   https://${BUCKET_NAME}.s3.amazonaws.com/install.sh"
echo ""
echo "Test with:"
echo "   curl -fsSL https://${BUCKET_NAME}.s3.amazonaws.com/install.sh | bash"

