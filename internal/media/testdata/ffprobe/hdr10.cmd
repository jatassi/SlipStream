# hdr10: made in a scratch folder with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24 -frames:v 1 -vf setparams=color_primaries=bt2020:color_trc=smpte2084:colorspace=bt2020nc,format=yuv420p10le -c:v libx265 -preset ultrafast -x265-params 'pools=1:frame-threads=1:log-level=error:hdr10=1:master-display=G(13250,34500)B(7500,3000)R(34000,16000)WP(15635,16450)L(10000000,1):max-cll=1000,400' -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 hdr10.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams hdr10.mkv > hdr10.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 hdr10.mkv > hdr10.frame.json
