#!/bin/sh
set -eu

if ! command -v jq > /dev/null; then
  echo "Error: jq is not installed. Please install it to run this script." >&2
  exit 1
fi

MC_VERSION=$(jq -r '.game_version' version.json)
JAR_FILE="server-$MC_VERSION.jar"
OUTPUT_DIR="internal/mcdata"

if [ ! -f "$JAR_FILE" ]; then
    echo "Fetching metadata for Minecraft $MC_VERSION..."

    # Get the version manifest
    MANIFEST_URL=$(curl -s "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json" | \
        jq -r --arg VER "$MC_VERSION" '.versions[] | select(.id == $VER) | .url')

    if [ -z "$MANIFEST_URL" ]; then
        echo "Error: Version $MC_VERSION not found in manifest."
        exit 1
    fi

    DOWNLOAD_URL=$(curl -s "$MANIFEST_URL" | jq -r '.downloads.server.url // empty')

    if [ -z "$DOWNLOAD_URL" ]; then
        echo "Error: Manifest for $MC_VERSION has no server download." >&2
        exit 1
    fi

    echo "Downloading server.jar..."
    # Without --fail curl happily writes an HTTP error page into the jar, which
    # then poisons every later run (and the CI cache) as an "existing" jar.
    if ! curl -fSL -o "$JAR_FILE" "$DOWNLOAD_URL"; then
        rm -f "$JAR_FILE"
        echo "Error: Failed to download $DOWNLOAD_URL" >&2
        exit 1
    fi
else
    echo "server.jar already exists, skipping download."
fi

java -DbundlerMainClass=net.minecraft.data.Main -jar "$JAR_FILE" --all --output "$OUTPUT_DIR"
