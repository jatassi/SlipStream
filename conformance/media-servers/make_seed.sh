#!/bin/bash
set -e
cd "$(dirname "$0")"
mkdir -p seed media/movies media/tv config/plex config/jellyfin config/emby transcode
mk() { ffmpeg -loglevel error -y -f lavfi -i "testsrc2=size=${2}:rate=24" -f lavfi -i "sine=frequency=440" -t 60 -c:v libx264 -preset ultrafast -b:v "$3" -pix_fmt yuv420p -c:a aac -b:a 96k "seed/$1"; }
mk 1080.mkv 1920x1080 2M
mk 2160.mkv 3840x2160 4M
mk 720.mkv 1280x720 2M
printf '1\n00:00:01,000 --> 00:00:04,000\nHello conformance\n\n' > seed/en.srt
printf '1\n00:00:01,000 --> 00:00:04,000\nBonjour conformance\n\n' > seed/fr.srt
printf '1\n00:00:01,000 --> 00:00:04,000\nReplaced cue\n\n' > seed/en-v2.srt
