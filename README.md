# Ticket System (Golang)

A small backend service where a user can register, log in, create tickets,
view only their own tickets, and update the status of their own tickets.
Built for the Backend Intern two-day assignment. Includes a lightweight
web UI on top of the same API, served from the same binary and URL.

## Frontend

`web/index.html` is a single-file, dependency-free HTML/CSS/JS page that
talks to the REST API below via `fetch`. It's compiled into the Go binary
with `go:embed` and served at `/`, so there's nothing separate to deploy or
host — the same URL that serves `/health` also serves the UI.

- Register or log in (JWT is kept in `localStorage`).
- File a new ticket with a title and optional description.
- See only your own tickets, newest first.
- Advance a ticket's status one step at a time (`open → in_progress →
  closed`); once closed, no further action is offered, matching the API's
  status-flow rule.

If you'd rather exercise the API directly instead of through the UI, see
the **API** section further down, or use the Postman collection in
`postman/`.

## Tech notes

- **Language:** Go 1.22, standard library only — no external Go modules.
  Auth uses a small hand-rolled HS256 JWT implementation and PBKDF2-HMAC-SHA256
  password hashing, both built on `crypto/hmac` and `crypto/sha256` from the
  standard library. This keeps the Docker build fully self-contained (no
  `go.sum`, no module-proxy network access required at build time).
- **Storage:** in-memory, guarded by a mutex. Data resets on restart — this
  is explicitly allowed by the assignment scope ("No complex database schema
  is required. Use in-memory storage..."). Swapping in Postgres/SQLite later
  would only mean replacing `internal/store` behind the same interface used
  by the handlers.
- **Routing:** Go 1.22's `net/http.ServeMux` with method + path-parameter
  patterns (e.g. `PATCH /tickets/{id}/status`) — no router dependency needed.

## Local run

```bash
go run .
# or build a binary:
go build -o ticket-system .
JWT_SECRET=some-long-random-secret ./ticket-system
```

The server listens on port **8080** by default (`PORT` env var to override).
Open `http://localhost:8080` in a browser to use the web UI, or hit the API
paths below directly.

## Docker

```bash
docker build -t ticket-system .
docker run -p 8080:8080 -e JWT_SECRET=some-long-random-secret ticket-system
curl http://localhost:8080/health
```

Expected response:

```json
{"status": "ok"}
```

## Environment variables

See `.env.example`.

| Variable     | Default                        | Description                          |
|--------------|---------------------------------|---------------------------------------|
| `PORT`       | `8080`                          | HTTP port to listen on                |
| `JWT_SECRET` | `dev-secret-change-me` (unsafe) | HMAC secret used to sign/verify JWTs  |

**Always set `JWT_SECRET` explicitly outside of local dev** — the server logs
a warning and falls back to an insecure default if it's unset.

## API

All request/response bodies are JSON.

| Method | Endpoint               | Auth required | Purpose                    |
|--------|-------------------------|:---:|-----------------------------|
| GET    | `/health`               | No  | Health check                 |
| POST   | `/auth/register`        | No  | Register a new user          |
| POST   | `/auth/login`           | No  | Log in, returns a JWT        |
| POST   | `/tickets`               | Yes | Create a ticket               |
| GET    | `/tickets`               | Yes | List the caller's tickets     |
| GET    | `/tickets/{id}`          | Yes | Get one of the caller's tickets |
| PATCH  | `/tickets/{id}/status`   | Yes | Update the status of the caller's ticket |

Protected endpoints require `Authorization: Bearer <token>`.

### `POST /auth/register`

```json
// request
{ "email": "alice@example.com", "password": "secret123" }
```
Returns `201` with `{ "id": "...", "email": "..." }`, `400` for invalid
input (missing/invalid email, password under 6 characters), or `409` if the
email is already registered.

### `POST /auth/login`

```json
// request
{ "email": "alice@example.com", "password": "secret123" }
```
Returns `200` with `{ "token": "<jwt>" }`, or `401` for invalid credentials.

### `POST /tickets`

```json
// request
{ "title": "Printer broken", "description": "Not turning on" }
```
Returns `201` with the created ticket (`status` starts as `"open"`), or
`400` if `title` is missing.

### `GET /tickets`

Returns `200` with a JSON array of the caller's own tickets (newest first).

### `GET /tickets/{id}`

Returns `200` with the ticket if it exists and belongs to the caller, `404`
if no ticket with that ID exists, or `403` if it exists but belongs to a
different user.

### `PATCH /tickets/{id}/status`

```json
// request
{ "status": "in_progress" }
```
Returns `200` with the updated ticket. Ownership errors mirror `GET
/tickets/{id}` (`404` / `403`). Returns `400` for an unsupported status
value or for a transition that doesn't follow the required flow below.

## Status flow

```
open -> in_progress -> closed
```

- Only the next step in that sequence is accepted in a single call (e.g.
  `open -> closed` directly is rejected as `400`, same as `in_progress ->
  open`).
- `closed` is terminal: any further update to a closed ticket is rejected
  with `400`.

## Assumptions

- The status flow is enforced strictly and linearly (one step forward at a
  time); skipping a step (e.g. `open` straight to `closed`) is treated as an
  invalid transition rather than allowed shorthand, since the brief shows the
  flow as a specific sequence.
- Viewing or updating a ticket that exists but belongs to another user
  returns `403 Forbidden`; a ticket ID that doesn't exist at all returns
  `404 Not Found`. This distinguishes "not yours" from "doesn't exist" for
  easier debugging/testing, at the cost of confirming a ticket ID exists to
  a non-owner.
- Emails are treated as case-insensitive and trimmed of surrounding
  whitespace for both registration and login.
- Passwords must be at least 6 characters. This isn't specified by the
  brief; it's a minimal sanity check rather than a strict policy.
- JWTs are valid for 24 hours from issuance; there is no refresh-token flow
  since the brief doesn't call for one.
- No pagination is implemented on `GET /tickets`, since the brief doesn't
  request it and ticket volume per user is expected to be small.

## Deployment

- **GitHub repository:** https://github.com/yashikashakywal/Ticket-System)
- **Deployed application URL:** https://ticket-system-qoxg.onrender.com/)
- **Public health check URL:** https://ticket-system-qoxg.onrender.com/health

Any free-tier platform that can run a Docker image or a Go binary works
(e.g. Render, Railway, Fly.io, Koyeb). The general steps are:

1. Push this repository to GitHub.
2. Create a new Web Service on the chosen platform, pointing it at the repo
   and telling it to build from the included `Dockerfile`.
3. Set the `JWT_SECRET` environment variable in the platform's dashboard to
   a long random value (do **not** reuse the local dev default).
4. Make sure the platform exposes/maps port `8080` (most platforms auto-detect
   `EXPOSE 8080` from the Dockerfile, or let you set a `PORT` env var — this
   app already reads `PORT` if the platform sets one).
5. Once deployed, confirm `GET <deployed-url>/health` is publicly reachable
   and returns `{"status": "ok"}`.
