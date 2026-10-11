import sys, time
from mb import MB, setup_libs, scan_running
from lib import poll
m = MB(sys.argv[1]); m.login()
tid = setup_libs(m)
t = time.time()
print("refresh", m.call("POST", "/Library/Refresh")[:2])
time.sleep(2)
poll(lambda: not scan_running(m, tid), 300, 2)
print("full scan took", round(time.time() - t, 1), "s")
