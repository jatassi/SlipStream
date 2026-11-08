# truehd-atmos: made by go run ./fakes/internal/mediafix/capture with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
# FFmpeg has no Atmos encoder (E-AC-3 JOC or TrueHD with Atmos), so the capture of a plain truehd track
# has only streams[1].profile set by hand to "Dolby TrueHD + Dolby Atmos", the string the bundled ffprobe prints for it:
# strings ffprobe | grep Atmos  ->  Dolby Digital Plus + Dolby Atmos, Dolby TrueHD + Dolby Atmos
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24:duration=1 -f lavfi -i sine=frequency=440:sample_rate=48000:duration=1 -filter_complex '[1:a]asplit=1[s0];[s0]pan=5.1(side)|c0=c0|c1=c0|c2=c0|c3=c0|c4=c0|c5=c0[a0]' -map 0:v -map '[a0]' -c:v libx264 -preset ultrafast -threads 1 -c:a:0 truehd -metadata:s:a:0 language=eng -disposition:a:0 0 -strict -2 -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 truehd-atmos.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams truehd-atmos.mkv > truehd-atmos.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 truehd-atmos.mkv > truehd-atmos.frame.json
