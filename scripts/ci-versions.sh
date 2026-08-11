#!/bin/sh
# Resolves the game version from version.json and the JDK major version Mojang
# built that server jar with, then exports both to the workflow environment.
# Reading the JDK from the manifest keeps CI from drifting when Mojang bumps it.
set -eu

if [ -z "${GITHUB_ENV:-}" ]; then
  echo "Error: GITHUB_ENV is not set, this script only runs in GitHub Actions." >&2
  exit 1
fi

if ! command -v jq > /dev/null; then
  echo "Error: jq is not installed." >&2
  exit 1
fi

MC_VERSION=$(jq -r '.game_version' version.json)

MANIFEST_URL=$(curl -sS --fail "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json" | \
    jq -r --arg VER "$MC_VERSION" '.versions[] | select(.id == $VER) | .url')

if [ -z "$MANIFEST_URL" ]; then
    echo "::error::Version $MC_VERSION not found in manifest."
    exit 1
fi

JAVA_VERSION=$(curl -sS --fail "$MANIFEST_URL" | jq -r '.javaVersion.majorVersion // empty')

if [ -z "$JAVA_VERSION" ]; then
    echo "::error::Manifest for $MC_VERSION does not declare javaVersion.majorVersion."
    exit 1
fi

{
  echo "MC_VERSION=$MC_VERSION"
  echo "JAR_FILENAME=server-$MC_VERSION.jar"
  echo "JAVA_VERSION=$JAVA_VERSION"
} >> "$GITHUB_ENV"

echo "Game version: $MC_VERSION (Java $JAVA_VERSION)"
