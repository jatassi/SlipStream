import json, os, time, urllib.request, urllib.error, urllib.parse

C = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MEDIA = os.path.join(C, "media")
SEED = os.path.join(C, "seed")
EVID = os.path.join(C, "evidence")
os.makedirs(EVID, exist_ok=True)


def creds():
    d = {}
    for line in open(os.path.join(C, "creds.txt")):
        k, _, v = line.strip().partition("=")
        d[k] = v
    return d["user"], d["password"]


def req(method, url, body=None, headers=None, raw=False, timeout=60):
    h = {"Accept": "application/json"}
    h.update(headers or {})
    data = None
    if body is not None:
        if isinstance(body, (dict, list)):
            data = json.dumps(body).encode()
            h.setdefault("Content-Type", "application/json")
        else:
            data = body if isinstance(body, bytes) else body.encode()
    r = urllib.request.Request(url, data=data, method=method, headers=h)
    t = time.time()
    try:
        with urllib.request.urlopen(r, timeout=timeout) as resp:
            content, code = resp.read(), resp.status
    except urllib.error.HTTPError as e:
        content, code = e.read(), e.code
    ms = int((time.time() - t) * 1000)
    if raw:
        return code, content, ms
    try:
        out = json.loads(content) if content else None
    except ValueError:
        out = content.decode(errors="replace")
    return code, out, ms


class Log:
    def __init__(self, name):
        self.f = open(os.path.join(EVID, name + ".log"), "a")

    def __call__(self, *a):
        s = " ".join(str(x) for x in a)
        print(s, flush=True)
        self.f.write(time.strftime("%H:%M:%S ") + s + "\n")
        self.f.flush()


def poll(fn, timeout=120, every=1.0):
    t = time.time()
    while time.time() - t < timeout:
        v = fn()
        if v:
            return v, round(time.time() - t, 1)
        time.sleep(every)
    return None, round(time.time() - t, 1)


def copy(src, dst):
    os.makedirs(os.path.dirname(dst), exist_ok=True)
    with open(src, "rb") as a, open(dst, "wb") as b:
        b.write(a.read())


def trim(o, n=1500):
    s = json.dumps(o, indent=1) if not isinstance(o, str) else o
    return s if len(s) <= n else s[:n] + "...[trimmed]"
