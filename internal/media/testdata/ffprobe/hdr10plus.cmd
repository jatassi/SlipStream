# hdr10plus: made in a scratch folder with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
cat > hdr10plus.json <<'EOF'
{"JSONInfo":{"HDR10plusProfile":"A","Version":"1.0"},"SceneInfo":[{"BezierCurveData":{"Anchors":[256,512,768],"KneePointX":0,"KneePointY":0},"LuminanceParameters":{"AverageRGB":1000,"LuminanceDistributions":{"DistributionIndex":[1,5,10,25,50,75,90,95,99],"DistributionValues":[0,100,200,300,400,500,600,700,800]},"MaxScl":[1000,1000,1000]},"NumberOfWindows":1,"TargetedSystemDisplayMaximumLuminance":400,"SceneFrameIndex":0,"SceneId":0,"SequenceFrameIndex":0}],"SceneInfoSummary":{"SceneFirstFrameIndex":[0],"SceneFrameNumbers":[1]},"ToolInfo":{"Tool":"slipstream","Version":"1"}}
EOF
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24 -frames:v 1 -vf setparams=color_primaries=bt2020:color_trc=smpte2084:colorspace=bt2020nc,format=yuv420p10le -c:v libx265 -preset ultrafast -x265-params 'pools=1:frame-threads=1:log-level=error:hdr10=1:dhdr10-info=hdr10plus.json:master-display=G(13250,34500)B(7500,3000)R(34000,16000)WP(15635,16450)L(10000000,1):max-cll=1000,400' -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 hdr10plus.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams hdr10plus.mkv > hdr10plus.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 hdr10plus.mkv > hdr10plus.frame.json
