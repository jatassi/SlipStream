import time, json, sys, uuid
from scen import *
from lib import trim
kind, mode = sys.argv[1], sys.argv[2]   # mode: transcode | direct
notl = "noreport" in sys.argv
log = Log(f"s6-{kind}")
s = MBS(kind); m = s.m
it, src = s.source("movies/Collateral (2004)/Collateral (2004) - 1080p.mkv")
iid, msid = it["Id"], src["Id"]
ps = uuid.uuid4().hex
if mode == "profile":
    prof = {"MaxStreamingBitrate": 400000, "MaxStaticBitrate": 400000,
            "DirectPlayProfiles": [{"Container": "mkv,mp4", "Type": "Video", "VideoCodec": "h264", "AudioCodec": "aac"}],
            "TranscodingProfiles": [{"Container": "ts", "Type": "Video", "VideoCodec": "h264", "AudioCodec": "aac", "Protocol": "hls", "Context": "Streaming", "MaxAudioChannels": "2"}],
            "CodecProfiles": [], "SubtitleProfiles": []}
    c, pi, ms = m.call("POST", f"/Items/{iid}/PlaybackInfo", {"DeviceProfile": prof, "MediaSourceId": msid, "MaxStreamingBitrate": 400000, "UserId": m.uid, "EnableDirectPlay": True, "EnableDirectStream": True, "EnableTranscoding": True, "AutoOpenLiveStream": True},
                       UserId=m.uid, MaxStreamingBitrate=400000, MediaSourceId=msid)
    srcs = pi.get("MediaSources", [])
    sel = [x for x in srcs if x["Id"] == msid][0]
    ps = pi.get("PlaySessionId") or ps
    log(mode, "PlaybackInfo", c, "sources", [(x["Id"], x.get("SupportsDirectPlay"), x.get("SupportsDirectStream")) for x in srcs], "TranscodeReasons field:", sel.get("TranscodeReasons"), "TranscodingUrl:", (sel.get("TranscodingUrl") or "")[:400])
    url = sel["TranscodingUrl"]
    c, body, ms = m.call("GET", url, raw=True)
    variant = [l for l in body.decode(errors="replace").splitlines() if l and not l.startswith("#")]
    c, pl, ms = m.call("GET", f"/Videos/{iid}/" + variant[0], raw=True)
    segs = [l for l in pl.decode(errors="replace").splitlines() if l and not l.startswith("#")]
    c, seg, ms = m.call("GET", f"/Videos/{iid}/" + segs[0], raw=True)
    log("segment0", c, len(seg))
    method = "Transcode"
elif mode == "transcode":
    q = dict(MediaSourceId=msid, PlaySessionId=ps, DeviceId=m.devid, VideoCodec="h264", AudioCodec="aac", VideoBitrate=300000, AudioBitrate=64000,
             MaxWidth=640, MaxHeight=360, MaxStreamingBitrate=400000, SegmentContainer="ts", TranscodingMaxAudioChannels=2, BreakOnNonKeyFrames="true")
    c, body, ms = m.call("GET", f"/Videos/{iid}/master.m3u8", raw=True, **q)
    log(mode, "master.m3u8", c, ms, "ms", body[:300])
    variant = [l for l in body.decode(errors="replace").splitlines() if l and not l.startswith("#")]
    if variant:
        c, pl, ms = m.call("GET", f"/Videos/{iid}/" + variant[0], raw=True)
        log("variant", c, ms, "ms", pl[:250])
        segs = [l for l in pl.decode(errors="replace").splitlines() if l and not l.startswith("#")]
        if segs:
            c, seg, ms = m.call("GET", f"/Videos/{iid}/" + segs[0], raw=True)
            log("segment0", c, len(seg), "bytes", ms, "ms")
    method = "Transcode"
else:
    c, body, ms = m.call("GET", f"/Videos/{iid}/stream", raw=True, static="true", MediaSourceId=msid, PlaySessionId=ps, DeviceId=m.devid)
    log(mode, "static stream", c, len(body), "bytes", ms, "ms")
    method = "DirectPlay"
ev = {"ItemId": iid, "MediaSourceId": msid, "PlaySessionId": ps, "PlayMethod": method, "CanSeek": True, "PositionTicks": 0, "IsPaused": False}
if not notl:
    log("POST /Sessions/Playing", m.call("POST", "/Sessions/Playing", ev)[:2])
    for i in range(3):
        time.sleep(1); ev["PositionTicks"] = (i + 1) * 10_000_000
        m.call("POST", "/Sessions/Playing/Progress", ev)
time.sleep(2)
c, sess, _ = m.call("GET", "/Sessions")
mine = [x for x in sess if x.get("DeviceId") == m.devid]
log(mode, "noreport" if notl else "", "GET /Sessions", c, "total", len(sess), "mine", len(mine))
for x in mine:
    log("  session keys:", sorted(x.keys()))
    log("  NowPlayingItem:", (x.get("NowPlayingItem") or {}).get("Id"), (x.get("NowPlayingItem") or {}).get("Name"), "PlayState:", x.get("PlayState"))
    log("  TranscodingInfo:", x.get("TranscodingInfo"))
time.sleep(2)
if not notl:
    m.call("POST", "/Sessions/Playing/Stopped", ev)
log("stop encodings", m.call("DELETE", "/Videos/ActiveEncodings", DeviceId=m.devid, PlaySessionId=ps)[:2])
time.sleep(2)
c, sess, _ = m.call("GET", "/Sessions")
log("after stop: mine NowPlaying", [(x.get("NowPlayingItem") or {}).get("Name") for x in sess if x.get("DeviceId") == m.devid], [x.get("TranscodingInfo") for x in sess if x.get("DeviceId") == m.devid])
