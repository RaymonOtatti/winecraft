# WineCraft — Rebuild Plan

**Date:** 2026-10-01 · **Status:** plan only, no code yet · **Source inspected:** `RaymonOtatti/winecraft` @ `bbf50c4` (single commit)

**What we're building:** a 2D top-down pixel-art game set in the **real Valle de Uco**. The world plays like Pokémon (towns, routes, NPCs, a collection dex, medals, turn-based duels). Building and crafting work like Minecraft. Winemaking is realistic, step by step, and runs on real numbers. The economy uses real prices. It's one **shared multiplayer world**, written in **Go end-to-end** and hosted publicly on your TrueNAS.

Each phase has a PASS / NOT-PASS gate, and nothing moves to the next phase until its gate passes.

---

## 0. TL;DR

| | Today (prototype) | Target |
|---|---|---|
| View | First-person 3D voxel (three.js r128, 2021) | 2D top-down pixel art, grid movement, Pokémon-style |
| World | 73 × 73 blocks, hand-placed, invented layout | The real valley built from elevation + OpenStreetMap data, tens of km across, streamed in chunks |
| Bodegas | 4, coordinates invented | Every bodega in the official registry inside the valley; a few "hero" bodegas built in detail |
| Winemaking | 2 grapes → mosto → wait 7.5 s → "100-point" bottle | About 15 real stages with real parameters (°Brix, SO₂, temperature, days, vessels), plus lab analysis and faults |
| Data | Partly fabricated (see Appendix B) | Every user-visible fact has a source record; CI blocks unsourced strings |
| Economy | None | Real grape, bulk-wine, bottle, land and equipment prices, dated and in USD |
| Server | Python `http.server` on 127.0.0.1, one global save file | Go server, authoritative, WebSockets, SQLite, per-player state |
| Language | JS client + Python | **Go** client (Ebitengine → WebAssembly) + Go server + Go data pipeline |
| Hosting | localhost | `winecraft.francomichetti.com` → Cloudflare Tunnel → Traefik → container on TrueNAS |

**Can we build it in Go, fast and optimized? Yes.** Choosing 2D is what makes Go the right call for the client too. Details are in §10.1, including what actually makes it fast; that part is not the language.

---

## 1. What's in the repo today

3,244 lines: `server.py` (92), `public/js/*` (2,497), `index.html`, `styles.css`, vendored `three.min.js` r128 + `PointerLockControls.js`, and a committed `world_save.json`. No license file, no tests, one commit.

### 1.1 Performance — measured, not guessed

`bench/legacy-measure.js` replays the real `world.js` generation headless with a stubbed THREE and counts what the GPU gets asked to do. Re-run it with `node bench/legacy-measure.js`. Three runs on this Mac, Node 22:

| Metric | Value | Why it matters |
|---|---|---|
| **Draw calls per frame** | **19,946** | Every visible face is pushed as its own geometry group (`world.js` `bg.groups.push` per face), and three.js issues **one draw call per group** on multi-material meshes. A browser budget is about 1–2k. This is the frame-rate killer. |
| Triangles | 39,892 | About 2 triangles per draw call. The GPU idles while the CPU drowns in call overhead. |
| World generation + meshing | 260–370 ms headless | Runs on the main thread at page load; texture generation and GPU upload come on top. |
| One block edit | 4–12 ms, rebuilds 890 groups | One stutter frame per click, and it gets worse as the world grows. |
| Block lookup | String-keyed `Map` (`"x,y,z"`) is **35–40× slower** than a flat typed array | Every physics step, raycast and mesh face goes through `getBlock`. In a profile, `getBlock` was the top self-time function. |
| World size | 73 × 73 × 24 | Nothing streams. The whole map is built at once, so it can't get bigger. |

Smaller issues: `shadowMap.enabled` with no shadow casters (pure cost); `updateRegionHUD` rewrites `innerHTML` every frame; each particle is its own `Mesh`; a save POSTs the entire edit history every time, so it grows without bound.

### 1.2 Gameplay and correctness bugs

- **Harvesting destroys the vineyard.** Left-click deletes the vine block, so vineyards disappear forever.
- **Fermenting vats are never saved.** `activeVats` lives only in memory; reload and the wine is gone.
- **Inventory can desync.** Pressing and bottling don't trigger a save; only block edits do.
- **You can't choose what to press.** The press takes the first grape type with ≥ 2 bunches, in a fixed order.
- **Region detection is duplicated and contradicts itself.** `main.js` splits PerSe and La Estocada at `x < 3`, `winemaking.js` at `x < 0`.
- Vines are solid full cubes you collide with, two blocks tall. Water renders all six faces, including hidden ones.
- At terminal velocity (30 m/s × 0.1 s delta clamp) the player moves 3 blocks per step, so they can fall through 1-block floors.
- The server banner still says "Argentina y Borgoña", left over from an earlier theme.

### 1.3 Why the current server must not go public

- **One global save file for everyone.** Any visitor overwrites every other visitor's world.
- `POST /api/save_world` has **no body-size limit**, so anyone can fill the disk, and no auth or rate limit.
- It's single-threaded `http.server`, which is meant for development.
- `innerHTML` templating is harmless while all strings are constants. The moment player names or chat exist, it becomes XSS. A canvas-rendered Go client removes that surface entirely.

### 1.4 Data problems ("real data only")

The four bodegas are **real**, and most headline facts check out. The problems are in the details (full audit in **Appendix B**):

- **Scores on a vintage that hasn't been rated.** "100 Puntos" is shown on a **2026** bottle. The real 100s belong to specific vintages: Finca Piedra Infinita 2016, Gravascal 2018, Supercal 2019 (Wine Advocate) and PerSe La Craie 2018. A 2026 Uco wine hasn't even finished aging.
- **Invented tasting notes labeled "Oficial".** The "Ficha de Degustación Oficial" text is made up and attributed to named real winemakers.
- **"Gualtallary" is presented as an appellation, but it isn't an official IG** (a third-party trademark blocks it). It's a real place; the game can name it, but not label it an IG.
- **Wrong roles and names.** Manolo Pelegrina is Cru de Montaña's vineyard manager, not a co-owner. PerSe's "Uní" is really "Uni del Bonesant".
- **A sensitive location.** PerSe's vineyard is leased from the Monasterio del Cristo Orante, which the Archdiocese of Mendoza closed in January 2019 after sexual-abuse complaints against two monks. The game must not portray it as an active monastery or as scenery.
- **Invented coordinates.** All four bodegas are placed on a made-up grid.

### 1.5 What survives the rebuild

Honestly, about 10%: the theme, the list of tile and block concepts, the bodega-by-bodega structure, and whatever content passes the Appendix B audit. The engine, physics, renderer and server all get replaced. The new direction (2D + Go) is a rewrite, not a refactor, and that's fine: the prototype did its job of proving the idea.

---

## 2. The core loop

```
 EXPLORE (Pokémon world) ──► COLLECT (Ampelodex) ──► HARVEST + CRAFT (Minecraft)
        ▲                                                       │
        │                                                       ▼
 EXPAND your bodega ◄── SELL at real prices ◄── DUEL / COMPETE ◄── MAKE WINE (realistic)
 (build on your parcel)    (real economy)       (medals)         (your wines = your party)
```

The idea that ties it together: **the wines you make are your "party"**. Each bottle's stats come from your real process choices (harvest °Brix, maceration days, vessel, months of aging). You take up to six bottles into duels. Better winemaking makes stronger wines, which win medals, which open zones and markets, which earn money for the next building. Crafting feeds battling feeds the economy, the same way catching feeds battling in Pokémon.

---

## 3. The "real data" contract

This is the hardest requirement, so it's enforced by tooling rather than by good intentions.

1. **A data registry.** `data/registry/*.yaml` holds every fact as a record with `id`, `value`, `source_url`, `retrieved_at`, `license` and `verified_by`. Raw downloads go in `data/sources/` with a checksummed manifest.
2. **No string without a source.** Game text refers to registry ids, not literals. `cmd/datacheck` runs in CI and fails the build if any user-visible fact lacks a source record. Same rule as Gungnir: no finding without proof.
3. **What's real and what's game, stated openly** on an in-game "Fuentes" screen:
   - **Real:** geography, elevation, rivers, roads, towns; bodegas and their public facts; grape varieties and their real planted hectares; native species; winemaking parameters; market prices (dated).
   - **Game:** your character, your bodega and parcel, NPC characters, time compression, duel scoring. Duel scoring is *derived from* cited principles, and the rules doc says how.
4. **Real people are never speaking NPCs** unless their bodega agrees in writing. Without that consent a bodega is a place with a factual card, and its staff are generic roles (enóloga, viñatero, sommelier).
5. **Some things can't be real, and we say so.** Private bodegas don't publish their earnings, so "bodega earnings" in the game come from real *market prices and cost studies* applied to *your* production (§8). We never invent a real company's revenue.
6. **Licenses are respected.** OpenStreetMap is ODbL: credit "© OpenStreetMap contributors", and publish the derived world data under ODbL. The DEM keeps its own attribution. Species records are CC0/CC-BY only.

---

## 4. The world: Pokémon-style, built from the real valley

### 4.1 Scale and projection

- **Projection:** UTM zone 19S (EPSG:32719). Tile `x` = easting, tile `y` = southing, so north is up as on any map.
- **Overworld: 1 tile = 5 m (recommended, confirmed in Phase 0).** At this scale a bodega building is 8–20 tiles across, a town spans hundreds of tiles, and the valley floor is thousands of tiles in each direction. That's big, and you cross it by bike or truck in tens of minutes, not hours. Distances and directions stay true, so the map is the real map.
- **Interiors and finca grounds: 1 tile ≈ 1 m**, as separate maps entered through doors, the way Pokémon buildings work. This is where tanks, barrels, presses and the sorting table sit at believable sizes.
- **Height:** Pokémon-style terrain bands (ledges, cliffs, stairs) come from the elevation model. One band covers a set elevation step (e.g., 25 m). The Cordón del Plata shows as an impassable mountain wall with real peaks named on the map.

### 4.2 Generation pipeline (`cmd/worldgen`, offline, deterministic)

```
Elevation model (Copernicus GLO-30)   ─┐
OpenStreetMap extract (.pbf)           ─┼─► reproject ─► rasterize to 5 m tiles ─► classify ─► bake chunks
INV vineyard + bodega registry         ─┤                                                 (versioned, checksummed,
Species occurrences (GBIF, CC0/CC-BY)  ─┘                                                  provenance per feature)
```

- **Terrain:** elevation sets height bands and ledges. Slope and land cover set the ground tile: monte (jarilla, chañar), irrigated farmland, riverbed boulders, sand.
- **Water:** rivers, streams and **irrigation canals (acequias)** from OSM waterways.
- **Roads become routes:** RN 40 and the provincial routes, plus dirt roads from OSM highways. Pokémon "Routes" are the real roads, numbered by their real numbers.
- **Vineyards** from OSM `landuse=vineyard` polygons. Rows run along each polygon's long axis; variety comes from INV department statistics where the parcel's variety isn't known.
- **Poplar windbreaks (álamos)** along field edges. They define the look of Uco and drive the Pokémon-style "tree walls" that shape routes.
- **Towns** (Tupungato, Tunuyán, San Carlos, La Consulta, Vista Flores and others) from OSM places and building footprints, rendered as Pokémon-style hub towns. Each has a market, a lab and a supply shop.
- **Bodegas** at their sourced coordinates, as landmark buildings with doors to interior maps.

### 4.3 Zones

The valley's official **geographical indications (IGs)** become the game's regions, each with its own soil tiles, music, species list, NPC quests and medal. INV's list for Valle de Uco:

| IG | Department | Resolution |
|---|---|---|
| Valle de Uco (regional) | San Carlos, Tunuyán, Tupungato | C.37/2002 |
| Tupungato – Valle de Tupungato | Tupungato | C.32/2002, C.20/2006 |
| Tunuyán | Tunuyán | C.32/2002 |
| San Carlos | San Carlos | C.32/2002 |
| Vista Flores | Tunuyán | C.11/2012 |
| Paraje Altamira | San Carlos | C.44/2013 |
| La Consulta | San Carlos | C.19/2014 |
| Los Chacayes | Tunuyán | RESOL-2017-249 |
| Pampa El Cepillo | San Carlos | RESOL-2019-1 |
| San Pablo | Tunuyán | RESOL-2019-10 |
| Cordón El Cepillo | San Carlos | Res. 7/2022 |
| El Peral | Tupungato | Res. 1/2022 |

**Gualtallary is not an IG.** A third-party trademark blocks the name, so it appears as a named area inside Tupungato, never labeled "IG".

IG boundaries are published only as vertex lists in the official resolutions (e.g., Cordón El Cepillo has 28 lat/lon vertices). No shapefile exists, so Phase 0 digitizes them from the resolutions, with the resolution as the source record.

### 4.4 Size and streaming

- **Chunk:** 32 × 32 tiles, with layers ground / overlay / object / collision / height, packed `uint16` per layer and compressed with `compress/flate` from the standard library.
- The whole valley's *terrain* is generated from the pipeline on day one, which is cheap. *Content density* (NPCs, quests, interiors) is built **zone by zone**, so the world is big from the first release and fills in over time.
- The client streams chunks around the player. The server keeps hot chunks in memory and the rest on disk.

### 4.5 Travel

On foot (about 4 tiles/s), bike, a rural "micro" bus between real towns on a timetable, and a pickup truck once you own a bodega. Fast travel opens to bodegas and towns you've already visited.

---

## 5. Crafting and building (the Minecraft half)

- **Gather:** stone from riverbeds, poplar wood, clay and sand, lime for concrete, river boulders (the Paraje Altamira look).
- **Craft:** recipes at a workbench, like Minecraft. Tools (pruning shears, harvest bins, refractometer), materials (concrete, planks, bricks), and winery equipment (sorting table, destemmer, basket press, concrete tank, barrel rack).
- **Build:** place and break tiles and objects on **your own parcel**. Walls, roofs, floors, cellars as interior maps, and vineyard rows you plant from cuttings.
- **The real world is read-only.** Nobody can bulldoze a real bodega or reroute a river. Players build on parcels they buy with in-game money at real per-hectare prices (§8). This one rule covers anti-griefing in the shared world and the "real data" rule at the same time.

---

## 6. Winemaking: step by step and realistic

Every stage is a **station** (an object you craft and place). It has a UI panel, takes real inputs, and writes an entry in your **Cuaderno de Bodega** (cellar log). The simulation lives in a shared Go package, `sim/`: pure and deterministic, built test-first against cited reference numbers, run authoritatively on the server.

*Stage list and parameters: see §6.1, filled from the fact-check sources.*

### 6.1 Stages (red wine; whites and rosé reuse most of them in a different order)

Every number below is a **registry record** with its source (ids refer to Appendix C). The winery figures are the published tech-sheet data of real Uco wines, so a player who copies a real wine's process can find that in the cellar log.

| # | Stage | Station | What you decide | Real reference |
|---|---|---|---|---|
| 1 | Ripeness sampling | refractometer (hand tool) | which parcel, which day; read °Brix | Gualtallary (Catena, Adrianna): harvest 15 Mar–26 Apr, °Brix never above 24.5 [W1]. Altamira (Zuccardi): Malbec picked 31 Mar–13 Apr in 2012, late Mar–29 Apr in 2014 [Z1] |
| 2 | Hand harvest | harvest bins | when to pick; bunch selection in the row | Zuccardi: hand harvest with bunch selection [Z2] |
| 3 | Sorting | sorting table | how strict (% rejected) | Double selection, then gravity filling (Altos Las Hormigas) [H1] |
| 4 | Destemming / whole cluster | destemmer | 0–100% whole cluster | Zuccardi Concreto 100% whole cluster [Z3]; Supercal 2019 destemmed and crushed [Z4]; Altos Las Hormigas 30% stems [H1]; Estocada Cabernet Franc 100% stems [E1] |
| 5 | SO₂ at crush | dosing | mg/L (or none) | AWRI small-lot method: 50 mg/L at crush [A1]. OIV maximum total SO₂ for reds: 150 mg/L when sugar ≤ 4 g/L [O1]. Some PerSe wines: no sulfites until bottling [P1] |
| 6 | Cold soak | tank with cooling | days at 5–10 °C, or skip | AWRI: 5–10 °C, from 5–10 hours up to 10 days [A2] |
| 7 | Alcoholic fermentation | concrete / steel / open vat | native or inoculated yeast; temperature | Red range 20–30 °C [U1]. Altos Las Hormigas: native yeast, 26 °C, 25 days [H1]. Zuccardi and PerSe: native yeast [Z2][P1] |
| 8 | Cap management | in the vat | pump-over or punch-down, times per day, days on skins | Altos Las Hormigas: no pumping, punch-down twice a day [H1]. La Craie 2013: two punch-downs a day, 35 days on skins [P2]; current La Craie: light foot treading, 45 days [P3]. AWRI: plunge 3–4 times a day [A1] |
| 9 | Pressing | basket or pneumatic press | when; free-run vs. press fraction | AWRI: press when residual sugar < 2 g/L [A1] |
| 10 | Malolactic | any vessel | temperature; when it's done | 18–22 °C, pH 3.2–3.5, free SO₂ < 5 mg/L; done at malic < 30–50 mg/L [U2] |
| 11 | Racking | tank | settle, then rack off the solids | AWRI: settle 48 h at 4 °C, then rack [A1] |
| 12 | Aging (élevage) | concrete, used 500–600 L oak, 3,500 L foudre, barrique | vessel mix and months | Finca Piedra Infinita 2020/2023: 100% concrete; 2016: part in used 500 L French oak [Z2][Z5]. Supercal 2019: 50% used 500 L oak [Z4]. Altos Las Hormigas: 20 months, half untoasted 3,500 L foudres, half concrete [H1]. La Craie: 12 months neutral French oak [P3]. Cru de San Pablo: 18 months 600 L oak, then 12 months concrete [C1] |
| 13 | Blending | lab bench | lots and % | Your choice; the cellar log records it |
| 14 | Fining / filtration | filter | none, coarse or gentle | La Craie unfiltered [P3]; Altos Las Hormigas coarse [H1]; Cru de San Pablo gentle [C1] |
| 15 | Bottling | bottling line | final SO₂ adjustment, closure, label | OIV limits apply at the lab check [O1] |
| 16 | Bottle aging | cellar | months before release | Altos Las Hormigas 6 months [H1]; Cru de San Pablo 24 months [C1] |

**Still needs a source (Phase 5 blocker):** the sugar-to-alcohol conversion factor and its fermentation curve; a typical °Baumé for Uco Malbec (the fact-check found none beyond the 24.5 °Brix ceiling); INV's sugar-by-variety data (not found in the harvest yearbook). These must be sourced before `sim/` uses them.

### 6.2 Vineyard year (the calendar)

Real Southern Hemisphere phenology drives a **shared world calendar**: winter pruning, budbreak, flowering, veraison, harvest. Each zone's dates shift with its real climate. Growing degree days come from historical weather for that zone's coordinates, so high zones like Gualtallary really do ripen later than lower ones. Weather events (frost, hail, Zonda wind) appear at their real seasonal frequency.

**Time compression (recommended default):** 1 game year = 4 real weeks. The harvest window then lasts several real days, so players in any timezone catch it. A ten-day fermentation takes about a real day, and a year in barrel takes a real month of play. This is tuned in playtests (Phase 5 gate).

### 6.3 Lab and faults

- **Lab:** measure °Brix/density, alcohol, pH, titratable acidity, volatile acidity, free and total SO₂, residual sugar, malic acid.
- **Faults happen for real reasons:** fermentation too hot → stuck fermentation; too little SO₂ plus oxygen → oxidation and volatile acidity; dirty barrels → Brett. Each fault has a cause you can see in the cellar log and a real fix.
- **Legal limits:** where Argentine regulation (INV) sets a limit (e.g., maximum volatile acidity or SO₂), the lab flags it, and the market refuses wine that exceeds it.

---

## 7. The Pokémon half

### 7.1 Towns, routes and NPCs

Real towns are hubs: market, lab, supply shop, inn (respawn), notice board for quests. Routes are real roads. NPCs are archetypes (viñatero, cosechadora, enólogo, sommelier, cooper, mule driver, the old acequia keeper). Their dialog teaches real practice and points to the Fuentes screen. Quests are grounded in real tasks: fix an acequia, sample a parcel's ripeness, deliver bins before the heat of the day.

### 7.2 Ampelodex (the collection)

- **Grape varieties.** Encounter rarity is **proportional to the real planted hectares** in INV statistics (the 2025 CSV has hectares by district and variety: in Uco, Malbec is 16,592.7 of 29,934.5 ha, while Nebbiolo has 1.8 ha), so Malbec is everywhere and rare varieties are rare for real. You "collect" a variety by sampling a vine in the field (no capture mechanics, see §13). Collected varieties unlock as cuttings you can plant.
- **Soils** of each zone (calcareous, alluvial boulders, sandy), with their sourced descriptions.
- **Native species** from **real GBIF occurrence records** for the valley, CC0/CC-BY only. You photograph them in the field (cóndor, guanaco, zorro gris, jarilla, chañar…), so the "tall grass" encounters are real wildlife, observed and not fought.
- **Bodegas and IGs** you've visited, each a factual card with sources.

### 7.3 Medals and progression

Each zone has a **medal**, earned by beating that zone's challenge: a duel series against that zone's sommelier/enólogo NPCs, plus a winemaking task suited to its terroir (e.g., a wine from your own grapes that passes the lab within spec). Medals unlock routes, markets, equipment tiers and parcel purchases in new zones. The final event is a **regional competition** (in-game, fictional name) against the strongest NPCs and other players.

### 7.4 Duels (turn-based)

Your party is up to six of **your own bottles**. Two duel modes, both grounded in cited principles:

1. **Maridaje duel.** The opponent serves real Mendoza dishes (empanadas, chivito, asado, trout…), each with properties (fat, salt, acidity, protein, spice). You answer with a wine. Scoring follows documented food–wine interactions: acidity against fat, tannin against protein, alcohol against spice, sweetness against heat. The "type chart" is a cited table, not invented magic.
2. **Blind tasting duel.** The opponent pours a wine. You spend turns on moves (Mirar, Oler, Probar, Analizar) to narrow down variety, zone and style, and score by deduction. Opponent wines are your previous bottles or real wines described **only with the winery's published tech-sheet data**, cited.

PvP (player against player) uses the same rules, so a deterministic shared Go `duel/` package runs on the server and is predicted on the client.

---

## 8. Economy: real prices, your earnings

**Confirmed 2026-10-01:** in-game money valued at **real market prices**, with real bodegas in the world. Your bodega earns by selling at real prices and pays real costs. **No real-money purchases**; real money would bring payments, consumer law and alcohol-sales law with it.

- **Currency:** USD stored as integer cents, with an ARS display using a dated official exchange rate. Argentine inflation makes ARS prices go stale within months, so every price record carries its date.
- **Prices from the registry, with dates:** grapes per kg by variety and zone; bulk wine per liter; bottle retail by tier; vineyard land per hectare; equipment (barrels, tanks, press, destemmer) and dry goods (bottle, cork, label); harvest labor per bin; vineyard cost per hectare per year from official cost studies. Sources are in Appendix C.
- **Your earnings = your volume × real price for your tier and quality − real costs.** Quality comes from the lab and duel record, never from fake critic scores.
- **Markets:** a town market (NPC buyers at real prices, with daily noise inside a sourced range), bulk-wine sales, and later player-to-player trading.
- **The ledger is correct by construction:** a double-entry append-only ledger in SQLite, with invariants checked by tests (money is never created or destroyed outside defined sources and sinks). The server is the only authority; the client never computes a balance.
- **Real bodegas:** cards show public facts (hectares, founding year, IG, wines, published production if it exists). Never invented finances.
- **Every price record has a `kind`:** `paid`, `offered`, `asked`, `list` or `sale`. The research shows these differ a lot (in 2026 producers asked ≥ 300 ARS/kg while wineries offered about 210), so the game never mixes them silently.

### 8.1 What the real numbers say (researched 2026-10-01, sources in C.4)

**Growing and selling grapes in Valle de Uco loses money right now.** For the 2025/26 season the rural federation and the Uco rural society put the cost at **7.05 M ARS per hectare** against **4.4 M of revenue**: a **loss of 2.65 M ARS/ha**. Wineries offered about **220–300 ARS/kg** for fine grapes in 2026, against the roughly 500 producers say they need. Bulk red wine was about **350 ARS/L** in July 2026.

The value is in the bottle: Zuccardi Serie A Malbec lists at about USD 12, Zuccardi Q at about USD 18, Finca Piedra Infinita 2020 at about USD 152, and PerSe La Craie at USD 250 from the winery.

**This makes a good game, honestly told.** Selling raw grapes is the realistic losing move. Players earn by adding value: making wine, raising quality, bottling, building a name through medals, and selling direct. The economy shows the real squeeze instead of hiding it.

**Anchor numbers for the first price table** (all dated; full list in C.4):

| Item | Value | Kind | Date |
|---|---|---|---|
| Fine grapes, Uco | 220 ARS/kg | offered | 2026-02 |
| Malbec, Tupungato | 600–680 ARS/kg | paid | 2025 harvest |
| Bulk basic red | 350 ARS/L | list (Bolsa) | 2026-07 |
| Vineyard cost, Uco | 7,050,000 ARS/ha/yr | study | 2025/26 |
| Vineyard worker, entry level (CCT 154/91) | 17,252.84 ARS/day | agreement | 2026-03 |
| Harvest pay, fine grapes | 600–700 ARS/bin | paid | 2025 |
| Planted vineyard, Gualtallary | USD 70,000–90,000/ha | bank estimate | **2019 (stale)** |
| New French oak barrique, 225 L | USD 1,377 | list | 2026-10 |
| Official exchange rate (BCRA) | 1,517 ARS/USD | official | 2026-09-30 |

**Gaps the registry must fill before Phase 7:** official grape prices by variety and department (the Bolsa de Comercio de Mendoza publishes them, but its site timed out); the weight of a harvest bin in kg; Uco land prices after 2019; prices for a concrete tank and a pneumatic press; an itemized high-altitude cost model (the IDR site doesn't resolve).

---

## 9. Shared world (multiplayer)

- **One world**, many players. You see others walk, wave and emote. Duels and trades happen by walking up to someone, as in Pokémon.
- **Interest management:** each player subscribes to the chunks around them and gets position updates only for players in that area. Server tick is 10 Hz for movement; the simulation steps on a slower clock.
- **The server is authoritative** for movement (speed and collision checks), inventory, the simulation, the ledger, duels and edits. The client predicts movement only.
- **Identity:** anonymous at launch, with a device token and a chosen name (filtered). An optional account (email link) comes later so progress moves between devices.
- **Chat:** preset phrases and emotes at launch, as in Pokémon. Free text means a moderation workload; add it only with a moderation plan.
- **Griefing:** the real world is read-only (§5); parcels have owners and per-player edit permissions; there are rate limits on every action.

---

## 10. Architecture (Go)

### 10.1 Why Go works here, and what actually makes it fast

**Language isn't the speedup.** The prototype is slow because of 20k draw calls, string-keyed lookups and main-thread generation. Rewriting those same algorithms in Go would be just as slow. What fixes them:

- **2D tiles batch into a handful of draw calls.** Ebitengine automatically batches draws that share a texture atlas, so a full screen of tiles is a few calls, not 20,000.
- **Flat arrays** (`[]uint16` per chunk layer) instead of string maps.
- **Offline generation**: the world is baked by `cmd/worldgen`, not built while the page loads.
- **Zero allocations per frame**: pooled buffers, so Go's garbage collector stays quiet.

**What Go really buys you:**

- **One language, one codebase.** Client, server, load-test bots and the data pipeline share `world/`, `proto/`, `sim/`, `duel/` and `econ/` types. Client prediction and server truth run the same code, so they can't drift.
- **A cheap concurrent server.** Goroutines per connection, a tight tick loop, one static binary in a small container.
- **Load testing for free.** `cmd/loadbot` reuses the real client's protocol code to simulate hundreds of players.

**What Go costs**, and why Phase 1 is a spike with a hard gate: a larger `.wasm` download than a hand-written JS game, a single-threaded runtime in the browser, and garbage-collector pauses if per-frame allocations creep in. **If the spike fails its gate**, the fallback is a TypeScript client (PixiJS) speaking the same Go protocol. The server, simulation and pipeline stay Go either way.

**Checked 2026-10-01** (sources in Appendix C.5):

- **Engine:** Ebitengine v2.10.4 (2026-09-25) needs Go 1.25+. We build with Go 1.26.4; the latest Go is 1.27.1. The web is an officially documented target and needs WebGL2. Touch input is built in, and iOS audio unlocks on the first tap.
- **Download size:** a minimal Ebitengine tile program is **14.6 MB raw, 3.5 MB gzip, 2.5 MB brotli** (measured with Go 1.26.4). Our budget is about 4 MB brotli for the first playable, which is why the server must send the wasm compressed. TinyGo is not an option: Ebitengine closed that as "not planned".
- **Networking:** `github.com/coder/websocket` compiles to wasm and wraps the browser WebSocket. A shipped Ebitengine game, bgammon, uses exactly this pair. Three rules for our code: (1) the read limit defaults to 32 KiB on both ends, so call `SetReadLimit` and keep chunks small; (2) never block inside a JS callback: dial and read in goroutines; (3) the browser can't set headers, so the join code and token travel in the first message, and the browser's `Ping` does nothing, so heartbeats are game messages.
- **Unknowns the Phase 1 gate must measure:** frame rate on a current mid-range phone (the only public number is years old), startup time, and memory staying flat for 30 minutes (an old report traced a leak to this WebSocket library's previous version).

### 10.2 Repository layout

```
winecraft/
  cmd/
    client/      Ebitengine game → web/winecraft.wasm
    server/      HTTP + WebSocket server; embeds web/ via embed.FS
    worldgen/    offline: elevation + OSM + INV + GBIF → baked chunks (+ provenance)
    datacheck/   CI: every user-visible fact has a source record
    loadbot/     headless bot clients for load tests
  internal/
    world/       tiles, chunks, layers, collision, projection (shared)
    proto/       binary protocol, versioned (shared)
    sim/         winemaking simulation, pure + deterministic (shared)
    duel/        turn-based duel rules (shared)
    econ/        prices, market, ledger (server)
    dex/         collection (shared types)
    registry/    loads data/registry with provenance (shared)
    store/       SQLite persistence (server)
  data/
    sources/     raw downloads (gitignored) + MANIFEST with sha256
    registry/    curated YAML facts, each with source_url + retrieved_at + license
  assets/        pixel art + audio, with a LICENSES file
  web/           index.html, wasm_exec.js, service worker
```

### 10.3 Client (Ebitengine → WebAssembly)

- 16 px tiles from one or a few **texture atlases**, with tile layers drawn in order; sprites y-sorted for depth.
- Grid movement with smooth interpolation; keyboard and an on-screen **touch D-pad**, because Pokémon controls suit phones well.
- Chunk cache with LRU eviction; decoding is cheap because chunks arrive pre-baked and compressed.
- UI (dialog boxes, menus, dex, duels, station panels) drawn in-engine, Pokémon-style. No HTML injection anywhere.
- Static assets carry content hashes so Cloudflare can cache them at the edge forever.

### 10.4 Server

- `net/http` plus a WebSocket library; one goroutine per connection for reading, a central tick loop for the world.
- **Binary protocol**, length-prefixed and versioned. Messages: Hello, ChunkRequest/ChunkData, Move, EntityDelta, Interact, Edit, StationOp, LabResult, DexUpdate, DuelOp, MarketOp, Emote.
- **SQLite in WAL mode:** players, parcels, chunk edits (one blob per edited chunk), wine lots (state + log), ledger, dex, medals, duel history. Nightly `VACUUM INTO` backups, plus ZFS snapshots on TrueNAS.
- Limits at every edge: message size, messages per second, connections per IP, chunk requests per second.

### 10.5 Performance budgets (these are the Phase 1/3 gates)

| Budget | Target |
|---|---|
| Client frame time | ≤ 8 ms CPU on a mid-range Android phone and on an M-series Air, 60 fps |
| First playable | ≤ 5 s on a 20 Mbps connection (wasm + first chunks) |
| Wasm download | ≤ 4 MB brotli (a minimal Ebitengine program is 2.5 MB) |
| Allocations per frame (steady state) | 0 |
| Server tick at 200 players online | ≤ 5 ms |
| Server footprint at 200 players | ≤ 1 CPU core, ≤ 512 MB RAM |
| Chunk served (p99) | ≤ 20 ms |
| Bandwidth per player, steady state | ≤ 5 KB/s |

Your **home upload bandwidth** is the real ceiling on players online, not Go. Cloudflare caches the static `.wasm` and art at the edge, so only WebSocket traffic reaches the NAS. Measure the uplink in Phase 0 and turn it into a hard concurrent-player cap.

---

## 11. Hosting on your TrueNAS (public)

- **Path:** `winecraft.francomichetti.com` → Cloudflare token tunnel → Traefik → `winecraft` container. **No host port**: Traefik reaches it on a dedicated Docker network, so the game is never bound on the LAN.
- **This is your first fully public, unauthenticated, interactive app on the box** that also holds your SIEM, smart home and databases. So:
  - non-root user, read-only root filesystem, `no-new-privileges`, all capabilities dropped, CPU and memory limits;
  - its own Docker network, attached to Traefik only, with no route to other containers;
  - data in `/mnt/fast_pool/compose/winecraft/data` only;
  - add the hostname as an **explicit** tunnel route. The `*.francomichetti.com` wildcard catch-all is still on your Cloudflare TODO list;
  - the admin endpoints (metrics, moderation) sit behind your SSO or Cloudflare Access; the game itself is public.
- **WebSockets through Cloudflare:** they work on all plans, but Cloudflare drops idle connections and restarts its servers now and then. So: a game-level heartbeat every ≤ 30 s, and the client reconnects and resumes on its own.
- **Real client IP:** Cloudflare sends `CF-Connecting-IP`, but Traefik's trusted-IP setting covers only `X-Forwarded-*`. Anyone who can reach Traefik directly could fake the header. Make Traefik reachable only from cloudflared, and have the server trust the header only from that hop.
- **Caching the wasm:** Cloudflare compresses `application/wasm` but doesn't cache it by default, so it needs a Cache Rule. If asset traffic grows, the free-plan terms on "large files" point to moving assets to R2.
- **Staging from Phase 3:** the same stack behind Cloudflare Access (only you and testers), so hosting problems show up early, not at launch.
- **Backups and restore:** nightly SQLite backup plus ZFS snapshots. A restore drill is part of the launch gate.
- **Ports:** none on the host. If a debug port is ever needed, check it against the full TCP and UDP listening set first and bind it to `127.0.0.1` only, per your NAS port rule.

---

## 12. Phases and gates

The first playable target is a **vertical slice**: one zone at full detail, one town, two bodegas, the full Malbec chain from vine to bottle, a dex with about 20 entries, one medal, one duel mode, the market, and a shared world for 50 players. The world then grows **zone by zone**.

### Phase 0 — Ground truth and decisions
- Finish the Appendix B audit; build the registry schema and the first 100 sourced facts.
- Download and checksum the elevation model, the OSM extract, INV statistics and the first price tables (Appendix C).
- Fix the bounding box and scale: lay out walking and biking times on the real map at 5 m per tile.
- Settle permissions: the repo owner's OK for the rebuild and hosting; which hero bodegas we ask.
- Settle names: no Nintendo trademarks (§13).
- Measure the home uplink → concurrent-player cap.
- **PASS when:** every fact in the slice has a source record · data manifest checksummed · bbox and scale written down · repo owner agrees · names cleared.

### Phase 1 — Go/WASM spike (about a week)
Ebitengine client to wasm: a 3-layer tilemap scrolling over a real 256×256-tile piece of the valley, 50 fake players moving, a WebSocket echo to a Go server **through a Cloudflare tunnel test hostname**, touch D-pad, iOS audio unlock.
- **PASS when:** the §10.5 frame budget holds on your phone, an iPhone and the Air · first playable ≤ 5 s · wasm ≤ 4 MB brotli · zero steady-state allocations · WebSocket round trips work in Chrome, Android and iOS Safari through the tunnel, with memory flat over 30 minutes · reconnect after a dropped connection works.
- **NOT-PASS →** switch the client to TypeScript + PixiJS on the same Go server and protocol. Write it up in a decision record.

### Phase 2 — World pipeline
`cmd/worldgen` turns elevation + OSM + INV into baked chunks, with projection, height bands, roads, rivers, acequias, vineyards, windbreaks, towns and bodega footprints, and provenance per feature.
- **PASS when:** 10 checkpoints (bodegas, town centres, river crossings) fall within 2 tiles of their sourced coordinates · height bands match the elevation model at sampled points · a full valley bake is reproducible (same input → same checksum) · chunk size budget met.

### Phase 3 — Server core and shared world
Accounts (device token), authoritative movement, interest management, chunk streaming, SQLite store, presets and emotes, rate limits. Staging goes live behind Cloudflare Access.
- **PASS when:** `loadbot` at 200 players meets §10.5 · `kill -9` of the server loses at most the last tick · abuse tests are rejected (oversized frames, floods, speed hacks, edits outside your parcel).

### Phase 4 — Crafting and building
Gathering, recipes, workbench, parcels and claims, placing and breaking, interiors.
- **PASS when:** recipe and parcel tests are green · no edit is possible to the real world or another player's parcel (tests) · a player can build a working cellar from scratch in a playtest.

### Phase 5 — Winemaking simulation
`sim/` written test-first: all §6 stages, lots, the cellar log, the lab, faults, the calendar and time compression.
- **PASS when:** golden tests reproduce the cited reference numbers (e.g., the sugar-to-alcohol conversion, fermentation curves within the cited range) · a full vine-to-bottle playtest works · every stage has a station, a UI panel and a log entry · time compression tuned in a playtest.

### Phase 6 — Pokémon layer
Towns and NPC dialog, quests, the Ampelodex (varieties weighted by hectares, soils, species, bodegas), medals, both duel modes.
- **PASS when:** every dex entry and every duel rule has a source · the duel rules doc is reviewed · the zone's medal can be earned in a playtest.

### Phase 7 — Economy
Price tables, markets, the ledger, parcel purchase, the bodega P&L screen.
- **PASS when:** ledger invariants are green (property-based tests) · a scheduled job flags price records older than N months · a week of simulated bot trading shows no money created from nothing.

### Phase 8 — Public launch
Hardening (§11), public hostname, monitoring, backups.
- **PASS when:** an external probe shows only the public hostname · the restore drill passes · a 48-hour bot soak stays inside budgets · the player cap (from the Phase 0 uplink measurement) is enforced.

### Phase 9 onward — Grow the world
Zone after zone: content, NPCs, interiors, medal, species. Whites and rosé. Real sun position and seasons. Spanish first, then English. Accessibility.

---

## 13. Risks

| Risk | Mitigation |
|---|---|
| Go/WASM too heavy or slow on phones | Phase 1 hard gate; PixiJS fallback on the same Go backend |
| **Pixel art is the largest cost**: someone has to draw hundreds of tiles and sprites | CC0 tilesets for the spike and slice; budget for an artist, or decide to draw; one consistent palette |
| Nintendo IP: patents on specific mechanics, trademarks on names | Nintendo's suit against Pocketpair (Palworld) rests on three Japanese patents: aiming and releasing a capture item at a creature in the field (JP7493117), choosing a capture item or fighting creature then aiming and releasing it (JP7545191), and seamless mount switching (JP7528390). A ruling is expected in November 2026. In the US, the creature-summoning patent (US 12,403,397) had all claims rejected in a non-final April 2026 decision; the mount-switching patent (US 12,409,387) was granted in September 2025. **WineCraft avoids all of these**: you collect by sampling and photographing, never by throwing an item; mounts (bike, truck) are entered and exited, never switched mid-motion. No Pokémon names (no "Pokédex"), no art, no trade dress. Turn-based duels, collections and medals are genre mechanics. Re-check after the November ruling. Not legal advice |
| In-game AI chat terms | Google Antigravity (`agy`) terms forbid use "in connection with products not provided by us" (account bans confirmed); the Gemini API forbids services likely used by under-18s. Use a local model on the NAS, or the paid Gemini API behind an 18+ gate; never agy for players. OWASP LLM Top 10 controls: no tools, server-side quest state, delimited player text, output caps, quotas, fixed-hint fallback |
| Asset licenses | CC0 assets only (e.g., Kenney Tiny Town, ArMM1998 Zelda-like, both 16×16 CC0), or attribution tracked. **No share-alike assets** (LPC is CC-BY-SA/GPL and 32×32 anyway) |
| SQLite on the NAS | One writer at a time, and the database on local disk, never on an SMB/NFS share (WAL mode doesn't work over network filesystems) |
| Real wineries and people | Factual cards only, with sources; no speaking real people without consent; contact the hero bodegas early, since this is good marketing for them |
| Data licenses (ODbL share-alike, GBIF per-record licenses) | Attribution screen; publish the derived world data under ODbL; CC0/CC-BY records only |
| Prices go stale (inflation) | Store USD + date; staleness job; ARS shown only as a dated conversion |
| Home uplink, abuse, DDoS | Cloudflare in front; edge-cached assets; player cap; rate limits; the container is isolated from the rest of the NAS |
| Economy exploits | Server-authoritative ledger with invariants; rate limits; bot soak before launch |
| Scope (this is a big game) | Vertical slice first; one zone at a time; the gates stop a phase from sprawling |

---

## 14. Decisions still open (yours)

1. **Repo:** a new private repo (`onembyte/winecraft`) or contribute to `RaymonOtatti/winecraft`? Either way, the repo has no license, so we need **Raymon's OK** to rebuild and host it publicly.
2. ~~Real money~~ **Settled:** in-game money at real prices, real bodegas in the world, no real-money purchases.
3. **Scale:** 5 m per tile overworld (recommended) after the Phase 0 walk-time check.
4. **Time:** 1 game year = 4 real weeks (recommended).
5. **Hero bodegas:** which bodegas we contact for permission and detailed interiors.
6. **Art:** commission an artist, draw it yourself, or ship the slice on CC0 tiles.
7. **Accounts:** anonymous at launch, with email-link login later (could reuse your Hermod plan).

---

## Appendix A — How the legacy numbers were measured

`node bench/legacy-measure.js` (Node 22). It loads the real `public/js/blocks.js` and `world.js`, stubs THREE so every `addGroup` counts as one draw call (three.js renders one draw per group on a multi-material mesh), runs `generateTerrain()`, then times one block edit and 1M `getBlock` lookups against a flat `Uint8Array`. Results from three runs: 19,946 draw calls · 39,892 triangles · 260–370 ms generation and meshing · 4–12 ms and 890 groups per edit · lookups 35–40× slower than the flat array.

## Appendix B — Audit of the prototype's claims

Checked 2026-10-01 against the sources listed (source ids in Appendix C).

| Claim in the prototype | Verdict | What's actually true |
|---|---|---|
| Zuccardi Valle de Uco: Paraje Altamira, ~1,100 m, Sebastián Zuccardi, opened 2016 | ✅ True | Inaugurated March 2016 (built from 2013). Sebastián Zuccardi is Winemaking Director [Z2][Z6] |
| Zuccardi #1 World's Best Vineyards | ✅ True | #1 in 2019 and 2020 [B1][B2]. A third #1 in 2021 and the Hall of Fame are reported but unconfirmed |
| Zuccardi ferments in epoxy-free concrete | ✅ True | "Hormigón sin epoxi" on the tech sheets [Z7] |
| Piedra Infinita, Gravascal, Supercal: 100 pts Wine Advocate | ◐ Partly | Wine Advocate 100s: Finca Piedra Infinita 2016, Gravascal 2018, Supercal 2019 (Luis Gutiérrez); Gravascal 2021 (Matthew Luczy). Other 100s came from other critics (Tapia, Atkin, Dunnuck) [Z8] |
| "More than 1,000 trucks of stones" | ✅ True | Stated on the official tech sheet [Z5] |
| PerSe: 2012, Edy Del Pópolo & David Bonomi, Gualtallary ~1,450–1,500 m | ✅ True | Santiago del Pópolo is also on the team [P1][P4] |
| PerSe next to the Monasterio del Cristo Orante | ⚠️ True, sensitive | Vines are on monastery land (leased). The monastery was **closed in January 2019** by the Archdiocese after abuse complaints [P3][N1]. Never show it as active |
| La Craie 100 pts | ◐ Partly | 2018 got 100 (Gutiérrez); 2015 and 2016 got 98. 2019 at 100 is unconfirmed [P5] |
| PerSe wines Iubileus, Inseparable, Uní | ◐ Fix name | "Uni del Bonesant" (312 vines, 0.06 ha). 2018: Uní 99, Iubileus 98, Inseparable 94 [P4][P5] |
| Sitio La Estocada, Matías Michelini & family, 4 ha biodynamic | ✅ True | 4 ha, 9 vineyard parcels, 2.111 ha of vines, biodynamic. Address: Ruta 89 – Camino de los Europeos s/n, Gualtallary [E2][E3] |
| "Camino de los Europeos" wine line | ✅ True | "Fase 1": Rosado de Pinot Noir, Sauvignon Blanc, Cabernet Franc; Wine Club only. The Cabernet Franc 2022 is from San Pablo, not the estate [E1][E4] |
| Restaurant Cal, MICHELIN | ✅ True from 2026 | 1 Star + Green Star + Young Chef Award (Enzo González Petra) in the **2026** guide; absent from 2024/2025 [M1][M2] |
| Cru de Montaña by sommelier Rodrigo Calderón | ✅ True | He leads the project [C2] |
| …and Manolo Pelegrina | ◐ Partly | Vineyard manager, not co-owner. Matías Michelini is the collaborating enologist [C1] |
| Cru de Montaña at San Pablo, 1,400–1,470 m | ◐ Partly | San Pablo vineyard at 1,470 m [C1] |
| Wines Cru de San Pablo, Cru de Gualtallary, Días Perfectos | ✅ True | Cru de San Pablo is 85% Cabernet Franc / 15% Malbec. Cru de Gualtallary is from Gualtallary. Días Perfectos includes a San Pablo Malbec and an El Peral Semillón [C1][C3] |
| "Gualtallary" as an appellation | ❌ False | Not an IG; a trademark blocks the name [G1] |
| Tasting notes and "Ficha Oficial" on 2026 vintages | ❌ Invented | Replace with tech-sheet data only, cited |
| Bodega coordinates | ❌ Invented | Zuccardi winery: -33.7730, -69.1569 (OSM way 676944463, DEM 1,079 m). PerSe, Cru de Montaña and Sitio La Estocada have **no OSM feature**, so Phase 0 must source coordinates from the wineries or their official addresses |

**Unconfirmed, so not usable until sourced:** Zuccardi's 2021 #1 and Hall of Fame; La Craie 2019 at 100; La Estocada's altitude (1,350 m) and founding year; PerSe and Cru de Montaña coordinates; which of two OSM "Cristo Orante" features (6 km apart) is right; Cru de Gualtallary at 1,400 m; "Giorgio Bendetti" as a partner; the department of Diam's (DiamAndes).

## Appendix C — Data sources

### C.1 World and geography

| Source | Format | License / credit | Use |
|---|---|---|---|
| Copernicus GLO-30 DEM — https://registry.opendata.aws/copernicus-dem/ | Cloud-optimized GeoTIFF | Free; credit "© DLR e.V. 2010-2014 and © Airbus Defence and Space GmbH 2014-2018 provided under COPERNICUS by the European Union and ESA" | Terrain and height bands (primary) |
| SRTM GL1 v003 — https://www.earthdata.nasa.gov/data/catalog/lpcloud-srtmgl1-003 | GeoTIFF/HGT | No restrictions; cite DOI 10.5067/MEASURES/SRTM/SRTMGL1.003 | Cross-check |
| ALOS AW3D30 — https://www.eorc.jaxa.jp/ALOS/en/dataset/aw3d30/aw3d30_e.htm | GeoTIFF | Free incl. commercial; credit "©JAXA" | Cross-check |
| IGN MDE-Ar v2.1 — https://www.ign.gob.ar/NuestrasActividades/Geodesia/ModeloDigitalElevaciones/faq | .img | Free download; license text not found | Argentina-only, ~2 m vertical accuracy (use only after the license is confirmed) |
| OpenStreetMap via Overpass — https://overpass-api.de · https://www.openstreetmap.org/copyright | JSON/XML/PBF | ODbL: "© OpenStreetMap contributors"; derived DB must be ODbL | Roads, towns, buildings, vineyards (1,527 `landuse=vineyard` areas in the three departments), wineries (43 in the core box). Rivers and canals still to be queried |
| IG resolutions (Boletín Oficial), e.g. https://www.argentina.gob.ar/normativa/nacional/resoluci%C3%B3n-7-2022-377091 | Text vertex lists | Public law | IG boundaries (digitize) |
| INV IG list — https://www.argentina.gob.ar/sites/default/files/i.g._y_d.o.c._de_la_republica_argentina_1.pdf | PDF | Public | Official IG names and resolutions |

**Bounding boxes:** vineyard zone S -34.10, N -33.13, W -69.43, E -68.91 (the core 98% sits in -34.00…-33.29, -69.33…-68.99). With the Cordón del Plata and Volcán Tupungato: S -34.11, N -32.95, W -69.80, E -68.90. The Tupungato summit is on the Chilean border, so the elevation tiles must include the Chilean side. At 5 m per tile the core zone is about 15,800 × 6,300 tiles.

### C.2 Vineyards, wineries, climate

| Source | Format | License | Use |
|---|---|---|---|
| INV vineyard area — https://datos.magyp.gob.ar/dataset/superficie-implantada-con-vinedos-republica-argentina | CSV 2012–2025 | CC BY 4.0 | Hectares by department, locality, variety, planting year, training system → **dex rarity** |
| INV Malbec report 2025 — https://www.argentina.gob.ar/sites/default/files/2018/10/informe_malbec-2025-inv.pdf | PDF | Not stated | Malbec 2024: San Carlos 5,863 ha, Tunuyán 5,679 ha, Tupungato 5,042 ha |
| INV harvest yearbook 2024 — https://www.argentina.gob.ar/sites/default/files/2018/10/anuario_cosecha_y_elaboracion_2024.pdf | PDF | Not stated | Grapes received by variety (no sugar data) |
| INV winery registry — https://www.argentina.gob.ar/inv/vinos/consultas/inscriptos | Web form / PDF | Not stated | Registered bodegas by department |
| Open-Meteo Historical (ERA5) — https://open-meteo.com/en/license | JSON/CSV API | CC BY 4.0 | Daily temperatures, frost → phenology calendar (~25 km grid) |
| INTA SIGA — https://inta.gob.ar/unidades/212000/siga | CSV | "Free download", no license text | La Consulta station (finer detail) |
| DACC Mendoza — http://www.contingencias.mendoza.gov.ar/web1/agrometeorologia/estaciones.html | Web/PDF | Not stated | 28 stations incl. La Consulta, El Peral, Tunuyán; hail and frost reports |

### C.3 Winemaking and fact-check citations

| Id | Source |
|---|---|
| Z1 | Zuccardi harvest report 2014 — https://zuccardiwines.com/wp-content/uploads/2024/05/Zuccardi-Valle-de-Uco-Reporte-de-Cosecha-2014.pdf |
| Z2 | Finca Piedra Infinita 2020 tech sheet — https://zuccardiwines.com/wp-content/uploads/2024/06/FT-ESP-FINCA-PIEDRA-INFINITA-2020.pdf |
| Z3 | Zuccardi Concreto 2022 tech sheet — https://zuccardiwines.com/wp-content/uploads/2024/06/FT-ESP-CONCRETO-2022.pdf |
| Z4 | Supercal 2019 tech sheet — https://www.winesellersltd.com/wp-content/uploads/2022/05/Zuccardi_Finca_Piedra_Infinita_Supercal_Malbec_2019.pdf |
| Z5 | Finca Piedra Infinita 2023 and 2016 tech sheets — https://zuccardiwines.com/wp-content/uploads/2026/03/FT-ESP-FINCA-PIEDRA-INFINITA-2023.pdf · https://zuccardiwines.com/wp-content/uploads/2024/06/FT-ESP-FINCA-PIEDRA-INFINITA-2016.pdf |
| Z6 | Zuccardi history — http://zuccardiwines.com/en/historia/ |
| Z7 | Gravascal 2017 tech sheet — https://zuccardiwines.com/wp-content/uploads/2024/06/FT-ESP-ZUCCARDI-GRAVASCAL-2017.pdf |
| Z8 | Zuccardi 100-point list — https://zuccardiwines.com/wp-content/uploads/2025/09/10-100-POINTS-ingles.pdf |
| B1, B2 | World's Best Vineyards 2019, 2020 — https://wineindustryadvisor.com/2019/07/11/zuccardi-valle-de-uco-ranks-1-on-2019-list/ · https://wineindustryadvisor.com/2020/07/15/zuccardi-awarded-worlds-best-vineyard-south-america/ |
| P1 | Wine Anorak on PerSe — https://wineanorak.com/2020/09/01/perse-stunning-wines-from-gualtallary-in-argentina/ |
| P2 | La Craie 2013 tech sheet — https://persevines.com/wp-content/uploads/2020/12/La-Craie-2013-ingles.pdf |
| P3 | PerSe La Craie page — https://persevines.com/en/producto/per-se-la-craie/ |
| P4 | PerSe site — https://persevines.com/en/ |
| P5 | La Craie 2018 scores — https://thesourcingtable.com/blogs/offers/perse-2018-100-point-releases |
| N1 | Monastery closure — https://www.infobae.com/sociedad/2019/01/04/cerro-un-monasterio-de-mendoza-por-denuncias-de-abuso-sexual-contra-dos-monjes/ |
| E1 | Estocada Cabernet Franc 2022 — https://sitiolaestocada.com/los-vinos/cabernet-franc-2022/ |
| E2, E3 | Sitio La Estocada — https://sitiolaestocada.com/el-lugar/ · https://sitiolaestocada.com/vitivinicultura/viticultura/ |
| E4 | Estocada wines — https://sitiolaestocada.com/los-vinos/ |
| M1, M2 | MICHELIN Argentina 2026 — https://www.cnnbrasil.com.br/viagemegastronomia/gastronomia/guia-michelin-argentina-2026-4-restaurantes-conquistam-a-primeira-estrela/ · https://soloporgusto.com/tres-restaurantes-de-mendoza-reciben-su-primera-estrella-roja-michelin/ |
| C1 | Cru de San Pablo — https://briccowines.com.ar/productos/cru-de-montana-cru-de-san-pablo/ |
| C2 | Cru de Montaña — https://www.rebellion.com.ar/bodegas/cru-de-montana/ |
| C3 | Cru de Montaña range — https://briccowines.com.ar/cru-de-montana/ |
| G1 | Gualtallary IG blocked — https://geografiadelvino.com/2024/03/20/hay-que-hacer-algo-con-gualtallary/ |
| H1 | Altos Las Hormigas Paraje Altamira 2020 — https://altoslashormigas.com/wp-content/uploads/2025/07/TS_Appellation_Paraje_Altamira_2020_ENG.pdf |
| W1 | Catena Zapata Malbec Argentino vintage notes — https://argentina.guides.winefolly.com/wineries/bodega-catena-zapata/wines/catena-zapata-malbec-argentino/vintages/VT-PVFTJKNUQ/ |
| A1 | AWRI small-lot fermentation method — https://www.awri.com.au/wp-content/uploads/small_lot_fermentation_method.pdf |
| A2 | AWRI cold soak — https://www.awri.com.au/industry_support/winemaking_resources/winemaking-practices/winemaking-treatment-cold-soak/ |
| O1 | OIV maximum acceptable limits — https://www.oiv.int/standards/international-code-of-oenological-practices/annexes/maximum-acceptable-limits |
| U1 | Penn State Extension, wine production — https://extension.psu.edu/food-safety-and-quality/grape-and-wine-production/wine-production |
| U2 | Oregon State Extension, malolactic — https://extension.oregonstate.edu/food/wine-beer/conducting-successful-malolactic-fermentation |

### C.5 Go, WebAssembly, hosting, IP

| Topic | Source |
|---|---|
| Ebitengine v2.10 | https://ebitengine.org/en/blog/v2.10.0.html · https://github.com/hajimehoshi/ebiten/releases · https://ebitengine.org/en/documents/webassembly.html |
| TinyGo not planned | https://github.com/hajimehoshi/ebiten/issues/747 |
| Go releases | https://go.dev/doc/devel/release · https://go.dev/doc/go1.26 · https://go.dev/doc/go1.27 |
| coder/websocket (wasm client) | https://pkg.go.dev/github.com/coder/websocket · https://github.com/coder/websocket/blob/master/ws_js.go |
| bgammon (shipped Ebitengine + coder/websocket) | https://codeberg.org/tslocum/boxcars/src/branch/main/go.mod |
| Wasm size with brotli | https://www.tqdev.com/2024-using-brotli-to-deliver-smaller-wasm-files/ |
| Browser memory issue | https://github.com/hajimehoshi/ebiten/issues/1497 |
| Cloudflare WebSockets | https://developers.cloudflare.com/network/websockets/ |
| Cloudflare headers | https://developers.cloudflare.com/fundamentals/reference/http-headers/ |
| Cloudflare cache defaults and compression | https://developers.cloudflare.com/cache/concepts/default-cache-behavior/ · https://developers.cloudflare.com/speed/optimization/content/compression/ |
| Cloudflare service terms | https://www.cloudflare.com/service-specific-terms-application-services/ |
| Traefik entrypoints (trustedIPs) | https://doc.traefik.io/traefik/reference/install-configuration/entrypoints/ |
| Nintendo v. Pocketpair patents | https://gamesfray.com/two-of-nintendos-three-patents-in-suit-against-pocketpair-relate-to-collecting-characters-lets-look-at-the-claims/ · https://www.techdirt.com/2026/07/02/the-nintendo-palworld-patent-suit-appears-to-be-heading-for-a-muted-conclusion/ |
| US patents | https://www.pcgamer.com/gaming-industry/us-patent-office-revokes-nintendos-controversial-pokemon-battling-patent-in-nonfinal-decision/ · https://thisweekinvideogames.com/news/nintendo-pokemon-company-us-patent-filings-creature-summoning-riding/ |
| Pokémon trademarks | https://www.pokemon.com/us/legal/ |
| CC0 tilesets | https://kenney.nl/assets/tiny-town · https://opengameart.org/content/zelda-like-tilesets-and-sprites |
| Pure-Go SQLite | https://pkg.go.dev/modernc.org/sqlite · https://github.com/ncruces/go-sqlite3 · https://sqlite.org/wal.html |
| OSM PBF in Go | https://pkg.go.dev/github.com/paulmach/osm/osmpbf |

### C.6 AI chat backends and safety (researched 2026-10-01)

| Topic | Source |
|---|---|
| Antigravity CLI (`agy`), headless mode, install | https://github.com/google-antigravity/antigravity-cli · https://antigravity.google/docs/cli/headless/ · https://antigravity.google/docs/cli/install/ |
| Antigravity terms ("products not provided by us") | https://antigravity.google/terms/ · https://github.com/google-gemini/gemini-cli/discussions/20632 |
| Gemini API terms (under-18 clause), pricing, safety settings | https://ai.google.dev/gemini-api/terms · https://ai.google.dev/gemini-api/docs/pricing · https://ai.google.dev/gemini-api/docs/safety-settings |
| OWASP LLM Top 10 2025 (prompt injection, excessive agency, prompt leakage, output handling, unbounded consumption) | https://genai.owasp.org/llmrisk/llm01-prompt-injection/ · https://genai.owasp.org/llmrisk/llm062025-excessive-agency/ · https://genai.owasp.org/llmrisk/llm072025-system-prompt-leakage/ · https://genai.owasp.org/llmrisk/llm052025-improper-output-handling/ · https://genai.owasp.org/llmrisk/llm102025-unbounded-consumption/ |

### C.4 Economy (researched 2026-10-01)

**Vineyard area by variety (INV, as of 2025-12-31).** Total Uco: 29,934.5 ha (21% of Mendoza); 109 registered / 97 producing wineries in 2021.

| Variety | Tupungato | Tunuyán | San Carlos | Uco total |
|---|---|---|---|---|
| Malbec | 5,124.4 | 5,697.9 | 5,770.4 | 16,592.7 |
| Cabernet Sauvignon | 941.6 | 1,222.3 | 759.4 | 2,923.3 |
| Chardonnay | 1,193.9 | 629.9 | 256.1 | 2,079.9 |
| Merlot | 515.8 | 490.8 | 177.4 | 1,184.0 |
| Pinot Noir | 548.5 | 401.2 | 194.3 | 1,144.0 |
| Cabernet Franc | 340.5 | 537.6 | 248.5 | 1,126.6 |
| Tempranillo | 264.2 | 201.2 | 552.6 | 1,018.0 |
| Bonarda | 786.7 | 163.2 | 57.7 | 1,007.6 |
| Sauvignon Blanc | 213.6 | 325.1 | 116.8 | 655.5 |
| Syrah | 168.8 | 288.1 | 189.6 | 646.5 |
| Semillón | 169.9 | 19.0 | 42.4 | 231.3 |
| Petit Verdot | 21.7 | 134.0 | 32.5 | 188.2 |
| Torrontés Riojano | 138.0 | 15.6 | 21.2 | 174.8 |
| Pinot Gris | 27.6 | 72.2 | 16.0 | 115.8 |

Rare in Uco (ha): Barbera 35.9, Tannat 28.8, Garnacha 19.7, Corvina 16.3 (97% of Mendoza's is in Tupungato), Croatina 14.9, Riesling 13.7, Carmenère 12.9, Nebbiolo 1.8, Garnacha Blanca 1.6. District examples: La Consulta 5,210.5 ha, Gualtallary 2,868.0, Vista Flores 2,065.8.

**Prices and costs**

| Item | Value | Kind | Date | Source |
|---|---|---|---|---|
| Grapes, Mendoza Tintas A | USD 0.70/kg (Uco Malbec "USD 1 and up") | asked | 2024-01 | https://campoandino.ar/precios-en-dolares-para-la-uva-2024-definieron-vinateros-de-mendoza-y-de-san-juan/ |
| Grapes, Uco | ~500 ARS/kg | offered | 2025-03 | https://bichosdecampo.com/las-bodegas-estan-ofreciendo-un-valor-similar-al-del-ano-pasado-cuando-de-por-medio-tuvimos-una-inflacion-monstruosa-denuncia-mario-leiva-que-defiende-a-los-productores-de-uva-del-valle/ |
| Malbec, Tupungato | 600–680 ARS/kg | paid | 2025 harvest | https://masp.lmneuquen.com/vitivinicultura/crisis-mendoza-uvas-precio-y-un-llamado-urgente-repensar-la-vitivinicultura-n1234713 |
| Fine grapes, Uco | 220 ARS/kg | offered | 2026-02 | https://www.sitioandino.com.ar/departamentales/crisis-la-vitivinicultura-el-valle-uco-productores-podrian-no-cosechar-y-levantar-vinedos-n5718016 |
| Grapes, Mendoza | asked ≥300 / offered ~210 ARS/kg | asked / offered | 2026-03 | https://www.diariodecuyo.com.ar/economia/alfredo-aciar-los-productores-uva-deben-cerrar-precio-al-final-la-elaboracion-no-malvender-n6567530 |
| Bulk basic red / superior red | 350 / 420 ARS/L | list (Bolsa) | 2026-07 | https://www.mendovoz.com/actualidad/panorama-vitivinicola/2026/8/3/vino-granel-la-bolsa-de-comercio-de-mendoza-analiza-el-escenario-exportador-174796.html |
| Bulk Malbec varietal | ~700 ARS/L | reported | 2026-04 | lmneuquen (above) |
| Vineyard cost, Uco 2025/26 | 7,050,000 ARS/ha (revenue 4,400,000; loss 2,650,000) | study (CRA + SRVU) | 2026-07 | https://www.mdzol.com/dinero/advierten-que-producir-vino-ya-implica-perder-265-millones-hectarea-n1561204 |
| Itemized cost model (East Mendoza, trellis) | 323,583 ARS/ha = 24.89 ARS/kg at 13,000 kg/ha | study (INTA) | 2021-01 | https://www.argentina.gob.ar/sites/default/files/estimacion_de_costos_de_produccion_para_vid_de_vinificar_enero_2021_-_inta_final.pdf |
| Vineyard worker, CCT 154/91 | 17,252.84 ARS/day; 431,321/month | agreement | 2026-03/04 | https://soevarivadavia.org.ar/Escalas/Vina_2026-03y04.pdf |
| Harvest pay per bin | 400–500 common, 600–700 fine | paid | 2025 | https://www.mdzol.com/dinero/2025/1/29/pagarian-la-uva-igual-que-el-ano-pasado-se-encendieron-alarmas-entre-los-productores-1184366.html |
| Planted vineyard | Gualtallary USD 70–90k/ha; Tupungato ≤60k; Tunuyán 40k; San Carlos 35k | bank estimate | 2019-02 | https://www.iprofesional.com/vinos/286154-vinos-argentinos-vinos-recomendados-vinos-malbec-Vinos-de-terruno-cuanto-vale-una-hectarea-en-Gualtallary |
| French oak barrique 225 L | USD 1,377 | list | 2026-10 | https://www.wineandbeersupply.com/products/world-cooperage-traditional-series-french-oak-barrel-225l |
| Destemmer | USD 787.60 (small) – 17,750 (W7) | list | 2026-10 | https://dwinesupplies.com/collections/winemaking-equipment/destemmers |
| Bottle / cork / label | USD 0.50–3.00 / 0.30–1.00+ / 0.20–1.00 | range | 2025-06 | https://ashlandcontainer.com/blog/wine-packaging-pricing-guide |
| Bottle cost split (Coviar) | glass 23.44%, labels 9.38%, cork 4.50%, capsule 3.44% | study | 2026-08 | https://www.lanacion.com.ar/economia/campo/es-triste-el-desconocido-dato-que-muchos-ya-advierten-detras-del-valor-de-una-botella-de-vino-nid05082026/ |
| Zuccardi Finca Piedra Infinita 2020 | ARS 230,000 (≈ USD 152) | list | 2026-10 | https://tienda.aldosvinoteca.com/search/?q=piedra+infinita |
| Zuccardi Q Malbec / Serie A Malbec | ARS 27,300 / 17,800 list | list | 2026-10 | https://www.espaciovino.com.ar/vinos-ficha/Zuccardi-Q-Malbec · https://www.espaciovino.com.ar/vinos-ficha/Zuccardi-Serie-A-Malbec |
| PerSe La Craie / Iubileus | USD 250 each (members, shipping included) | list | 2026-10 | https://persevines.com/en/shop/ |
| Salentein Reserva Malbec | ARS 14,600 list | list | 2026-10 | https://www.espaciovino.com.ar/vinos-ficha/Salentein-Reserva-Malbec |

**Machine-readable sources to automate**

| Source | Endpoint | Format |
|---|---|---|
| INV vineyard area 2025 | https://datos.magyp.gob.ar/dataset/5f97a0b9-f677-4657-9e07-0cd3b98da754/resource/b5e49c34-0a6e-448a-80d7-2bbf61632204/download/inv-superficie-viniedos-2025.csv (CKAN: `package_show?id=superficie-implantada-con-vinedos-republica-argentina`) | CSV, annual, CC BY 4.0 |
| BCRA official rate | https://api.bcra.gob.ar/estadisticascambiarias/v1.0/Cotizaciones/USD?fechadesde=YYYY-MM-DD&fechahasta=YYYY-MM-DD | JSON, daily |
| MEP rate | https://api.argentinadatos.com/v1/cotizaciones/dolares/bolsa | JSON, daily |
| Bolsa de Comercio de Mendoza (grapes by variety and department; bulk wine) | `www.bolsamza.com.ar/web2/mercados/uvas/mercado_varietal.php` (origen 17 Tunuyán, 18 Tupungato, 19 San Carlos; variedad 101 Malbec) | HTML tables; **timed out 2026-10-01**, retry in Phase 0 |
