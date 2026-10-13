# API

Base: `/api`. Authentication: `poker_session` cookie (`HttpOnly`, `SameSite=Strict`, `Secure` in production). Mutations use JSON and same-origin requests. Sessions are not sent in query strings.

| Method | Path | Body / result |
| --- | --- | --- |
| GET | `/health` | `{status, environment}` |
| POST | `/auth/otp` | `{phone}` → `{phone, expires_in:180, retry_after:60, dev_code?}` |
| POST | `/auth/verify` | `{phone, code}` → profile + session cookie |
| POST | `/auth/logout` | `{}` → 204, session revoked |
| GET | `/me` | Own `{id, phone, name, avatar, avatar_config?, role, created_at}` |
| PATCH | `/me` | `{name, avatar, role, avatar_config?}` → updated own profile |
| GET | `/rooms` | Rooms where current user is a member |
| GET | `/rooms/public?limit=12&offset=0` | Public, open room summaries; no authentication required; limit 1–50 |
| POST | `/rooms` | `{name, deck:"fibonacci"\|"tshirt", theme?, visibility?:"public"\|"private"}` → created room, 201 |
| POST | `/rooms/:id/join` | `{observer:false}` → room snapshot; authenticated user joins a public room directly or a private room through its invitation URL |
| GET | `/rooms/:id` | Sanitized room snapshot; members only |
| GET | `/rooms/:id/events` | SSE room snapshots; terminal `closed`, `idle`, `expired` or `unavailable` ends the stream; already-closed room returns 204 |
| POST | `/rooms/:id/commands` | Command below → current room snapshot |

## Commands

Voting requires at least one authenticated live SSE connection in that room; otherwise it returns `409 disconnected`. Only `vote` is allowed for voting members; all other commands require the host. `set_theme` is allowed for the creator even after a room is closed; other commands remain blocked.

```json
{"action":"add_round","title":"ورود با موبایل","description":"معیار پذیرش…"}
{"action":"start_round","round_id":"<pending-round-id>"}
{"action":"vote","round_id":"<active-round-id>","value":"5"}
{"action":"vote","round_id":"<active-round-id>","value":""}
{"action":"reveal"}
{"action":"reset"}
{"action":"estimate","value":"5"}
{"action":"set_theme","value":"pink"}
{"action":"set_visibility","value":"public"}
{"action":"close"}
```

The first added round starts immediately. `start_round` requires the previous active round to be completed. `vote` requires the current round ID and a valid card. `reveal` requires at least one vote. `reset` clears votes on the active voting/revealed round. `estimate` requires reveal and a numeric/T-shirt card, excluding `?` and `☕`.

## Room snapshot

```json
{
  "id":"<room-id>", "name":"Sprint", "host_id":"<user-id>",
  "deck":"fibonacci", "theme":"pink", "visibility":"private", "last_activity_at":"<ISO-8601>", "cards":["0","1","2","3","5","8","13","21","34","?","☕"],
  "active_round_id":"<round-id>", "closed":false, "revision":3,
  "my_vote":"5", "average":0, "consensus":false, "distribution":{},
  "members":[{"id":"<user-id>","name":"پارسا","avatar":"violet","role":"Developer","observer":false,"online":true,"has_voted":true}],
  "rounds":[{"id":"<round-id>","title":"Task","description":"Details","status":"voting","estimate":"","started_at":"<ISO-8601>"}],
  "created_at":"<ISO-8601>"
}
```

Before reveal, `members[].vote` is omitted for everyone, including host, and aggregates remain empty/zero. `my_vote` belongs only to the requesting user. After reveal, participant votes are included; distribution, average and consensus count connected non-observers only. Consensus requires all eligible voters to vote the same valid estimate (at least two voters). Completed rounds and closed rooms preserve historical results. Phone numbers are only returned by `/me` and login verification. Round history includes titles and saved final estimates.

Errors: `{ "error": { "code": "host_required", "message": "…" } }`. Codes use HTTP 400 validation, 401 authentication/OTP, 403 membership/role/origin, 404 unknown room/round, 409 lifecycle conflict, 429 rate limit, 502 SMS failure. Clients apply snapshots with `revision >= current`; SSE refreshes snapshots every 10 seconds. Network failures may reconnect; terminal events close the browser EventSource and stop automatic reconnecting. An idle/unavailable stream offers manual reconnection.

Room themes: `violet` (default), `pink`, `blue`, `mint`, `amber`, `coral`. Omitting `theme` on creation keeps compatibility with existing clients. Older stored rooms return `violet`. Theme changes increment the room revision, persist through the repository and broadcast via SSE; they do not reset votes. Unknown themes return 400 `invalid_theme`, and non-creators receive 403 `host_required`.

## Character avatars

`avatar` remains the background color (`violet`, `mint`, `peach`, `blue`, `pink`, `yellow`). Optional `avatar_config` is returned in the user's profile and `members[]`, and persists with the existing repository. Profile changes broadcast fresh room snapshots through SSE.

```json
{
  "name": "پارسا",
  "role": "Developer",
  "avatar": "pink",
  "avatar_config": {
    "version": 1,
    "face": "square",
    "skin": "warm",
    "hair": "swoop",
    "hair_color": "brown",
    "expression": "wink",
    "glasses": "none",
    "shirt": "mint"
  }
}
```

| Field | Allowed values |
| --- | --- |
| `version` | `1` |
| `face` | `round`, `square`, `oval` |
| `skin` | `porcelain`, `warm`, `sand`, `tan`, `deep`, `rich` |
| `hair` | `crop`, `swoop`, `bob`, `curls`, `bun`, `bald`, `scarf` |
| `hair_color` | `espresso`, `brown`, `gold`, `copper`, `silver`, `pink` |
| `expression` | `smile`, `laugh`, `wink`, `calm` |
| `glasses` | `none`, `round`, `square` |
| `shirt` | `violet`, `pink`, `mint`, `blue`, `amber`, `coral` |

A supplied config must contain all fields with valid values; invalid/partial configs return 400 `invalid_avatar_config` without changing the profile. Omission or `null` preserves an existing custom avatar for older clients. Users without a saved config receive a deterministic local illustration in React. No image URL, SVG markup or upload is accepted.

## Public/private rooms and stream lifetime

Visibility defaults to `private`, including imported legacy rooms. Only the host can run `set_visibility` on an open room. Private room IDs remain unguessable invitation URLs; private rooms never appear in discovery. Room snapshots remain member-only in both modes. Public joining still requires authentication, a completed profile and capacity below 20 connected people. Membership history does not occupy a seat; the SSE subscription enforces the limit atomically, including reconnects. Multiple tabs of one person use one seat.

Discovery returns `[{id,name,deck,theme,host_name,member_count,capacity:20,created_at}]` with `member_count` counting connected people, ordered newest first with ID as the tie breaker. It does not return member identities, phones, task contents or votes. Closed rooms are excluded. Home refreshes discovery every 30 seconds while visible, on returning to the tab, and via its refresh button.

Closing sends the final `room` snapshot (`closed:true`) followed by `event: closed`, ends the HTTP body and unsubscribes all finished listeners. SSE requests for an already-closed room return 204. After `realtime.idle_timeout` (default `24h`) without a successful room command, member join or intentional opening/reconnection, `event: idle` terminates the stream. Heartbeats, list calls and read-only snapshots do not renew activity. The room remains open and can be resumed. Logout/session expiration emits `expired`; transport/storage failure emits `unavailable`; application shutdown also ends writers. Browser navigation/unmount closes EventSource. Failed/slow writes are bounded to five seconds.

## Browser connection and presence

An open room renders the table and voting controls only after an authenticated SSE `room` snapshot confirms the current user's `online` presence. Network errors, the browser offline event, terminal SSE events and a 45-second missing-snapshot watchdog show a dedicated disconnected screen. Native EventSource retry or browser online recovery can restore the room; a retry button starts a fresh connection. Session expiry requests login. Background tabs keep their connections; `pagehide` closes the stream and a restored browser-history page reconnects.

Active participants and seats include `members[].online` only. Persistent membership and saved votes permit a return without duplication. Closing one tab preserves presence while another tab remains connected; closing the last releases the seat when the server detects the broken stream. Server detection follows the 10-second heartbeat and TCP write failure (it is not instantaneous for other participants). Historical completed results remain available after all members leave.
