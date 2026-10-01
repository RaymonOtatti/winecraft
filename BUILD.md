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
- [ ] **B0.2** `Makefile`: `test` (with `-race`), `wasm`, `run`, `share`, `snap` (screenshot).
  *Verify:* `make test` is green on an empty module.

## B1 — World core (`internal/world`, TDD)
- [ ] **B1.1** Tile registry (id, name, walkable, layer, breakable, placeable, drop). Chunk = 32×32 with layers `ground`/`object`, stored as `[]uint16`. World chunks keyed by an integer pair, never strings. Get/Set across chunk borders, negative coordinates included.
  *Verify:* `go test ./internal/world/...` covers borders and negatives.
- [ ] **B1.2** Movement rules: `CanStep(from, dir)` blocks on solid objects and water; **ledges are one-way** (hop down, not up, Pokémon style).
  *Verify:* table tests per direction and ledge case.
- [ ] **B1.3** Deterministic DEV MAP, about 96×96: vine rows, a dirt road, a stream with a bridge, a small plaza, a fenced **sandbox build zone**, and a ledge band. Same seed → same bytes.
  *Verify:* determinism test (hash); the spawn is walkable; the sandbox can be reached from spawn (BFS test).

## B2 — Protocol (`internal/proto`, TDD)
- [ ] **B2.1** Versioned binary messages: Hello{ver, name, joinCode, token}, Welcome{id, spawn, mapW, mapH}, Chunk{cx, cy, layers}, Move{dir, seq}, PlayerState{id, x, y, facing, name}, PlayerLeft{id}, Edit{x, y, layer, tile}, Inventory{items}, Error{code}. Length-prefixed, with a hard size cap.
  *Verify:* round-trip tests for every message, plus `go test -fuzz=FuzzDecode -fuzztime=30s` with no panics. The server faces the public internet, so this matters.

## B3 — Server (`cmd/server`)
- [ ] **B3.1** HTTP: serve `web/` (index.html, wasm_exec.js, the wasm), `/healthz`, and WebSocket `/ws` behind a **join code**. One reader goroutine per connection; a 10 Hz world loop broadcasts state deltas.
  *Verify:* an httptest integration test where two clients connect and each sees the other's PlayerState; a wrong join code is rejected.
- [ ] **B3.2** Authoritative movement: each step is validated against `world.CanStep` with at most 1 step per 150 ms; a rejected step snaps the client back.
  *Verify:* tests for a wall bump, a speed hack, and a ledge going up.
- [ ] **B3.3** Edits: break and place only inside the sandbox zone, rate-limited, broadcast to everyone.
  *Verify:* tests for an edit outside the sandbox (rejected), a flood (rate-limited), and an edit reaching both clients.
- [ ] **B3.4** Hardening: max players (8 for the test), message size cap, ping/idle timeout, origin check, graceful shutdown.
  *Verify:* tests for an oversized frame, a 9th player, and an idle drop.

## B4 — Client (`cmd/client`, Ebitengine; desktop for fast iteration, wasm for the browser)
- [ ] **B4.1** Boot and render: placeholder 16 px tiles **drawn in code** into one atlas (no external art, no license questions), visible chunks only, a camera that follows the player, and a `-snap out.png` flag that saves frame 30 and exits.
  *Verify:* `make snap` → read the PNG and check that the dev map renders.
- [ ] **B4.2** Grid movement with smooth interpolation, facing, a 2-frame walk; arrows/WASD plus a **touch D-pad** in the browser; local prediction with server reconciliation.
  *Verify:* a snapshot after scripted moves; a manual browser check.
- [ ] **B4.3** Networking: WebSocket in wasm and on desktop, Hello/Welcome, other players drawn with name tags, reconnect when the connection drops.
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
