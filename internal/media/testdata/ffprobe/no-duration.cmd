# no-duration: made by go run ./fakes/internal/mediafix/capture with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
# No container this ffmpeg writes (.ts or .m2ts included, truncated or not) leaves format.duration out, so the case is an H.264 elementary stream.
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24:duration=1 -map 0:v -c:v libx264 -preset ultrafast -threads 1 -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 -f h264 no-duration.h264
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams no-duration.h264 > no-duration.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 no-duration.h264 > no-duration.frame.json
