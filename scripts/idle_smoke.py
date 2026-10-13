#!/usr/bin/env python3
"""Verify actual SSE EOF after 25h inactivity, using an isolated local QA DB."""
import http.cookiejar
import json
import os
import queue
import re
import secrets
import subprocess
import sys
import threading
import urllib.parse
import urllib.request

base = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8083").rstrip("/")
dsn = os.environ.get("TEST_DATABASE_URL", "")
db = urllib.parse.urlparse(dsn)
assert urllib.parse.urlparse(base).hostname in ("localhost", "127.0.0.1"), "Use local QA only"
assert db.hostname in ("localhost", "127.0.0.1") and db.path.endswith("_qa"), "Use a dedicated _qa database"
client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

def request(path, method="GET", body=None):
    req = urllib.request.Request(base + "/api" + path, method=method,
        data=None if method == "GET" else json.dumps(body or {}).encode(),
        headers={"Content-Type": "application/json"})
    with client.open(req, timeout=20) as response:
        b = response.read()
        return json.loads(b) if b else None

assert request("/health")["environment"] == "development"
phone = "0912" + "".join(str(secrets.randbelow(10)) for _ in range(7))
challenge = request("/auth/otp", "POST", {"phone": phone})
request("/auth/verify", "POST", {"phone": phone, "code": challenge["dev_code"]})
request("/me", "PATCH", {"name": "QA Idle", "avatar": "pink", "role": "QA"})
room = request("/rooms", "POST", {"name": "QA idle stream", "deck": "fibonacci", "visibility": "private"})
rid = room["id"]
assert re.fullmatch(r"[a-f0-9]{32}", rid)
path = "/rooms/" + rid

def stream():
    snapshots = queue.Queue()
    terminal = []
    finished = threading.Event()
    errors = []
    def worker():
        try:
            with client.open(base + "/api" + path + "/events", timeout=20) as response:
                event = ""
                for line in response:
                    if line.startswith(b"event: "):
                        event = line[7:].decode().strip()
                    elif line.startswith(b"data: "):
                        if event == "room": snapshots.put(json.loads(line[6:]))
                        else: terminal.append(event)
        except Exception as e:
            errors.append(e)
        finally:
            finished.set()
    threading.Thread(target=worker, daemon=True).start()
    return snapshots, terminal, finished, errors

snapshots, terminal, finished, errors = stream()
assert snapshots.get(timeout=5)["id"] == rid
env = os.environ.copy()
env.update(PGHOST=db.hostname, PGPORT=str(db.port or 5432), PGUSER=db.username,
    PGPASSWORD=urllib.parse.unquote(db.password or ""), PGDATABASE=db.path.lstrip("/"))
# Only the newly-created synthetic room is aged; all other room data is untouched.
subprocess.run(["psql", "-X", "-v", "ON_ERROR_STOP=1", "-c",
    "UPDATE rooms SET last_activity_at=now()-interval '25 hours' WHERE id='" + rid + "'"],
    env=env, check=True, capture_output=True, timeout=5)
assert finished.wait(15), "Idle SSE never reached EOF"
assert not errors and "idle" in terminal, (errors, terminal)
snapshot = request(path)
assert not snapshot["closed"] and not any(m["online"] for m in snapshot["members"])
print("PASS: 25h-inactive room emitted idle, HTTP response reached EOF, subscription released")

snapshots, terminal, finished, errors = stream()
assert snapshots.get(timeout=5)["id"] == rid, "Room cannot resume after inactivity"
request(path + "/commands", "POST", {"action": "close"})
assert finished.wait(5) and not errors and "closed" in terminal, (errors, terminal)
request("/auth/logout", "POST")
print("PASS: manual reopening renewed activity; room closure ended the resumed stream")
