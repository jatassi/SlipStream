# eac3-atmos: made in a scratch folder with the tools SlipStream bundles (native/lock.json):
# ffmpeg version 8.1.3-Jellyfin, dovi_tool 2.3.4.
# FFmpeg has no Atmos encoder (E-AC-3 JOC or TrueHD with Atmos), so the capture of a plain eac3 track
# has only streams[1].profile set by hand to "Dolby Digital Plus + Dolby Atmos", the string the bundled ffprobe prints for it:
# strings ffprobe | grep Atmos  ->  Dolby Digital Plus + Dolby Atmos, Dolby TrueHD + Dolby Atmos
ffmpeg -v error -y -f lavfi -i testsrc2=size=320x180:rate=24 -f lavfi -i sine=frequency=440:sample_rate=48000:duration=1 -frames:v 1 -af 'pan=5.1(side)|c0=c0|c1=c0|c2=c0|c3=c0|c4=c0|c5=c0' -c:v libx264 -preset ultrafast -threads 1 -c:a eac3 -strict -2 -metadata:s:a:0 language=eng -fflags +bitexact -flags:v +bitexact -flags:a +bitexact -map_metadata -1 eac3-atmos.mkv
ffprobe -v error -protocol_whitelist file,pipe -print_format json -show_format -show_streams eac3-atmos.mkv > eac3-atmos.streams.json
ffprobe -v error -protocol_whitelist file,pipe -print_format json -select_streams v:0 -show_frames -read_intervals %+#1 eac3-atmos.mkv > eac3-atmos.frame.json
