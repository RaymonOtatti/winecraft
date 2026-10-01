---
name: winecraft-phase0
description: Phase 0 – Ground truth, data registry, and decisions for WineCraft.
metadata:
  type: project
---
# Phase 0 objectives

1. **Finish the Appendix B audit** – ensure every claim in the slice has a source record. Build the registry schema (`data/registry/*.yaml`) and populate the first ~100 facts.
2. **Download and checksum the elevation model, OSM extract, INV statistics, and price tables** (see Appendix C.1‑C.4). Store raw files under `data/sources/` and list SHA‑256 checksums in `data/sources/MANIFEST.txt`.
3. **Fix the bounding‑box and scale** – lay out walking and biking times on the real map at 5 m per tile. Record the bounding box coordinates and tile scale.
4. **Settle permissions** – we have Raymon’s OK (pending email reply) and need hero‑bodega consent for interior details.
5. **Settle names** – confirm no Nintendo‑related trademarks remain.
6. **Measure home uplink** – run a quick speed‑test (e.g., `speedtest-cli`) and record max downstream bandwidth; derive a concurrent‑player cap.

## Checklist (PASS when all items are satisfied)
- [ ] Every fact in the slice has a source record (`data/registry/*.yaml`).
- [ ] Data manifest checksummed.
- [ ] Bounding‑box and scale written down in `data/world_params.yaml`.
- [ ] Raymon’s permission obtained (see `memory/winecraft-repo-permission.md`).
- [ ] Hero bodegas have consent for interior details.
- [ ] Names cleared (no Pokémon/other trademark issues).
- [x] Home uplink measured and player‑cap noted in `data/player_cap.yaml`.

## Next actions
- Draft email to Raymon (completed, see memory).
- Begin downloading the Copernicus GLO‑30 DEM (large file, ~3 GB) – we’ll checksum it.
- [x] Create the registry schema file (`data/registry/schema.yaml`).

When you’re ready, let me know the speed‑test results or if you’d like me to start the DEM download.
