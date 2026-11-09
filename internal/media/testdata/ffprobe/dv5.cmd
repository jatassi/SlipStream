# dv5: made by go run ./fakes/internal/mediafix/capture with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
cat > dv5.dovi.json <<'EOF'
{"cm_version":"V40","profile":"5","length":24}
EOF
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24:duration=1 -vf format=yuv420p10le -c:v libx265 -preset ultrafast -x265-params pools=1:frame-threads=1:log-level=error:repeat-headers=1 -fflags +bitexact -flags:v +bitexact -f hevc dv5-base.hevc
dovi_tool generate -j dv5.dovi.json -o dv5.rpu
dovi_tool inject-rpu -i dv5-base.hevc --rpu-in dv5.rpu -o dv5-rpu.hevc
ffmpeg -v error -y -r 24 -i dv5-rpu.hevc -map 0:v -c:v libx265 -preset ultrafast -x265-params pools=1:frame-threads=1:log-level=error:vbv-bufsize=20000:vbv-maxrate=20000 -dolbyvision 1 -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 dv5.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams dv5.mkv > dv5.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 dv5.mkv > dv5.frame.json
