# multi-audio-subs: made by go run ./fakes/internal/mediafix/capture with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
cat > multi-audio-subs.1.srt <<'EOF'
1
00:00:00,000 --> 00:00:01,000
Subtitle 1
EOF
cat > multi-audio-subs.2.srt <<'EOF'
1
00:00:00,000 --> 00:00:01,000
Subtitle 2
EOF
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24:duration=1 -f lavfi -i sine=frequency=440:sample_rate=48000:duration=1 -filter_complex '[1:a]asplit=3[s0][s1][s2];[s0]pan=5.1(side)|c0=c0|c1=c0|c2=c0|c3=c0|c4=c0|c5=c0[a0];[s1]pan=stereo|c0=c0|c1=c0[a1];[s2]pan=5.1(side)|c0=c0|c1=c0|c2=c0|c3=c0|c4=c0|c5=c0[a2]' -i multi-audio-subs.1.srt -i multi-audio-subs.2.srt -map 0:v -map '[a0]' -map '[a1]' -map '[a2]' -map 2 -map 3 -c:v libx264 -preset ultrafast -threads 1 -c:a:0 eac3 -metadata:s:a:0 language=eng -metadata:s:a:0 title=Surround -disposition:a:0 default -c:a:1 aac -metadata:s:a:1 language=jpn -disposition:a:1 0 -c:a:2 truehd -metadata:s:a:2 language=fre -disposition:a:2 0 -strict -2 -metadata:s:s:0 language=spa -disposition:s:0 forced -metadata:s:s:1 language=eng -disposition:s:1 0 -c:s srt -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 multi-audio-subs.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams multi-audio-subs.mkv > multi-audio-subs.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 multi-audio-subs.mkv > multi-audio-subs.frame.json
