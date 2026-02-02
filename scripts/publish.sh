#!/bin/bash

error() {
  echo "Error: $1" >&2
  exit 1
}

if [ $# -ne 1 ]; then
  error "Please specify a version (e.g.: \"1.0.0\")"
fi

VERSION=$1

if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  error "Version format must be \"x.y.z\" (e.g.: \"1.0.0\")"
fi

[ -f './go.mod' ] || error "go.mod file not found"

if ! command -v git &> /dev/null; then
  error "git is not installed."
fi

if ! command -v go &> /dev/null; then
  error "go is not installed."
fi

if ! git diff-index --quiet HEAD --; then
  echo "Warning: You have uncommitted changes:"
  git status -s
  read -r -p "Do you want to continue anyway? (y/n): " answer
  if [[ "$answer" != "y" && "$answer" != "Y" ]]; then
    exit 1
  fi
fi

CURRENT_BRANCH=$(git branch --show-current)
LOCAL_COMMIT=$(git rev-parse HEAD)
REMOTE_COMMIT=$(git rev-parse "origin/$CURRENT_BRANCH" 2>/dev/null || echo "")

if [[ -n "$REMOTE_COMMIT" && "$LOCAL_COMMIT" != "$REMOTE_COMMIT" ]]; then
  echo "Warning: You have unpushed commits:"
  git log --oneline "origin/$CURRENT_BRANCH"..HEAD
  read -r -p "Do you want to continue anyway? (y/n): " answer
  if [[ "$answer" != "y" && "$answer" != "Y" ]]; then
    exit 1
  fi
fi

if git rev-parse -q --verify "refs/tags/v$VERSION" >/dev/null; then
  error "Local tag v$VERSION already exists."
fi

if git ls-remote --exit-code --tags origin "refs/tags/v$VERSION" >/dev/null 2>&1; then
  error "Remote tag v$VERSION already exists."
fi

echo "Creating and publishing tag v$VERSION..."
git tag -a "v$VERSION" -m "Version $VERSION" || error "Unable to create tag"
git push origin "v$VERSION" || error "Unable to push tag"

echo "✅ Tag v$VERSION successfully created and pushed!"
