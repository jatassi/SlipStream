import sys
from mb import MB
v = int(sys.argv[1])
for k, key in [("jellyfin", "LibraryMonitorDelay"), ("emby", "LibraryMonitorDelaySeconds")]:
    m = MB(k); m.login(); c, cfg, _ = m.call("GET", "/System/Configuration")
    cfg[key] = v
    print(k, m.call("POST", "/System/Configuration", cfg)[:2], m.call("GET", "/System/Configuration")[1][key])
