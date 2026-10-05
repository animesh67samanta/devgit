#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo " DevGit Release Pre-Flight Check"
echo "=========================================="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

echo "--> 1. Checking code formatting (gofmt)..."
DIFFS=$(gofmt -l .)
if [ -n "${DIFFS}" ]; then
  echo "❌ Unformatted files detected:"
  echo "${DIFFS}"
  exit 1
fi
echo "✅ Code formatting is clean."

echo "--> 2. Running go vet..."
go vet ./...
echo "✅ go vet passed."

echo "--> 3. Running unit tests..."
go test -count=1 ./...
echo "✅ Unit tests passed."

echo "--> 4. Running race detector..."
go test -race -count=1 ./...
echo "✅ Race detector passed."

echo "--> 5. Building native binary..."
go build -v .
echo "✅ Native build passed."

echo "--> 6. Verifying cross-platform builds (CGO_ENABLED=0)..."
TMP_BUILD_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_BUILD_DIR}"' EXIT

PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
  "windows/arm64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS="${PLATFORM%/*}"
  GOARCH="${PLATFORM#*/}"
  OUT="${TMP_BUILD_DIR}/devgit_${GOOS}_${GOARCH}"
  if [ "${GOOS}" = "windows" ]; then
    OUT="${OUT}.exe"
  fi
  printf "    Compiling %-15s ... " "${PLATFORM}"
  CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -o "${OUT}" .
  echo "OK"
done
echo "✅ All 6 target platforms cross-compiled successfully."

echo "=========================================="
echo " All release checks passed successfully!"
echo "=========================================="
