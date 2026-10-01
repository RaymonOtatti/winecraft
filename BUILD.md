# BUILD — Base game + 2-computer multiplayer test

The working checklist for `/loop`. **One item per iteration.** Each item:
1. write the failing test first (for logic: world, proto, server rules);
2. make it pass, then refactor;
3. run the item's **Verify** command and paste the result into the log below;
4. tick the box and commit locally on `go-rebuild` (leaf by leaf, never `git add -A`).

🛑 = **stop and ask Franco** before doing it: system changes, anything outward-facing, pushes.
Nothing gets pushed to any remote until Franco says so.

**Scope of BASE:** the engine, the shared multiplayer world, movement, building in a sandbox, a first harvest, and persistence, on a **DEV MAP**. The dev map is a placeholder that makes **no real-world claims**. The real valley comes from the Phase 2 data pipeline in `PLAN.md`; nothing here is presented as real data.

---

## B0 — Setup
- [x] **B0.1** Branch `go-rebuild`; `go mod init github.com/RaymonOtatti/winecraft`; `.gitignore` gets `web/*.wasm`, `bin/`, `data/sources/`, `*.db`. Commit `PLAN.md`, `BUILD.md`, `bench/`. The legacy JS/Python stays untouched.
  *Verify:* `git status` is clean on `go-rebuild`; `go vet ./...` runs.
- [x] **B0.2** `Makefile`: `test` (with `-race`), `wasm`, `run`, `share`, `snap` (screenshot).
  *Verify:* `make test` is green on an empty module.

## B1 — World core (`internal/world`, TDD)
- [x] **B1.1** Tile registry (id, name, walkable, layer, breakable, placeable, drop). Chunk = 32×32 with layers `ground`/`object`, stored as `[]uint16`. World chunks keyed by an integer pair, never strings. Get/Set across chunk borders, negative coordinates included.
  *Verify:* `go test ./internal/world/...` covers borders and negatives.
- [x] **B1.2** Movement rules: `CanStep(from, dir)` blocks on solid objects and water; **ledges are one-way** (hop down, not up, Pokémon style).
  *Verify:* table tests per direction and ledge case.
- [x] **B1.3** Deterministic DEV MAP, about 96×96: vine rows, a dirt road, a stream with a bridge, a small plaza, a fenced **sandbox build zone**, and a ledge band. Same seed → same bytes.
  *Verify:* determinism test (hash); the spawn is walkable; the sandbox can be reached from spawn (BFS test).

## B2 — Protocol (`internal/proto`, TDD)
- [x] **B2.1** Versioned binary messages: Hello{ver, name, joinCode, token}, Welcome{id, spawn, mapW, mapH}, Chunk{cx, cy, layers}, Move{dir, seq}, PlayerState{id, x, y, facing, name}, PlayerLeft{id}, Edit{x, y, layer, tile}, Inventory{items}, Ping{}, Error{code}. **One game message per WebSocket binary message** (the socket already frames it, so no length prefix): a 1-byte type, then fixed little-endian fields. A hard size cap, and every chunk message stays well under the 32 KiB default read limit.
  *Verify:* round-trip tests for every message, plus `go test -fuzz=FuzzDecode -fuzztime=30s` with no panics. The server faces the public internet, so this matters.

## B3 — Server (`cmd/server`)
- [x] **B3.1** HTTP: serve `web/` (index.html, wasm_exec.js, the wasm), `/healthz`, and WebSocket `/ws` behind a **join code**. One reader goroutine per connection; a 10 Hz world loop broadcasts state deltas. `SetReadLimit` explicitly on both ends. **Game-level heartbeat every ≤ 20 s** (the browser's WebSocket ping does nothing, and Cloudflare drops idle connections). Serve the wasm **gzip-compressed** (stdlib; it's ~14 MB raw). Trust `CF-Connecting-IP` only when the TCP peer is loopback, which is where cloudflared connects from.
  *Verify:* an httptest integration test where two clients connect and each sees the other's PlayerState; a wrong join code is rejected.
- [x] **B3.2** Authoritative movement: each step is validated against `world.CanStep` with at most 1 step per 150 ms; a rejected step snaps the client back.
  *Verify:* tests for a wall bump, a speed hack, and a ledge going up.
- [x] **B3.3** Edits: break and place only inside the sandbox zone, rate-limited, broadcast to everyone.
  *Verify:* tests for an edit outside the sandbox (rejected), a flood (rate-limited), and an edit reaching both clients.
- [x] **B3.4** Hardening: max players (8 for the test), message size cap, ping/idle timeout, origin check, graceful shutdown.
  *Verify:* tests for an oversized frame, a 9th player, and an idle drop.

## B4 — Client (`cmd/client`, Ebitengine; desktop for fast iteration, wasm for the browser)
- [x] **B4.1** Boot and render: placeholder 16 px tiles **drawn in code** into one atlas (no external art, no license questions), visible chunks only, a camera that follows the player, and a `-snap out.png` flag that saves frame 30 and exits.
  *Verify:* `make snap` → read the PNG and check that the dev map renders.
- [x] **B4.2** Grid movement with smooth interpolation, facing, a 2-frame walk; arrows/WASD plus a **touch D-pad** in the browser; local prediction with server reconciliation.
  *Verify:* a snapshot after scripted moves; a manual browser check.
- [x] **B4.3** Networking: `coder/websocket` in wasm and on desktop. Dial and read in **goroutines** (blocking inside a JS callback deadlocks). Hello/Welcome, other players drawn with name tags, reconnect when the connection drops.
  *Verify:* the desktop client and a browser tab against the local server see each other (snapshot from each).
- [ ] **B4.4** Building: face a tile; **A** places the selected hotbar tile, **B** breaks. A 5-slot hotbar.
  *Verify:* the edit appears on the second client (snapshot).
- [ ] **B4.5** HUD: online players, connection state, the join code screen (name + code).
  *Verify:* snapshot.

## B5 — Two-computer link (Cloudflare quick tunnel)
- [ ] 🛑 **B5.1** `brew install cloudflared`. Ask Franco first: this changes the system.
- [ ] **B5.2** `make share`: builds the wasm, picks a free port (checked against the full listening set), starts the server on `127.0.0.1` only, generates a random join code, starts `cloudflared tunnel --url`, prints **"Send Raymon: <url> code <code>"**, wraps everything in `caffeinate -i` so the Mac doesn't sleep, and tears it all down on Ctrl-C.
  *Verify:* the URL serves the game from outside (curl via the tunnel); WebSocket upgrade through the tunnel works; Ctrl-C leaves no processes behind.
- [ ] **B5.3** Self-test through the tunnel: two browser tabs using the public URL; both move, see each other, and see edits.
  *Verify:* server log + snapshots.
- [ ] 🛑 **B5.4** Live test with Raymon. Franco sends him the link; I watch the server log.

## B6 — First loop seed
- [ ] **B6.1** Server-side inventory. Harvesting a vine tile **does not destroy it**: the vine goes to a "harvested" state and regrows on a timer. Sandbox materials can be gathered and placed.
  *Verify:* tests (harvest, regrow, no double harvest); snapshot.
- [ ] **B6.2** Persistence behind a `Store` port: a file adapter (atomic write and rename) that saves edits, vine states, and per-token inventories and positions every 30 s and on shutdown. SQLite comes later as a second adapter.
  *Verify:* a test that restarts the server and checks edits and inventory survive; `kill -9` loses at most 30 s.

## ✅ BASE gate (all must hold)
- Two computers over the tunnel: both move, see each other, build in the sandbox, harvest, and edits sync.
- A server restart keeps edits and inventories.
- `go test -race ./...` green; fuzz 30 s clean; zero `go vet` findings.
- Native client frame ≤ 8 ms on the Air (measured); wasm size recorded (raw + brotli) for the PLAN Phase 1 gate.

---

## Log
<!-- one line per finished item: date · item · what was verified (command + result) · commit -->
- 2026-10-01 · B0.1 · branch `go-rebuild`, `go.mod` (module github.com/RaymonOtatti/winecraft, go 1.26.4), `.gitignore` extended, root `doc.go` · `go vet ./...` exit 0 · see commit
- 2026-10-01 · B0.2 · `Makefile` with test (-race), vet, fuzz, wasm, size, run, client, snap, share, clean · `make test` exit 0 · see commit
- 2026-10-01 · B1.1 · `internal/world`: 17-tile registry (vines not breakable), 32×32 chunks as flat `[2][1024]uint16`, `ChunkCoord` integer keys, shift/mask `ChunkOf` · `go test -race ./internal/world/...` 6/6 pass (borders, negatives, layers, no alloc on read) · see commit
- 2026-10-01 · B1.2 · `world.CanStep` (walls, water, vines, void block; `LedgeSouth` hop lands 2 tiles south, no climbing, no walking along, landing must be free) + `Dir` · `go test -race ./internal/world/...` 10/10 pass · see commit
- 2026-10-01 · B1.3 · `GenerateDevMap(seed)`: 96×96 placeholder (poplar border, ledge band y=30 with a road gap, two vineyard blocks, stream with 2 bridges, plaza spawn (47,47), fenced sandbox 24×13 with an east gate), `World.Digest()` · determinism (same seed same digest, different seed differs), spawn standable, sandbox + vine + ledge hop reachable by BFS over `CanStep`, no escape from bounds · 14/14 pass · see commit
- 2026-10-01 · B2.1 · `internal/proto`: 11 message types, one per WS message, 1-byte type + LE fields; bounded strings (UTF-8 checked), enums, tile ids, item counts; MaxMessage 8 KiB (a chunk is 4,105 B) · round trip for every type, 10 malformed-input rejections, oversized-field refusals; `make fuzz` 30 s = 10,020,060 execs, no crash, re-encode stable · see commit
- 2026-10-01 · B3.1 · `internal/game` (State: Join/Leave/dirty set, `CleanName`), `internal/server` (hub actor owns state; per-conn reader + writer goroutines; Hello handshake with constant-time join-code check; Welcome + chunks + existing players on join; 10 Hz dirty broadcast; PlayerLeft; 15 s game heartbeat; slow clients dropped; gzip wasm cache; `CF-Connecting-IP` trusted only from loopback), `cmd/server` (random wine-word join code, graceful SIGINT) · `go test -race -count=10 ./internal/server/...` pass (two players see each other, chunks, PlayerLeft, bad code/version/name rejected, Hello required, healthz, gzip wasm, IP trust); binary smoke test `/healthz` → `ok 0 online` · see commit
- 2026-10-01 · B3.2 · `game.State.Move`: shared `CanStep` + map bounds; token bucket (burst 3, 1 step / 150 ms, ledge hop costs 2); bumps turn in place; every processed move records seq + marks dirty so a mispredicting client snaps back; replayed/older seqs ignored. Hub routes `proto.Move` · 8 game tests (wall bump, ledge up, speed hack, sustained cap, hop cost, stale seq) + 2 network tests (move broadcast to both, 10 instant moves capped) · `go test -race -count=5 ./...` pass · see commit
- 2026-10-01 · B3.3 · `game.State.Edit`: sandbox only, target exactly 1 tile away, break only breakable (floors → dirt, never a hole), place only placeable on its own layer onto empty walkable ground with no player on it; every attempt costs an edit token (burst 5, 1 / 250 ms); shared `bucket` type now used by Move and Edit. Hub broadcasts `TileUpdate` or replies `Error` · 6 game tests (10 refusal cases, floors, other player, flood) + 3 network tests (edit reaches both, outside refused, flood rate-limited) · `go test -race -count=5 ./...` pass · see commit
- 2026-10-01 · B3.4 · `internal/rate` (shared token bucket; game moved onto it), `server/limits.go` per-IP gate (≤ 4 open sockets → HTTP 429; 5 wrong join codes → locked out, 1 forgiven / 30 s, map bounded); Config: idle drop 45 s, Hello timeout 10 s, client message allowance 100 burst / 50 per s, oversized frames close with 1009, origin check (Accept default), shutdown closes clients concurrently with 1001 · 9 new tests (9th player, oversized frame, idle drop vs heartbeat, silent pre-Hello, flood, per-IP cap, join-code lockout, cross-origin, going-away) · `go test -race -count=10 ./...` pass · see commit
- 2026-10-01 · B4.1 · Ebitengine v2.10.4; `internal/client/art` draws a 17-tile 16 px atlas + 4-dir × 2-frame player sheet in code (no external assets); `internal/client` camera (centre, visible tile range with floor division, integer pixel scale) + `Game` drawing only visible tiles from one atlas, `-snap`/`-at` flags; `web/index.html` loader · art 5/5 + camera 4/4 tests; `make snap` → `snap/client.png` (plaza, road, player), `snap/vines.png` (vine rows, ledge band), `snap/sandbox.png` (stream, bridge, fenced sandbox, rock) all render; `make size`: wasm 12,890,950 B raw / 3,085,468 gzip / 2,190,640 brotli (budget 4 MB brotli) · see commit
- 2026-10-01 · B4.2 · `client.Walker`: tile steps of 200 ms (above the server's 150 ms cap) with interpolation, tap-to-turn (90 ms, local only: the server treats every Move as a step), bump sent once, ledge hop with an 8 px arc over 400 ms, 2-frame walk, pending moves + `Reconcile` (drop acked, replay the rest, snap if different); `Input` (arrows/WASD, last pressed wins; touch D-pad hit-test) and `-walk`/`-touch` flags · 9 walker/D-pad tests; `snap/walk.png` after "N19,W3,S2" shows the player below the ledge (hop) with the translucent D-pad; wasm still builds · manual browser check deferred to B5.3 (needs a browser) · see commit
- 2026-10-01 · B4.3 · `client.Session` (applies Welcome/Chunk/PlayerState/PlayerLeft/TileUpdate/Error; own state → `Walker.Reconcile`; remotes slide one step or snap on jumps > 2; reconnect Welcome resets) and `client.Net` (coder/websocket; dial/read/write on goroutines, never in a JS callback; non-blocking Recv/Send; 15 s Ping; drains stale moves on reconnect; backoff 1→10 s; stops on wrong code/version/name); name tags (debug font, accents folded); browser reads the page host and `?name=&code=`; `-server/-name/-code/-stay` flags · 7 session + 3 live-server net tests (two clients see each other and a move, reconnect after server idle-drop, wrong code → Rejected) `-race -count=2` pass; live run: `bin/server` + two native clients → `snap/franco.png` shows "Raymon" 5 tiles west, `snap/raymon.png` shows "Franco" 5 tiles east; wasm builds · real-browser check deferred to B5.3 · see commit
