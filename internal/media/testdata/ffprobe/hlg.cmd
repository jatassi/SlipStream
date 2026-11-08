# hlg: made by go run ./fakes/internal/mediafix/capture with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24:duration=1 -map 0:v -vf setparams=color_primaries=bt2020:color_trc=arib-std-b67:colorspace=bt2020nc,format=yuv420p10le -c:v libx265 -preset ultrafast -x265-params pools=1:frame-threads=1:log-level=error -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 hlg.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams hlg.mkv > hlg.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 hlg.mkv > hlg.frame.json
