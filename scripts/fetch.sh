#!/bin/sh

MC_VERSION=$(jq -r '.game_version' version.json)
JAR_FILE="server-$MC_VERSION.jar"
OUTPUT_DIR="internal/mcdata"

if ! command -v jq > /dev/null; then
  error "jq is not installed. Please install it to run this script."
fi

if [ ! -f "$JAR_FILE" ]; then
    echo "Fetching metadata for Minecraft $MC_VERSION..."

    # Get the version manifest
    MANIFEST_URL=$(curl -s "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json" | \
        jq -r --arg VER "$MC_VERSION" '.versions[] | select(.id == $VER) | .url')

    if [ -z "$MANIFEST_URL" ]; then
        echo "Error: Version $MC_VERSION not found in manifest."
        exit 1
    fi

    DOWNLOAD_URL=$(curl -s "$MANIFEST_URL" | jq -r '.downloads.server.url')

    echo "Downloading server.jar..."
    curl -o "$JAR_FILE" "$DOWNLOAD_URL"
else
    echo "server.jar already exists, skipping download."
fi

java -DbundlerMainClass=net.minecraft.data.Main -jar "$JAR_FILE" --all --output "$OUTPUT_DIR"
