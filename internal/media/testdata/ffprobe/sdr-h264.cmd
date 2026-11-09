# sdr-h264: made by go run ./fakes/internal/mediafix/capture with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24:duration=1 -f lavfi -i sine=frequency=440:sample_rate=48000:duration=1 -filter_complex '[1:a]asplit=1[s0];[s0]pan=stereo|c0=c0|c1=c0[a0]' -map 0:v -map '[a0]' -c:v libx264 -preset ultrafast -threads 1 -c:a:0 aac -metadata:s:a:0 language=eng -disposition:a:0 0 -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 sdr-h264.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams sdr-h264.mkv > sdr-h264.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 sdr-h264.mkv > sdr-h264.frame.json
