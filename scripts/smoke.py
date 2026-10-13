#!/usr/bin/env python3
"""Real HTTP + SSE verification against a local development server only."""

import http.cookiejar
import json
import queue
import secrets
import sys
import threading
import urllib.error
import urllib.request

base = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080").rstrip("/")
assert (
    urllib.request.urlopen(base + "/api/health", timeout=5).read().find(b"development")
    >= 0
), "Smoke requires development mode"


def client():
    return urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar())
    )


def request(c, path, method="GET", body=None, status=200):
    req = urllib.request.Request(
        base + "/api" + path,
        data=None if method == "GET" else json.dumps(body or {}).encode(),
        method=method,
        headers={"Content-Type": "application/json"},
    )
    try:
        resp = c.open(req, timeout=8)
    except urllib.error.HTTPError as exc:
        resp = exc
    with resp:
        assert resp.status == status, (
            f"{method} {path}: got {resp.status}, expected {status}"
        )
        data = resp.read()
        return json.loads(data) if data else None


users = []
for name in ["Host", "Voter", "Observer"]:
    c = client()
    phone = "0912" + "".join(str(secrets.randbelow(10)) for _ in range(7))
    ch = request(c, "/auth/otp", "POST", {"phone": phone})
    user = request(c, "/auth/verify", "POST", {"phone": phone, "code": ch["dev_code"]})
    request(c, "/me", "PATCH", {"name": "QA " + name, "avatar": "mint", "role": "QA"})
    request(c, "/auth/verify", "POST", {"phone": phone, "code": ch["dev_code"]}, 401)
    users.append((c, user))
host, guest, observer = (u[0] for u in users)
room = request(host, "/rooms", "POST", {"name": "QA smoke", "deck": "fibonacci", "theme": "pink"}, 201)
room_id = room["id"]
path = "/rooms/" + room_id
request(guest, path, status=403)
request(guest, path + "/join", "POST", {"observer": False})
request(observer, path + "/join", "POST", {"observer": True})
request(
    guest,
    path + "/commands",
    "POST",
    {"action": "add_round", "title": "Forbidden"},
    403,
)
room = request(
    host,
    path + "/commands",
    "POST",
    {
        "action": "add_round",
        "title": "QA realtime OTP",
        "description": "Two authenticated clients and an observer",
    },
)
round_id = room["active_round_id"]
streams = []
stream_finished = []
terminal_events = []


def stream(c):
    q = queue.Queue()
    finished = threading.Event()
    terminal = []
    stream_finished.append(finished)
    terminal_events.append(terminal)

    def worker():
        try:
            response = c.open(base + "/api" + path + "/events", timeout=15)
            streams.append(response)
            event = ""
            for line in response:
                if line.startswith(b"event: "):
                    event = line[7:].decode().strip()
                elif line.startswith(b"data: ") and event == "room":
                    q.put(json.loads(line[6:]))
                elif line.startswith(b"data: "):
                    terminal.append(event)
            response.close()
        except Exception as exc:
            q.put(exc)
        finally:
            finished.set()

    threading.Thread(target=worker, daemon=True).start()
    return q


host_events = stream(host)
guest_events = stream(guest)


def wait_for(q, predicate):
    for _ in range(40):
        item = q.get(timeout=12)
        if isinstance(item, Exception):
            raise item
        if predicate(item):
            return item
    raise AssertionError("Expected realtime snapshot was not delivered")


wait_for(host_events, lambda r: r["id"] == room_id)
wait_for(guest_events, lambda r: r["id"] == room_id)
print("PASS: OTP login, one-time code, profiles, room membership, authenticated SSE")
anonymous = client()
assert room_id not in [r["id"] for r in request(anonymous, "/rooms/public?limit=50")]
request(guest, path + "/commands", "POST", {"action": "set_visibility", "value": "public"}, 403)
request(host, path + "/commands", "POST", {"action": "set_visibility", "value": "unknown"}, 400)
request(host, path + "/commands", "POST", {"action": "set_visibility", "value": "public"})
wait_for(guest_events, lambda r: r["visibility"] == "public")
listed = next(r for r in request(anonymous, "/rooms/public?limit=50") if r["id"] == room_id)
assert listed["member_count"] == 2 and "members" not in listed and "phone" not in json.dumps(listed)
request(host, path + "/commands", "POST", {"action": "set_visibility", "value": "private"})
wait_for(guest_events, lambda r: r["visibility"] == "private")
assert room_id not in [r["id"] for r in request(anonymous, "/rooms/public?limit=50")]
request(host, path + "/commands", "POST", {"action": "set_visibility", "value": "public"})
print("PASS: anonymous public discovery, private exclusion, host-only visibility and realtime change")
assert request(guest, path)["theme"] == "pink"
request(guest, path + "/commands", "POST", {"action": "set_theme", "value": "blue"}, 403)
request(host, path + "/commands", "POST", {"action": "set_theme", "value": "unknown"}, 400)
request(host, path + "/commands", "POST", {"action": "set_theme", "value": "blue"})
wait_for(guest_events, lambda r: r["theme"] == "blue")
request(host, path + "/commands", "POST", {"action": "set_theme", "value": "pink"})
wait_for(guest_events, lambda r: r["theme"] == "pink")
print("PASS: pink room creation, host-only theme change, invalid theme rejection, guest SSE theme updates")
request(
    observer,
    path + "/commands",
    "POST",
    {"action": "vote", "value": "8", "round_id": round_id},
    403,
)
request(
    guest,
    path + "/commands",
    "POST",
    {"action": "vote", "value": "8", "round_id": "stale"},
    409,
)
request(
    guest,
    path + "/commands",
    "POST",
    {"action": "vote", "value": "8", "round_id": round_id},
)
snapshot = wait_for(
    host_events,
    lambda r: any(
        m["id"] == users[1][1]["id"] and m["has_voted"] for m in r["members"]
    ),
)
assert all(not m.get("vote") for m in snapshot["members"]), "Hidden vote leaked in SSE"
assert snapshot["average"] == 0 and snapshot["distribution"] == {}, (
    "Hidden aggregates leaked"
)
assert not any("phone" in m for m in snapshot["members"]), "Member phone leaked"
assert request(guest, path)["my_vote"] == "8"
request(guest, path + "/commands", "POST", {"action": "reveal"}, 403)
request(
    host,
    path + "/commands",
    "POST",
    {"action": "vote", "value": "5", "round_id": round_id},
)
request(host, path + "/commands", "POST", {"action": "reveal"})
snapshot = wait_for(guest_events, lambda r: r["rounds"][0]["status"] == "revealed")
assert snapshot["distribution"] == {"5": 1, "8": 1} and snapshot["average"] == 6.5
print(
    "PASS: realtime hidden votes, observer restriction, stale round guard, host-only reveal"
)
request(
    guest,
    path + "/commands",
    "POST",
    {"action": "vote", "value": "13", "round_id": round_id},
    409,
)
request(host, path + "/commands", "POST", {"action": "reset"})
snapshot = wait_for(guest_events, lambda r: r["rounds"][0]["status"] == "voting")
assert not any(m["has_voted"] for m in snapshot["members"])
request(
    host,
    path + "/commands",
    "POST",
    {"action": "vote", "value": "5", "round_id": round_id},
)
request(host, path + "/commands", "POST", {"action": "reveal"})
request(host, path + "/commands", "POST", {"action": "estimate", "value": "5"})
request(host, path + "/commands", "POST", {"action": "close"})
wait_for(host_events, lambda r: r["closed"])
wait_for(guest_events, lambda r: r["closed"])
assert all(done.wait(5) for done in stream_finished), "SSE response never ended after room closure"
assert all("closed" in events for events in terminal_events), "Missing terminal event"
assert not any(m["online"] for m in request(host, path)["members"]), "SSE subscriptions leaked"
request(host, path + "/events", status=204)
assert room_id not in [r["id"] for r in request(anonymous, "/rooms/public?limit=50")]
print("PASS: both SSE responses reached EOF, presence released, closed-room events return 204")
assert request(host, path)["rounds"][0]["estimate"] == "5"
request(
    guest,
    path + "/commands",
    "POST",
    {"action": "vote", "value": "5", "round_id": round_id},
    409,
)
request(guest, "/auth/logout", "POST", status=204)
request(guest, "/me", status=401)
print("PASS: reset, final estimate, closed room, persistent result, logout revocation")
print("All HTTP/SSE smoke checks passed. QA room retained as a closed session.")
