# AGENTS.md

Terminal Gomoku in Go: a TUI client (`gotermoku`), a central gRPC server
(`gotermoku-server`), LAN P2P (client acts as gRPC host), and offline/bot modes.

## Trust the code, not the README

`README.md` describes an older **single-package, TCP** layout (`main.go`,
`engine.go`, etc. at repo root). The real layout is `cmd/` + `internal/`, and all
networking is gRPC. `go build .` at the repo root fails with "no Go files".
Use the Makefile or build the `cmd/...` packages.

## Commands

```sh
make all        # build ./gotermoku and ./gotermoku-server (linux/amd64, CGO off)
make client     # build only the client
make server     # build only the server
make test       # go test -v -race ./test/...
make clean      # remove both root binaries

# single integration test
go test -v -race ./test -run TestLAN_MultiplayerIntegration
```

- Tests live only in `./test` (package `test`); there are no unit tests.
  They start real gRPC servers (port 0 / `t.TempDir()` bbolt) and rely on
  timing; always keep `-race`.
- `make proto` needs `protoc` plus `protoc-gen-go` / `protoc-gen-go-grpc`.
  Generated code is committed in `internal/common/pb`; do not hand-edit it.
  Regenerate only when `api/proto/gomoku.proto` changes.

## Layout

- `internal/common/core` — shared pure logic: `GameState`, win/ray scanning,
  Elo. `Board` is indexed **`Board[y][x]`** (row-major); pieces are
  `uint8` (`Empty`, `White`, `Black`, `Draw`). White always moves first.
- `internal/client` — CLI flags (`config.go`), event loop (`engine.go`),
  input, TUI renderer (uses `github.com/ttasc/ttbox`), AI (`bot.go`), and the
  `Transport` abstraction (`transport.go`). Implementations: `transport_lan.go`
  (P2P) and `transport_central.go` (central server).
- `internal/server` — gRPC central server: auth (`auth.go`),
  matchmaking/private rooms/rejoin (`matchmaker.go`, `room.go`), bbolt storage
  (`storage.go`), interceptors (`interceptor.go`).
- `cmd/gotermoku`, `cmd/gotermoku-server` — thin `main` wrappers calling
  `client.Run` / `server.Run`.

## Architecture constraints

- Authority: LAN **host** and central **server** own the authoritative
  `GameState`. Clients send `move`/`restart`; peers receive full `sync`
  snapshots. Do not add client-side rules that can diverge.
- Wire messages use string types: `sync`, `move`, `restart`, `disconnect`,
  `pause`, `resume`. `NetMessage` (`internal/client/transport.go`) is the
  client-side view; changing it usually requires updating **both** transport
  implementations and the server's `toGameSyncLocked` mapping in
  `internal/server/room.go`.
- `api/proto/gomoku.proto` is the wire-format source of truth. Central-mode
  auth uses gRPC metadata `authorization: Bearer <token>`; interceptors exempt
  only `Register`, `Login`, and `PlayStream` (unauthenticated LAN P2P).
- Server behavior worth knowing: Elo matching starts at ±100 and widens by 50
  per 5s wait; private room codes are 6 chars; a dropped player gets a 45s
  reconnect grace period before the room closes; `--tls` without `--cert/--key`
  generates an in-memory ECDSA cert (clients use `--tls-insecure` for it).
- Client modes are mutually exclusive: `--server` cannot combine with
  `--bot/--host/--join`; `--create-room`, `--join-room`, `--rejoin` require
  `--server`. Board size is `ROWSxCOLS`, minimum 3x3. Server defaults are
  20x30, port 3333, DB `gotermoku.db`.

## Conventions / gotchas

- Comments, docs, and user-facing log strings are frequently Vietnamese. Match
  the surrounding file; don't mass-translate.
- `doc/test/script.md` is the end-to-end manual QA script for all modes
  (offline, bot, LAN, central Elo, private rooms, rejoin, TLS). Use it when
  verifying feature changes; `doc/test/init.sh` is a deployment helper.
- `doc/handonlab/grpctypes` is a standalone gRPC lab (not part of the game):
  proto + generated stubs + `server/main.go` + `client/main.go`. Regenerate with
  `make lab-proto`; run with `go run ./doc/handonlab/grpctypes/{server,client}`.
  Handout + implementation notes: `doc/handonlab/{README,NOTES}.md`; rubric
  submission: `doc/handonlab/docs.md`.
- `doc/handonlab/grpc-python` implements the same `profile.proto` in Python for
  Go↔Python interop. Stubs are committed; regenerate with `make lab-py-proto`
  (needs a venv + `grpc-python/requirements.txt`). Test both directions after
  proto changes.
- Root binaries `gotermoku`/`gotermoku-server`, `gotermoku.db`, and `/bin/` are
  build/runtime artifacts — do not commit them.
- `internal/client/engine.go` polls input with a 50ms timeout and runs the bot
  in a goroutine; keep the loop non-blocking when adding features.
