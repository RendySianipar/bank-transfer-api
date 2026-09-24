# ---- Stage 1: build ----
# Includes the full Go toolchain (~800MB+) - only needed to COMPILE the
# binary. None of this ships in the final image.
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache tzdata

WORKDIR /app

# Copy just go.mod/go.sum first and download dependencies BEFORE copying
# the rest of the source. Docker caches each instruction as a layer -
# as long as go.mod/go.sum don't change, this layer is reused on
# rebuilds even if you've only edited .go files, so `docker compose
# build` after a small code change is fast instead of re-downloading
# every dependency every time.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produces a fully static binary (no dependency on glibc/
# musl at runtime) - safe here because go-sql-driver/mysql is pure Go,
# no cgo required. GOOS=linux ensures a Linux binary regardless of what
# OS you're building on (relevant since you're on Windows).
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# ---- Stage 2: run ----
# alpine is ~5MB, vs golang:1.26-alpine's ~350MB+ with the full
# toolchain. This is the image that actually gets deployed/run.
FROM alpine:3.20

WORKDIR /app

# ca-certificates is needed if the app ever makes outbound HTTPS calls
# (not currently, but cheap to include and commonly needed later).
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server .

EXPOSE 7070

CMD ["./server"]
