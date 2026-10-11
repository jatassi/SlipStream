#!/usr/bin/env python3
import json, os, signal, subprocess, sys, time

R = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BINS = {
    "anacrolix": f"{R}/engine-anacrolix/engine-anacrolix",
    "rain": f"{R}/engine-rain/engine-rain",
    "lt-go": f"{R}/engine-lt-go/engine-lt-go",
    "lt-rust": f"{R}/engine-lt-rust/target/release/engine-lt-rust",
}
FXROOT = os.environ.get("FXROOT", "/home/user/fx")
TB = int(os.environ.get("TBASE", "7600"))
EB = int(os.environ.get("EBASE", "18500"))
TP = [TB, TB + 1, TB + 2, TB + 3]
URL = lambda p: f"http://127.0.0.1:{p}/pk/announce"

SCENARIOS = {
    "pacing": dict(n=1000, run=540, tracker=["--interval", "120", "--min-interval", "120", "--enforce"]),
    "pacing-latency": dict(n=4000, run=540, tracker=["--interval", "300", "--min-interval", "300", "--enforce", "--delay-ms", "250"]),
    "tiers": dict(n=50, run=540, announce_list=f"{URL(TP[0])},{URL(TP[1])};{URL(TP[2])},{URL(TP[3])}",
                  tracker=["--interval", "120", "--min-interval", "120", "--enforce"]),
    "failure": dict(n=20, run=600, fail=True, tracker=["--interval", "120", "--min-interval", "120", "--enforce"]),
    "lifecycle": dict(n=1, run=None, pieces=64, tracker=["--interval", "1800", "--min-interval", "1800", "--enforce"]),
}


def fixtures(name, sc):
    d = f"{FXROOT}/sc-{name}-{TB}"
    if not os.path.exists(f"{d}/manifest.json"):
        cmd = [f"{R}/fixtures/fixtures", "gen", "--out", d, "--count", str(sc["n"]), "--piece-len", "4194304",
               "--pieces", str(sc.get("pieces", 1024)), "--announce", URL(TP[0]), "--name-prefix", f"{name}-"]
        if "announce_list" in sc:
            cmd += ["--announce-list", sc["announce_list"]]
        subprocess.run(cmd, check=True, stdout=subprocess.DEVNULL)
    return d


def wait_status(path, event, proc, timeout=600):
    end = time.time() + timeout
    while time.time() < end:
        if os.path.exists(path) and any(json.loads(l).get("event") == event for l in open(path)):
            return
        if proc.poll() is not None:
            raise SystemExit(f"engine exited before {event}")
        time.sleep(0.5)
    raise SystemExit(f"timeout waiting for {event}")


def main():
    engine, name, out = sys.argv[1], sys.argv[2], sys.argv[3]
    sc = SCENARIOS[name]
    os.makedirs(out, exist_ok=True)
    fx = fixtures(name, sc)
    targs = list(sc["tracker"])
    if sc.get("fail"):
        targs += ["--fail-infohash", ",".join(m["infohash"] for m in json.load(open(f"{fx}/manifest.json")))]
    tr = subprocess.Popen([f"{R}/tracker/tracker", "--log", f"{out}/tracker.jsonl", "--listen",
                           ",".join(f"127.0.0.1:{p}" for p in TP)] + targs,
                          stdout=open(f"{out}/tracker.out", "w"), stderr=subprocess.STDOUT)
    time.sleep(0.5)
    procs = []
    try:
        if name == "lifecycle":
            seeder = subprocess.Popen([BINS["anacrolix"], "--mode", "seed", "--fixtures", fx, "--listen-port", str(EB),
                                       "--peer-id-prefix", "-SEEDER-", "--status", f"{out}/seeder.jsonl"],
                                      stdout=open(f"{out}/seeder.out", "w"), stderr=subprocess.STDOUT)
            procs.append(seeder)
            wait_status(f"{out}/seeder.jsonl", "all_added", seeder)
            dd = f"{out}/leech-data"
            os.makedirs(dd, exist_ok=True)
            p = subprocess.Popen([BINS[engine], "--mode", "lifecycle", "--fixtures", fx, "--index", "0", "--data-dir", dd,
                                  "--listen-port", str(EB + 1), "--status", f"{out}/status.jsonl"],
                                 stdout=open(f"{out}/engine.out", "w"), stderr=subprocess.STDOUT)
            procs.append(p)
            p.wait(timeout=900)
        else:
            p = subprocess.Popen([BINS[engine], "--mode", "seed", "--fixtures", fx, "--listen-port", str(EB + 1),
                                  "--status", f"{out}/status.jsonl"],
                                 stdout=open(f"{out}/engine.out", "w"), stderr=subprocess.STDOUT)
            procs.append(p)
            wait_status(f"{out}/status.jsonl", "all_added", p)
            time.sleep(sc["run"])
            p.send_signal(signal.SIGTERM)
            p.wait(timeout=300)
    finally:
        for q in procs:
            if q.poll() is None:
                q.send_signal(signal.SIGTERM)
                q.wait(timeout=120)
        time.sleep(1)
        tr.send_signal(signal.SIGTERM)
        tr.wait()
    subprocess.run([sys.executable, f"{R}/measure/announces.py", f"{out}/tracker.jsonl"],
                   stdout=open(f"{out}/announces.txt", "w"))
    if name == "lifecycle":
        log = [json.loads(l) for l in open(f"{out}/tracker.jsonl")]
        t0 = log[0]["t"]
        with open(f"{out}/announces.txt", "a") as f:
            f.write("\nleecher timeline (s, event, left, violations, rejected):\n")
            for l in log:
                if not l["peer_id"].startswith("-SEEDER-"):
                    f.write(f"{l['t'] - t0:8.1f} {l['event'] or '(none)':10} left={l['left']} {l['violations']} {l['rejected']}\n")
            for s in open(f"{out}/status.jsonl"):
                s = json.loads(s)
                f.write(f"status {s['t'] - t0:8.1f} {s.get('event')} {s.get('step', s.get('name', ''))}\n")
    print(open(f"{out}/announces.txt").read())


main()
