"""Minimal stdlib websocket client. Usage: ws.py plex|jellyfin|emby  -> appends JSON lines to evidence/ws-<kind>.jsonl"""
import base64, json, os, socket, struct, sys, time, threading
from lib import EVID


def connect(host, port, path, extra=""):
    s = socket.create_connection((host, port))
    key = base64.b64encode(os.urandom(16)).decode()
    s.sendall((f"GET {path} HTTP/1.1\r\nHost: {host}:{port}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
               f"Sec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n{extra}\r\n").encode())
    buf = b""
    while b"\r\n\r\n" not in buf:
        buf += s.recv(4096)
    head, _, rest = buf.partition(b"\r\n\r\n")
    status = head.split(b"\r\n")[0].decode()
    return s, status, rest


def send(s, text, opcode=1):
    p = text.encode() if isinstance(text, str) else text
    mask = os.urandom(4)
    hdr = bytes([0x80 | opcode])
    n = len(p)
    if n < 126:
        hdr += bytes([0x80 | n])
    elif n < 65536:
        hdr += bytes([0x80 | 126]) + struct.pack(">H", n)
    else:
        hdr += bytes([0x80 | 127]) + struct.pack(">Q", n)
    s.sendall(hdr + mask + bytes(b ^ mask[i % 4] for i, b in enumerate(p)))


class Reader:
    def __init__(self, s, rest):
        self.s, self.buf = s, rest

    def need(self, n):
        while len(self.buf) < n:
            d = self.s.recv(65536)
            if not d:
                raise EOFError
            self.buf += d
        out, self.buf = self.buf[:n], self.buf[n:]
        return out

    def frame(self):
        b0, b1 = self.need(2)
        op, n = b0 & 0x0F, b1 & 0x7F
        if n == 126:
            n = struct.unpack(">H", self.need(2))[0]
        elif n == 127:
            n = struct.unpack(">Q", self.need(8))[0]
        if b1 & 0x80:
            m = self.need(4)
            return op, bytes(b ^ m[i % 4] for i, b in enumerate(self.need(n)))
        return op, self.need(n)


def main(kind, token=None, devid="conformance-ws"):
    out = open(os.path.join(EVID, f"ws-{kind}.jsonl"), "a")
    if kind == "plex":
        host, port, path = "localhost", 32400, "/:/websockets/notifications"
    elif kind == "jellyfin":
        host, port, path = "localhost", 8096, "/socket"
    else:
        host, port, path = "localhost", 8097, f"/embywebsocket?api_key={token}&deviceId={devid}"
    extra = f'Authorization: MediaBrowser Client="conformance", Device="ws", DeviceId="{devid}", Version="1.0", Token="{token}"\r\n' if token else ""
    s, status, rest = connect(host, port, path, extra)
    out.write(json.dumps({"t": time.time(), "connect": status}) + "\n"); out.flush()
    print(kind, status, flush=True)
    if kind != "plex":
        send(s, json.dumps({"MessageType": "SessionsStart", "Data": "0,1500"}))

        def keep():
            while True:
                time.sleep(25)
                try:
                    send(s, json.dumps({"MessageType": "KeepAlive"}))
                except OSError:
                    return
        threading.Thread(target=keep, daemon=True).start()
    r = Reader(s, rest)
    while True:
        op, data = r.frame()
        if op == 9:
            send(s, data, 10); continue
        if op == 8:
            break
        try:
            msg = json.loads(data)
        except ValueError:
            msg = data.decode(errors="replace")
        out.write(json.dumps({"t": time.time(), "msg": msg}) + "\n"); out.flush()


if __name__ == "__main__":
    kind = sys.argv[1]
    tok = None
    if kind != "plex":
        from mb import MB
        m = MB(kind); m.devid = "conformance-ws"; m.login(); tok = m.token
    main(kind, tok)
