# Verification — 2026-10-13

## Passed

- `go test -race ./...`: OTP expiry, five-attempt limit, replay prevention, concurrent verification, cookie flags, logout revocation, origin checks, room authorization, hidden votes/aggregates, observer restriction, stale-round protection, lifecycle, concurrent votes, file persistence, realtime presence/connection limits and shutdown.
- `go vet ./...` and `go build ./cmd/server`.
- `TEST_DATABASE_URL=... go test -race ./internal/repositories -run TestPostgres -v`: 20 simultaneous transactions retained their data; a second repository connection read persisted state; failed OTP attempt counters remained committed. Tested against an isolated PostgreSQL 17.2 container.
- `python3 scripts/smoke.py http://0.0.0.0:8080`: real development server with file persistence; three authenticated users, two SSE streams, hidden votes before reveal, host-only controls, reset, estimate, closure and logout.
- The same HTTP/SSE smoke against port 8082 with the PostgreSQL adapter.
- `npm run build`: strict TypeScript and Vite production bundle; fonts packaged locally.
- `docker compose config --quiet`.
- Browser: phone → development OTP → profile → dashboard → room → task → vote → reveal → final estimate → close → logout.
- Desktop room and landing visual inspection; mobile at 390×844, no horizontal page overflow, selecting a card and revealing results, then viewport reset.

## Scope

- OTP is deliberately development-only for now, as requested; no live SMS provider, phone delivery or production deployment was tested.
- Dockerfile and Compose are included. Compose configuration and real PostgreSQL were tested; the complete Docker image was not built in this run.
- Runtime uses one API replica. PostgreSQL stores relational application tables transactionally (current sqlc/Goose implementation; see postgres-and-rooms.md); realtime subscriptions/presence live in the process.
- QA accounts and closed QA sessions created by smoke/browser checks are retained in ignored local development data. The isolated PostgreSQL test container is stopped and removed after verification.
- No Git commit was made.

Screenshots: [desktop room](screenshots/room.jpg), [mobile results](screenshots/mobile.jpg).

## LAN access configuration

- Detected workstation address: `192.168.15.87`; gateway: `192.168.14.1`.
- Go defaults, Compose and `.env.example` explicitly allow the workstation on ports 8080/5173 and keep localhost/127.0.0.1 access. Origins remain an exact allowlist.
- Vite binds to all interfaces and proxies `/api` to loopback. Invite copying provides a manual fallback on LAN HTTP where Clipboard API is unavailable.
- Focused Go config/HTTP tests and the React production build passed; Compose configuration validated.
- Live server verified from the workstation using its LAN address: frontend/health returned 200; allowed-origin login requests reached phone validation (400 `invalid_phone`); gateway/unrelated origins returned 403 `invalid_origin`. Another physical device was not tested.

## Shared room themes

- Six validated presets: violet, pink, blue, mint, amber and coral. Theme is stored in the room row; legacy rooms default to violet.
- `go test -race ./...` and `go vet ./...` passed. New tests cover creator-only changes, invalid values, rejected-command immutability, persistence after repository reopen, legacy defaults, broadcast notifications and preservation of active votes.
- Extended HTTP/SSE smoke passed: pink creation, guest rejection, invalid-theme rejection and blue/pink updates received by another authenticated client.
- React production build passed. Browser verified all six presets, pink light/dark appearance, actual pink-room creation and persistence after reload, settings changes and mobile picker at 390×844 without horizontal overflow. Temporary viewport was reset.
- Updated app is running on the existing port 8080. Existing sessions/data/environment were preserved during restart; QA sessions were closed and browser QA account logged out.

Images: [pink room](screenshots/room-pink.jpg), [pink dark room](screenshots/room-pink-dark.jpg), [mobile theme picker](screenshots/theme-picker-mobile.jpg).
