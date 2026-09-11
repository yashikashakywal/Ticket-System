# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# No external Go modules are used (stdlib only), so this build needs no
# network access to a module proxy.
COPY go.mod ./
COPY main.go ./
COPY internal ./internal
COPY web ./web

RUN CGO_ENABLED=0 GOOS=linux go build -o /ticket-system .

# ---- Run stage ----
FROM alpine:3.20

RUN adduser -D -H appuser
WORKDIR /app
COPY --from=builder /ticket-system /app/ticket-system

USER appuser
EXPOSE 8080

ENTRYPOINT ["/app/ticket-system"]
