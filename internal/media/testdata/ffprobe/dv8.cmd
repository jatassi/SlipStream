# dv8: made in a scratch folder with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
# Dolby Vision needs a re-encode with -dolbyvision 1; a remux of the RPU-injected stream carries no configuration record.
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24 -frames:v 1 -vf setparams=color_primaries=bt2020:color_trc=smpte2084:colorspace=bt2020nc,format=yuv420p10le -c:v libx265 -preset ultrafast -x265-params 'pools=1:frame-threads=1:log-level=error:hdr10=1:repeat-headers=1:master-display=G(13250,34500)B(7500,3000)R(34000,16000)WP(15635,16450)L(10000000,1):max-cll=1000,400' -fflags +bitexact -flags:v +bitexact -f hevc dv8-base.hevc
cat > dv8.json <<'EOF'
{"cm_version":"V40","profile":"8.1","length":1,"level6":{"max_display_mastering_luminance":1000,"min_display_mastering_luminance":1,"max_content_light_level":1000,"max_frame_average_light_level":400}}
EOF
dovi_tool generate -j dv8.json -o dv8.rpu
dovi_tool inject-rpu -i dv8-base.hevc --rpu-in dv8.rpu -o dv8-rpu.hevc
ffmpeg -v error -y -r 24 -i dv8-rpu.hevc -vf setparams=color_primaries=bt2020:color_trc=smpte2084:colorspace=bt2020nc -c:v libx265 -preset ultrafast -dolbyvision 1 -x265-params 'pools=1:frame-threads=1:log-level=error:hdr10=1:vbv-bufsize=20000:vbv-maxrate=20000:master-display=G(13250,34500)B(7500,3000)R(34000,16000)WP(15635,16450)L(10000000,1):max-cll=1000,400' -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 dv8.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams dv8.mkv > dv8.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 dv8.mkv > dv8.frame.json
