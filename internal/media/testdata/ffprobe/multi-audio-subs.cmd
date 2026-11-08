# multi-audio-subs: made in a scratch folder with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
cat > spa.srt <<'EOF'
1
00:00:00,000 --> 00:00:01,000
Hola
EOF
cat > eng.srt <<'EOF'
1
00:00:00,000 --> 00:00:01,000
Hello
EOF
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24:duration=1 -f lavfi -i sine=frequency=440:sample_rate=48000:duration=1 -i spa.srt -i eng.srt -filter_complex '[1:a]asplit=3[a0][a1][a2];[a0]pan=5.1(side)|c0=c0|c1=c0|c2=c0|c3=c0|c4=c0|c5=c0[s51];[a2]pan=5.1(side)|c0=c0|c1=c0|c2=c0|c3=c0|c4=c0|c5=c0[t51]' -map 0:v -map '[s51]' -map '[a1]' -map '[t51]' -map 2 -map 3 -c:v libx264 -preset ultrafast -threads 1 -c:a:0 eac3 -c:a:1 aac -ac:a:1 2 -c:a:2 truehd -strict -2 -c:s srt -metadata:s:a:0 language=eng -metadata:s:a:0 title=Surround -metadata:s:a:1 language=jpn -metadata:s:a:2 language=fre -metadata:s:s:0 language=spa -metadata:s:s:1 language=eng -disposition:a:0 default -disposition:a:1 0 -disposition:a:2 0 -disposition:s:0 forced -disposition:s:1 0 -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 multi-audio-subs.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams multi-audio-subs.mkv > multi-audio-subs.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 multi-audio-subs.mkv > multi-audio-subs.frame.json
