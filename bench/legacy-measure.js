// Headless measurement of the legacy three.js voxel renderer.
// Replays world.js generation with a stubbed THREE and counts what the GPU would be asked to draw.
// Usage: node bench/legacy-measure.js   (Node 22+)
const fs = require('fs');
const path = require('path');
const js = path.join(__dirname, '..', 'public', 'js');

let drawCalls = 0, meshes = 0, triangles = 0;
globalThis.window = globalThis;
globalThis.THREE = {
  Group: class { constructor() { this.children = []; } add(m) { this.children.push(m); } traverse(f) { this.children.forEach(f); } },
  // three.js issues one draw call per geometry group when a mesh has a material array
  BufferGeometry: class { setAttribute(n, a) { if (n === 'position') triangles += a.n / 9; } addGroup() { drawCalls++; } dispose() {} },
  Float32BufferAttribute: class { constructor(arr) { this.n = arr.length; } },
  Mesh: class { constructor() { meshes++; } },
};
window.TextureManager = { getBlockMaterials: () => [] };
for (const f of ['blocks.js', 'world.js']) (0, eval)(fs.readFileSync(path.join(js, f), 'utf8'));

const world = new window.VoxelWorld({ add() {}, remove() {} });
let t = performance.now();
world.generateTerrain();
console.log('world        ', { blocks: world.blocks.size, chunks: world.chunks.size, meshes, drawCallsPerFrame: drawCalls, triangles, genAndMeshMs: Math.round(performance.now() - t) });

drawCalls = 0; t = performance.now();
world.setBlock(5, 8, 5, 0);
console.log('one edit     ', { drawCallGroupsRebuilt: drawCalls, ms: +(performance.now() - t).toFixed(1) });

const N = 1e6;
t = performance.now(); let s = 0;
for (let i = 0; i < N; i++) s += world.getBlock(i % 70 - 35, 5, (i >> 7) % 70 - 35);
const mapMs = performance.now() - t;
const flat = new Uint8Array(80 * 32 * 80);
t = performance.now();
for (let i = 0; i < N; i++) s += flat[((i % 70) * 32 + 5) * 80 + ((i >> 7) % 70)];
const flatMs = performance.now() - t;
console.log('1M lookups   ', { stringKeyMapMs: Math.round(mapMs), typedArrayMs: +flatMs.toFixed(1), ratio: Math.round(mapMs / flatMs) + 'x' });
