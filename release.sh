#!/bin/bash

# release.sh - Automate the release process for go-pve-vdi
# Usage: ./release.sh [version]
# Example: ./release.sh 1.0.0

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if version is provided
if [ -z "$1" ]; then
    print_error "Version number required!"
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

VERSION=$1

# Remove 'v' prefix if provided
VERSION=${VERSION#v}

# Validate version format (basic semver check)
if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    print_error "Invalid version format. Please use semantic versioning (e.g., 1.0.0)"
    exit 1
fi

TAG="v${VERSION}"

print_info "Starting release process for version ${TAG}"
echo ""

# Check if git repo
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not a git repository!"
    exit 1
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    print_warn "You have uncommitted changes!"
    echo ""
    git status --short
    echo ""
    read -p "Do you want to commit these changes? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "Enter commit message: " commit_msg
        if [ -z "$commit_msg" ]; then
            commit_msg="Prepare for release ${TAG}"
        fi
        print_info "Committing changes..."
        git add .
        git commit -m "$commit_msg"
    else
        print_error "Please commit your changes before creating a release"
        exit 1
    fi
fi

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
    print_error "Tag ${TAG} already exists!"
    echo "Existing tags:"
    git tag | grep -E "^v[0-9]" | sort -V | tail -5
    exit 1
fi

# Show current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
print_info "Current branch: ${CURRENT_BRANCH}"

# Confirm release
echo ""
print_warn "This will:"
echo "  1. Create tag ${TAG}"
echo "  2. Push to remote (branch: ${CURRENT_BRANCH})"
echo "  3. Push tag ${TAG}"
echo "  4. Trigger GitHub Actions to build and release"
echo ""
read -p "Continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "Release cancelled"
    exit 0
fi

# Create tag
print_info "Creating tag ${TAG}..."
git tag -a "${TAG}" -m "Release version ${VERSION}"

# Push to remote
print_info "Pushing to remote repository..."
git push origin "${CURRENT_BRANCH}"

# Push tag
print_info "Pushing tag ${TAG}..."
git push origin "${TAG}"

echo ""
print_info "✅ Release ${TAG} initiated successfully!"
echo ""
echo "GitHub Actions will now:"
echo "  - Build binaries for all platforms"
echo "  - Generate checksums"
echo "  - Create a GitHub Release"
echo ""
echo "Check the progress at:"
echo "  https://github.com/javadmohebbi/go-pve-vdi/actions"
echo ""
echo "Once complete, the release will be available at:"
echo "  https://github.com/javadmohebbi/go-pve-vdi/releases/tag/${TAG}"
echo ""
