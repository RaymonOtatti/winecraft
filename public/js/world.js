// World Generation: Valle de Uco, Mendoza — Zuccardi, PerSe, Sitio La Estocada y Cru de Montaña

class VoxelWorld {
    constructor(scene) {
        this.scene = scene;
        this.chunkSize = 16;
        this.blocks = new Map();
        this.chunks = new Map();
        this.modifiedBlocks = new Map();
        this.worldBounds = { minX: -36, maxX: 36, minZ: -36, maxZ: 36, maxY: 24 };
    }

    getKey(x, y, z) {
        return `${x},${y},${z}`;
    }

    getChunkKey(cx, cy, cz) {
        return `${cx},${cy},${cz}`;
    }

    getBlock(x, y, z) {
        return this.blocks.get(this.getKey(x, y, z)) || 0;
    }

    setBlock(x, y, z, blockId, markModified = true) {
        const key = this.getKey(x, y, z);
        if (blockId === 0) {
            this.blocks.delete(key);
        } else {
            this.blocks.set(key, blockId);
        }

        if (markModified) {
            this.modifiedBlocks.set(key, blockId);
            this.updateChunkAt(x, y, z);
        }
    }

    updateChunkAt(x, y, z) {
        const cx = Math.floor(x / this.chunkSize);
        const cy = Math.floor(y / this.chunkSize);
        const cz = Math.floor(z / this.chunkSize);

        this.buildChunkMesh(cx, cy, cz);

        const lx = x - cx * this.chunkSize;
        const ly = y - cy * this.chunkSize;
        const lz = z - cz * this.chunkSize;

        if (lx === 0) this.buildChunkMesh(cx - 1, cy, cz);
        if (lx === this.chunkSize - 1) this.buildChunkMesh(cx + 1, cy, cz);
        if (ly === 0) this.buildChunkMesh(cx, cy - 1, cz);
        if (ly === this.chunkSize - 1) this.buildChunkMesh(cx, cy + 1, cz);
        if (lz === 0) this.buildChunkMesh(cx, cy, cz - 1);
        if (lz === this.chunkSize - 1) this.buildChunkMesh(cx, cy, cz + 1);
    }

    generateTerrain() {
        console.log("Generando mapa de Mendoza — Valle de Uco con las 4 bodegas de culto...");

        // 1. TOPOGRAFÍA BASE DEL VALLE DE UCO Y LA CORDILLERA DE LOS ANDES (Oeste, x < -20)
        for (let x = -36; x <= 36; x++) {
            for (let z = -36; z <= 36; z++) {
                let height = 6;

                // Cordillera de los Andes / Cordón del Plata en el flanco Oeste
                if (x < -20) {
                    const mountainDist = (-20 - x) / 16.0;
                    height = Math.floor(7 + mountainDist * 14 + Math.sin(z * 0.35) * 2.2 + Math.cos(x * 0.4) * 2.5);
                } else {
                    // Gradiente altitudinal: El norte (Gualtallary, z < -10) está más alto (1450m) que el sur (Altamira, z > 10, 1100m)
                    const altitudeFactor = (-z + 36) / 72.0 * 2.5;
                    const hill = Math.sin(x * 0.18) * 0.8 + Math.cos(z * 0.15) * 0.7;
                    height = Math.floor(6 + altitudeFactor + hill);
                }

                // Determinar el tipo de suelo según la zona geográfica
                let surfaceDirt = Blocks.DIRT_ALTAMIRA;
                if (z < -10) {
                    surfaceDirt = Blocks.DIRT_GUALTALLARY; // Gualtallary Alto
                } else if (z >= -10 && z <= 10) {
                    surfaceDirt = Blocks.DIRT_SAN_PABLO;    // San Pablo
                } else {
                    surfaceDirt = Blocks.DIRT_ALTAMIRA;      // Paraje Altamira
                }

                for (let y = 0; y <= height; y++) {
                    if (y === height) {
                        if (x < -25 && y > 15) {
                            this.setBlock(x, y, z, Blocks.STONE_ANDES, false); // Cumbres rocosas y glaciares
                        } else {
                            this.setBlock(x, y, z, Blocks.GRASS_UCO, false);
                        }
                    } else if (y >= height - 2) {
                        this.setBlock(x, y, z, surfaceDirt, false);
                    } else {
                        this.setBlock(x, y, z, Blocks.STONE_ANDES, false);
                    }
                }
            }
        }

        // 2. RED DE ACEQUIAS DE DESHIELO DEL RÍO TUNUYÁN
        this.generateAcequias();

        // 3. CAMINOS DE CANTO RODADO ENTRE BODEGAS
        this.generateTrails();

        // 4. BODEGA 1: PERSE VINES (Gualtallary Alto — Edy Del Pópolo & David Bonomi)
        this.generatePerSeBodega();

        // 5. BODEGA 2: SITIO LA ESTOCADA (Gualtallary — Matías Michelini)
        this.generateLaEstocadaBodega();

        // 6. BODEGA 3: CRU DE MONTAÑA (San Pablo — Rodrigo Calderón & Manolo Pelegrina)
        this.generateCruDeMontanaBodega();

        // 7. BODEGA 4: ZUCCARDI VALLE DE UCO (Paraje Altamira — Sebastián Zuccardi)
        this.generateZuccardiBodega();

        // Compilar mallas de chunks
        this.rebuildAllChunks();
    }

    generateAcequias() {
        // Canal principal de deshielo que baja desde los Andes hacia el valle
        for (let x = -20; x <= 26; x++) {
            const z = Math.floor(Math.sin(x * 0.15) * 3);
            const gy = this.getTopBlockY(x, z);
            this.setBlock(x, gy, z, Blocks.WATER_DESHIELO, false);
            this.setBlock(x, gy, z + 1, Blocks.WATER_DESHIELO, false);
        }

        // Ramal norte hacia Gualtallary
        for (let z = -28; z <= -2; z++) {
            const x = 2;
            const gy = this.getTopBlockY(x, z);
            this.setBlock(x, gy, z, Blocks.WATER_DESHIELO, false);
        }

        // Ramal sur hacia Paraje Altamira
        for (let z = 2; z <= 28; z++) {
            const x = -4;
            const gy = this.getTopBlockY(x, z);
            this.setBlock(x, gy, z, Blocks.WATER_DESHIELO, false);
        }
    }

    generateTrails() {
        // Camino de canto rodado norte-sur que une las 4 bodegas
        for (let z = -30; z <= 30; z++) {
            const x = Math.floor(Math.sin(z * 0.1) * 2 + 6);
            const gy = this.getTopBlockY(x, z);
            this.setBlock(x, gy, z, Blocks.PIEDRA_BODEGA, false);
            this.setBlock(x + 1, gy, z, Blocks.PIEDRA_BODEGA, false);
        }
    }

    // =========================================================================
    // BODEGA 1: PERSE VINES (Gualtallary Alto | Edy Del Pópolo & David Bonomi)
    // =========================================================================
    generatePerSeBodega() {
        const bx = -10;
        const bz = -22;
        const baseY = this.getTopBlockY(bx, bz);

        // Terrazas de ladera en suelo calcáreo puro (La Craie)
        for (let x = bx - 6; x <= bx + 6; x++) {
            for (let z = bz - 6; z <= bz + 6; z++) {
                this.setBlock(x, baseY, z, Blocks.CALIZA_LACRAIE, false);
            }
        }

        // Micro-bodega de culto con muros de caliza blanca
        for (let x = bx - 4; x <= bx + 4; x++) {
            for (let z = bz - 4; z <= bz + 4; z++) {
                const isWall = (x === bx - 4 || x === bx + 4 || z === bz - 4 || z === bz + 4);
                if (isWall) {
                    if (z === bz + 4 && (x === bx || x === bx + 1)) {
                        // Puerta de entrada abierta
                    } else {
                        for (let y = baseY + 1; y <= baseY + 4; y++) {
                            this.setBlock(x, y, z, Blocks.CALIZA_LACRAIE, false);
                        }
                    }
                }
                // Techo de madera rústica
                this.setBlock(x, baseY + 5, z, Blocks.WOOD_PLANKS, false);
            }
        }

        // Campanario del Monasterio del Cristo Orante anexo
        for (let y = baseY + 1; y <= baseY + 8; y++) {
            this.setBlock(bx - 4, y, bz - 4, Blocks.WHITE_STONE, false);
        }

        // Piletas de concreto pequeñas y barricas usadas de PerSe
        this.setBlock(bx - 2, baseY + 1, bz - 2, Blocks.HORMIGON_VASJA, false);
        this.setBlock(bx - 1, baseY + 1, bz - 2, Blocks.HORMIGON_VASJA, false);
        this.setBlock(bx + 2, baseY + 1, bz - 2, Blocks.BARREL_ROBLE, false);
        this.setBlock(bx + 2, baseY + 2, bz - 2, Blocks.BARREL_ROBLE, false);

        // Prensa de microvinificación
        this.setBlock(bx + 2, baseY + 1, bz + 1, Blocks.PRESS_LAGAR, false);

        // Pedestal de información con biografía de Edy Del Pópolo y David Bonomi
        this.setBlock(bx, baseY + 1, bz + 5, Blocks.PEDESTAL_INFO, false);

        // Viñedos en pendiente de "La Craie" (Malbec y Cabernet Franc a 1.450 msnm)
        for (let z = bz - 10; z <= bz - 6; z += 2) {
            for (let x = bx - 10; x <= bx + 2; x++) {
                const gy = this.getTopBlockY(x, z);
                this.setBlock(x, gy + 1, z, Blocks.VINE_LACRAIE, false);
                this.setBlock(x, gy + 2, z, Blocks.VINE_LACRAIE, false);
            }
        }
    }

    // =========================================================================
    // BODEGA 2: SITIO LA ESTOCADA (Gualtallary | Matías Michelini)
    // =========================================================================
    generateLaEstocadaBodega() {
        const bx = 16;
        const bz = -20;
        const baseY = this.getTopBlockY(bx, bz);

        // Finca biodinámica con casona encalada y restaurante "Cal"
        for (let x = bx - 5; x <= bx + 5; x++) {
            for (let z = bz - 5; z <= bz + 5; z++) {
                this.setBlock(x, baseY, z, Blocks.WHITE_STONE, false);
            }
        }

        // Muros del restaurante Cal (Guía Michelin)
        for (let x = bx - 4; x <= bx + 4; x++) {
            for (let z = bz - 4; z <= bz + 4; z++) {
                const isWall = (x === bx - 4 || x === bx + 4 || z === bz - 4 || z === bz + 4);
                if (isWall) {
                    if (z === bz + 4 && x === bx) {
                        // Entrada
                    } else {
                        for (let y = baseY + 1; y <= baseY + 3; y++) {
                            this.setBlock(x, y, z, Blocks.WHITE_STONE, false);
                        }
                    }
                }
                this.setBlock(x, baseY + 4, z, Blocks.WOOD_PLANKS, false);
            }
        }

        // Huevos de hormigón biodinámicos
        this.setBlock(bx - 2, baseY + 1, bz - 2, Blocks.HORMIGON_VASJA, false);
        this.setBlock(bx + 2, baseY + 1, bz - 2, Blocks.HORMIGON_VASJA, false);
        this.setBlock(bx - 2, baseY + 1, bz + 1, Blocks.BARREL_ROBLE, false);

        // Mesas de cata de Cal Restaurante en la huerta
        this.setBlock(bx + 2, baseY + 1, bz + 2, Blocks.WOOD_PLANKS, false);

        // Prensa de uvas
        this.setBlock(bx - 3, baseY + 1, bz + 2, Blocks.PRESS_LAGAR, false);

        // Pedestal informativo de Matías Michelini
        this.setBlock(bx, baseY + 1, bz + 5, Blocks.PEDESTAL_INFO, false);

        // Viñedos biodinámicos de Cabernet Franc y Chardonnay ("Camino de los Europeos")
        for (let z = bz - 9; z <= bz - 6; z += 2) {
            for (let x = bx - 2; x <= bx + 10; x++) {
                const gy = this.getTopBlockY(x, z);
                this.setBlock(x, gy + 1, z, Blocks.VINE_BIODINAMICO, false);
                this.setBlock(x, gy + 2, z, Blocks.VINE_BIODINAMICO, false);
            }
        }
    }

    // =========================================================================
    // BODEGA 3: CRU DE MONTAÑA (San Pablo | Rodrigo Calderón & Manolo Pelegrina)
    // =========================================================================
    generateCruDeMontanaBodega() {
        const bx = -10;
        const bz = 0;
        const baseY = this.getTopBlockY(bx, bz);

        // Plataforma de puesto de montaña al pie del Cordón del Plata (1420 msnm)
        for (let x = bx - 5; x <= bx + 5; x++) {
            for (let z = bz - 5; z <= bz + 5; z++) {
                this.setBlock(x, baseY, z, Blocks.DIRT_SAN_PABLO, false);
            }
        }

        // Construcción rústica andina de piedra andesita y madera
        for (let x = bx - 4; x <= bx + 4; x++) {
            for (let z = bz - 4; z <= bz + 4; z++) {
                const isWall = (x === bx - 4 || x === bx + 4 || z === bz - 4 || z === bz + 4);
                if (isWall) {
                    if (z === bz + 4 && (x === bx || x === bx + 1)) {
                        // Entrada abierta hacia el glaciar
                    } else {
                        for (let y = baseY + 1; y <= baseY + 3; y++) {
                            this.setBlock(x, y, z, Blocks.STONE_ANDES, false);
                        }
                    }
                }
                this.setBlock(x, baseY + 4, z, Blocks.WOOD_PLANKS, false);
            }
        }

        // Mirador al Cordón del Plata con barricas borgoñonas
        this.setBlock(bx - 2, baseY + 1, bz - 2, Blocks.BARREL_ROBLE, false);
        this.setBlock(bx - 1, baseY + 1, bz - 2, Blocks.BARREL_ROBLE, false);
        this.setBlock(bx + 2, baseY + 1, bz - 2, Blocks.HORMIGON_VASJA, false);

        // Prensa tradicional
        this.setBlock(bx + 2, baseY + 1, bz + 2, Blocks.PRESS_LAGAR, false);

        // Pedestal informativo de Rodrigo Calderón y Manolo Pelegrina
        this.setBlock(bx, baseY + 1, bz + 5, Blocks.PEDESTAL_INFO, false);

        // Viñedos de gran pendiente de Cabernet Franc y Malbec (Cru de San Pablo)
        for (let z = bz - 4; z <= bz + 4; z += 2) {
            for (let x = bx - 14; x <= bx - 6; x++) {
                const gy = this.getTopBlockY(x, z);
                this.setBlock(x, gy + 1, z, Blocks.VINE_SAN_PABLO, false);
                this.setBlock(x, gy + 2, z, Blocks.VINE_SAN_PABLO, false);
            }
        }
    }

    // =========================================================================
    // BODEGA 4: ZUCCARDI VALLE DE UCO (Paraje Altamira | Sebastián Zuccardi)
    // =========================================================================
    generateZuccardiBodega() {
        const bx = 4;
        const bz = 20;
        const baseY = this.getTopBlockY(bx, bz);

        // Finca Piedra Infinita: Plataforma monumental de piedras aluviales
        for (let x = bx - 7; x <= bx + 7; x++) {
            for (let z = bz - 6; z <= bz + 6; z++) {
                this.setBlock(x, baseY, z, Blocks.PIEDRA_BODEGA, false);
            }
        }

        // Estructura icónica geométrica moderna de concreto y piedra
        for (let x = bx - 5; x <= bx + 5; x++) {
            for (let z = bz - 5; z <= bz + 5; z++) {
                const isWall = (x === bx - 5 || x === bx + 5 || z === bz - 5 || z === bz + 5);
                if (isWall) {
                    if (z === bz - 5 && (x === bx || x === bx + 1)) {
                        // Gran portal de entrada
                    } else {
                        for (let y = baseY + 1; y <= baseY + 5; y++) {
                            this.setBlock(x, y, z, (y === baseY + 5) ? Blocks.HORMIGON_VASJA : Blocks.PIEDRA_BODEGA, false);
                        }
                    }
                }
                // Techo arquitectónico de concreto con cúpula
                this.setBlock(x, baseY + 6, z, Blocks.HORMIGON_VASJA, false);
            }
        }

        // Cúpula central
        this.setBlock(bx, baseY + 7, bz, Blocks.HORMIGON_VASJA, false);

        // Batería de vasijas y piletas troncocónicas de hormigón sin epoxi (firma de Sebastián Zuccardi)
        for (let pz = bz - 3; pz <= bz + 3; pz += 2) {
            this.setBlock(bx - 3, baseY + 1, pz, Blocks.HORMIGON_VASJA, false);
            this.setBlock(bx - 3, baseY + 2, pz, Blocks.HORMIGON_VASJA, false);
            this.setBlock(bx + 3, baseY + 1, pz, Blocks.HORMIGON_VASJA, false);
            this.setBlock(bx + 3, baseY + 2, pz, Blocks.HORMIGON_VASJA, false);
        }

        // Sala de barricas y restaurante "Piedra Infinita Cocina"
        this.setBlock(bx - 1, baseY + 1, bz - 2, Blocks.BARREL_ROBLE, false);
        this.setBlock(bx, baseY + 1, bz - 2, Blocks.BARREL_ROBLE, false);
        this.setBlock(bx + 1, baseY + 1, bz - 2, Blocks.BARREL_ROBLE, false);

        // Prensa de alta precisión
        this.setBlock(bx, baseY + 1, bz + 2, Blocks.PRESS_LAGAR, false);

        // Pedestal informativo de Sebastián Zuccardi
        this.setBlock(bx, baseY + 1, bz - 6, Blocks.PEDESTAL_INFO, false);

        // Viñedos Finca Piedra Infinita (Malbec de Paraje Altamira entre cantos rodados)
        for (let z = bz + 8; z <= bz + 14; z += 2) {
            for (let x = bx - 10; x <= bx + 10; x++) {
                const gy = this.getTopBlockY(x, z);
                this.setBlock(x, gy + 1, z, Blocks.VINE_MALBEC_ALTAMIRA, false);
                this.setBlock(x, gy + 2, z, Blocks.VINE_MALBEC_ALTAMIRA, false);
            }
        }
    }

    getTopBlockY(x, z) {
        for (let y = this.worldBounds.maxY; y >= 0; y--) {
            const b = this.getBlock(x, y, z);
            if (b !== 0 && Blocks.get(b).solid) {
                return y;
            }
        }
        return 6;
    }

    rebuildAllChunks() {
        const chunkCoords = new Set();
        for (const [key] of this.blocks.entries()) {
            const [x, y, z] = key.split(',').map(Number);
            const cx = Math.floor(x / this.chunkSize);
            const cy = Math.floor(y / this.chunkSize);
            const cz = Math.floor(z / this.chunkSize);
            chunkCoords.add(this.getChunkKey(cx, cy, cz));
        }

        for (const ckey of chunkCoords) {
            const [cx, cy, cz] = ckey.split(',').map(Number);
            this.buildChunkMesh(cx, cy, cz);
        }
    }

    buildChunkMesh(cx, cy, cz) {
        const chunkKey = this.getChunkKey(cx, cy, cz);
        const existingMesh = this.chunks.get(chunkKey);
        if (existingMesh) {
            this.scene.remove(existingMesh);
            existingMesh.traverse(child => {
                if (child.geometry) child.geometry.dispose();
            });
            this.chunks.delete(chunkKey);
        }

        const startX = cx * this.chunkSize;
        const startY = cy * this.chunkSize;
        const startZ = cz * this.chunkSize;

        const blockTypeGeometries = {};

        const faces = [
            { dir: [1, 0, 0], corners: [[1, 0, 0], [1, 1, 0], [1, 1, 1], [1, 0, 1]], norm: [1, 0, 0] },
            { dir: [-1, 0, 0], corners: [[0, 0, 1], [0, 1, 1], [0, 1, 0], [0, 0, 0]], norm: [-1, 0, 0] },
            { dir: [0, 1, 0], corners: [[0, 1, 1], [1, 1, 1], [1, 1, 0], [0, 1, 0]], norm: [0, 1, 0] },
            { dir: [0, -1, 0], corners: [[0, 0, 0], [1, 0, 0], [1, 0, 1], [0, 0, 1]], norm: [0, -1, 0] },
            { dir: [0, 0, 1], corners: [[1, 0, 1], [1, 1, 1], [0, 1, 1], [0, 0, 1]], norm: [0, 0, 1] },
            { dir: [0, 0, -1], corners: [[0, 0, 0], [0, 1, 0], [1, 1, 0], [1, 0, 0]], norm: [0, 0, -1] }
        ];

        let hasBlocks = false;

        for (let x = startX; x < startX + this.chunkSize; x++) {
            for (let y = startY; y < startY + this.chunkSize; y++) {
                for (let z = startZ; z < startZ + this.chunkSize; z++) {
                    const blockId = this.getBlock(x, y, z);
                    if (blockId === 0) continue;

                    hasBlocks = true;
                    if (!blockTypeGeometries[blockId]) {
                        blockTypeGeometries[blockId] = {
                            positions: [],
                            normals: [],
                            uvs: [],
                            groups: []
                        };
                    }

                    const bg = blockTypeGeometries[blockId];

                    for (let f = 0; f < faces.length; f++) {
                        const face = faces[f];
                        const nx = x + face.dir[0];
                        const ny = y + face.dir[1];
                        const nz = z + face.dir[2];

                        const neighborId = this.getBlock(nx, ny, nz);
                        const neighborSolid = (neighborId !== 0 && Blocks.get(neighborId).solid && neighborId !== Blocks.WATER_DESHIELO);

                        if (!neighborSolid || blockId === Blocks.WATER_DESHIELO) {
                            const c = face.corners;
                            const startVertex = bg.positions.length / 3;

                            const v0 = [x + c[0][0], y + c[0][1], z + c[0][2]];
                            const v1 = [x + c[1][0], y + c[1][1], z + c[1][2]];
                            const v2 = [x + c[2][0], y + c[2][1], z + c[2][2]];
                            const v3 = [x + c[3][0], y + c[3][1], z + c[3][2]];

                            bg.positions.push(...v0, ...v1, ...v2);
                            bg.normals.push(...face.norm, ...face.norm, ...face.norm);
                            bg.uvs.push(0, 0, 0, 1, 1, 1);

                            bg.positions.push(...v0, ...v2, ...v3);
                            bg.normals.push(...face.norm, ...face.norm, ...face.norm);
                            bg.uvs.push(0, 0, 1, 1, 1, 0);

                            bg.groups.push({ start: startVertex, count: 6, materialIndex: f });
                        }
                    }
                }
            }
        }

        if (!hasBlocks) return;

        const chunkGroup = new THREE.Group();

        for (const [blockIdStr, bg] of Object.entries(blockTypeGeometries)) {
            const blockId = Number(blockIdStr);
            if (bg.positions.length === 0) continue;

            const geometry = new THREE.BufferGeometry();
            geometry.setAttribute('position', new THREE.Float32BufferAttribute(bg.positions, 3));
            geometry.setAttribute('normal', new THREE.Float32BufferAttribute(bg.normals, 3));
            geometry.setAttribute('uv', new THREE.Float32BufferAttribute(bg.uvs, 2));

            bg.groups.forEach(g => {
                geometry.addGroup(g.start, g.count, g.materialIndex);
            });

            const materials = window.TextureManager.getBlockMaterials(blockId);
            const mesh = new THREE.Mesh(geometry, materials);
            chunkGroup.add(mesh);
        }

        this.scene.add(chunkGroup);
        this.chunks.set(chunkKey, chunkGroup);
    }
}

window.VoxelWorld = VoxelWorld;
