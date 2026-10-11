#!/usr/bin/env python3
import argparse, json, os, signal, sys, time

ap = argparse.ArgumentParser()
ap.add_argument("--pid", type=int, required=True)
ap.add_argument("--interval", type=float, default=5)
ap.add_argument("--out", required=True)
a = ap.parse_args()

stop = False
def term(*_):
    global stop
    stop = True
signal.signal(signal.SIGTERM, term)
signal.signal(signal.SIGINT, term)

base = f"/proc/{a.pid}"

def sample():
    t = time.time()
    rss = pss = anon = file = shmem = 0
    with open(f"{base}/status") as f:
        for l in f:
            if l.startswith("VmRSS:"):
                rss = int(l.split()[1])
            elif l.startswith("RssAnon:"):
                anon = int(l.split()[1])
            elif l.startswith("RssFile:"):
                file = int(l.split()[1])
            elif l.startswith("RssShmem:"):
                shmem = int(l.split()[1])
            elif l.startswith("Threads:"):
                nthreads = int(l.split()[1])
    try:
        with open(f"{base}/smaps_rollup") as f:
            for l in f:
                if l.startswith("Pss:"):
                    pss = int(l.split()[1])
                    break
    except OSError:
        pass
    with open(f"{base}/stat") as f:
        s = f.read()
    fields = s[s.rindex(")") + 2:].split()
    ticks = int(fields[11]) + int(fields[12])
    try:
        fds = len(os.listdir(f"{base}/fd"))
    except OSError:
        fds = -1
    return {"t": t, "rss_kb": rss, "rss_anon_kb": anon, "rss_file_kb": file, "rss_shmem_kb": shmem, "pss_kb": pss, "cpu_ticks": ticks, "num_threads": nthreads, "fds": fds}

with open(a.out, "w") as out:
    while not stop:
        try:
            rec = sample()
        except (FileNotFoundError, ProcessLookupError):
            break
        if os.path.exists(f"{base}/stat"):
            with open(f"{base}/stat") as f:
                if f.read().split(") ")[-1].split()[0] == "Z":
                    break
        out.write(json.dumps(rec) + "\n")
        out.flush()
        end = time.time() + a.interval
        while not stop and time.time() < end:
            time.sleep(min(0.2, max(0, end - time.time())))
