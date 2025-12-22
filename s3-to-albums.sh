#!/bin/bash

# Configuration
BASE_URL="https://pub-a45a62d118874323b3ace83c57a9d6a7.r2.dev"
# You can pass a filename as $1 or pipe data to the script
INPUT="${1:-/dev/stdin}"

jq --arg base "$BASE_URL" '
  [
    # Iterate through the S3 contents
    .Contents[] | 
    {
      # Path parsing
      full_key: .Key,
      album_name: (.Key | split("/")[0]),
      # Remove extension for the track title
      clean_title: (.Key | split("/")[-1] | sub("\\.[^.]+$"; "")),
      # Encode each segment of the path separately to preserve slashes
      encoded_path: (.Key | split("/") | map(@uri) | join("/"))
    }
  ] | 
  # Group items by the album folder name
  group_by(.album_name) | 
  map({
    title: .[0].album_name,
    totalTracks: length,
    coverUrl: "",
    tracks: map({
      title: .clean_title,
      audioUrl: ($base + "/" + .encoded_path)
    })
  })
' "$INPUT"
