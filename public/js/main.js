// Main Loop, Dynamic HUD & Bodega Detection for WineCraft: Valle de Uco

class WineCraftApp {
    constructor() {
        this.container = document.getElementById('game-container');
        this.clock = new THREE.Clock();
        this.keys = {};
        this.selectedSlot = 1;
        this.hotbarSlots = [
            { id: 1, blockId: Blocks.DIRT_ALTAMIRA, name: 'Terruño Altamira' },
            { id: 2, blockId: Blocks.DIRT_GUALTALLARY, name: 'Arena Gualtallary' },
            { id: 3, blockId: Blocks.CALIZA_LACRAIE, name: 'Caliza La Craie' },
            { id: 4, blockId: Blocks.GRASS_UCO, name: 'Pasto de Uco' },
            { id: 5, blockId: Blocks.VINE_MALBEC_ALTAMIRA, name: 'Vid Malbec Altamira' },
            { id: 6, blockId: Blocks.VINE_LACRAIE, name: 'Vid La Craie' },
            { id: 7, blockId: Blocks.VINE_BIODINAMICO, name: 'Vid Biodinámica' },
            { id: 8, blockId: Blocks.VINE_SAN_PABLO, name: 'Vid Cru San Pablo' },
            { id: 9, blockId: Blocks.HORMIGON_VASJA, name: 'Vasija de Hormigón' }
        ];

        this.particles = [];
        this.currentBodegaIndex = 0;
        this.bodegaLocations = [
            { name: "Zuccardi Valle de Uco (Paraje Altamira)", pos: new THREE.Vector3(4, 9, 14) },
            { name: "PerSe Vines (Gualtallary Alto)", pos: new THREE.Vector3(-10, 11, -16) },
            { name: "Sitio La Estocada (Gualtallary)", pos: new THREE.Vector3(16, 10, -14) },
            { name: "Cru de Montaña (San Pablo)", pos: new THREE.Vector3(-10, 10, 6) }
        ];

        this.init();
    }

    init() {
        window.TextureManager.init();

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x8bc0ec);
        this.scene.fog = new THREE.FogExp2(0x8bc0ec, 0.012);

        this.camera = new THREE.PerspectiveCamera(72, window.innerWidth / window.innerHeight, 0.1, 1000);

        this.renderer = new THREE.WebGLRenderer({ antialias: true, powerPreference: 'high-performance' });
        this.renderer.setSize(window.innerWidth, window.innerHeight);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        this.renderer.shadowMap.enabled = true;
        this.container.appendChild(this.renderer.domElement);

        const ambientLight = new THREE.AmbientLight(0xfffaed, 0.7);
        this.scene.add(ambientLight);

        this.sunLight = new THREE.DirectionalLight(0xfff5e0, 0.95);
        this.sunLight.position.set(40, 80, 20);
        this.scene.add(this.sunLight);

        // Mundo Voxel de Mendoza
        this.world = new VoxelWorld(this.scene);
        this.world.generateTerrain();

        // Spawn inicial frente a la entrada de Zuccardi Finca Piedra Infinita
        this.player = {
            camera: this.camera,
            position: new THREE.Vector3(4, 9, 14),
            velocity: new THREE.Vector3(0, 0, 0),
            onGround: false
        };
        this.physics = new PhysicsEngine(this.world);

        this.controls = new THREE.PointerLockControls(this.camera, document.body);
        this.setupPointerLock();

        const boxGeo = new THREE.BoxGeometry(1.005, 1.005, 1.005);
        const boxMat = new THREE.MeshBasicMaterial({ color: 0xffffff, wireframe: true, transparent: true, opacity: 0.6 });
        this.highlightBox = new THREE.Mesh(boxGeo, boxMat);
        this.highlightBox.visible = false;
        this.scene.add(this.highlightBox);

        this.setupEventListeners();
        this.loadSavedWorld();

        this.animate = this.animate.bind(this);
        requestAnimationFrame(this.animate);
    }

    setupPointerLock() {
        const blocker = document.getElementById('blocker');
        const instructions = document.getElementById('instructions');

        instructions.addEventListener('click', () => {
            this.controls.lock();
            window.SoundSystem.init();
        });

        this.controls.addEventListener('lock', () => {
            instructions.style.display = 'none';
            blocker.style.display = 'none';
        });

        this.controls.addEventListener('unlock', () => {
            if (!window.WineCodex.isOpen && document.getElementById('tasting-modal').style.display !== 'flex') {
                blocker.style.display = 'flex';
                instructions.style.display = '';
            }
        });
    }

    setupEventListeners() {
        window.addEventListener('resize', () => {
            this.camera.aspect = window.innerWidth / window.innerHeight;
            this.camera.updateProjectionMatrix();
            this.renderer.setSize(window.innerWidth, window.innerHeight);
        });

        // Controles de teclado: WASD + Flechas de Dirección
        window.addEventListener('keydown', (e) => {
            this.keys[e.code] = true;

            // Ranuras de 1 a 9
            if (e.code.startsWith('Digit')) {
                const digit = parseInt(e.code.replace('Digit', ''), 10);
                if (digit >= 1 && digit <= 9) {
                    this.selectHotbarSlot(digit);
                }
            }

            // Tecla 'E' o 'C': Abrir Códice del Sommelier
            if (e.code === 'KeyE' || e.code === 'KeyC') {
                window.WineCodex.toggle();
            }

            // Tecla 'T': Viajar rápidamente entre las 4 Bodegas
            if (e.code === 'KeyT') {
                this.cycleBodegaTeleport();
            }

            // Tecla 'M': Música ambiental
            if (e.code === 'KeyM') {
                const playing = window.SoundSystem.toggleMusic();
                window.WinemakingSystem.showNotification(playing ? '🎵 Música de viñedo iniciada' : '🔇 Música pausada');
            }

            // Tecla 'R': Reaparecer
            if (e.code === 'KeyR') {
                this.player.position.set(4, 9, 14);
                this.player.velocity.set(0, 0, 0);
                window.WinemakingSystem.showNotification('📍 Reapareciste en Zuccardi Piedra Infinita');
            }
        });

        window.addEventListener('keyup', (e) => {
            this.keys[e.code] = false;
        });

        window.addEventListener('wheel', (e) => {
            if (!this.controls.isLocked) return;
            if (e.deltaY > 0) {
                this.selectHotbarSlot(this.selectedSlot === 9 ? 1 : this.selectedSlot + 1);
            } else {
                this.selectHotbarSlot(this.selectedSlot === 1 ? 9 : this.selectedSlot - 1);
            }
        });

        window.addEventListener('mousedown', (e) => {
            if (!this.controls.isLocked) return;
            if (e.button === 0) {
                this.breakBlock();
            } else if (e.button === 2) {
                this.interactOrPlaceBlock();
            }
        });

        window.addEventListener('contextmenu', (e) => e.preventDefault());
    }

    cycleBodegaTeleport() {
        this.currentBodegaIndex = (this.currentBodegaIndex + 1) % this.bodegaLocations.length;
        const target = this.bodegaLocations[this.currentBodegaIndex];
        this.player.position.copy(target.pos);
        this.player.velocity.set(0, 0, 0);
        window.SoundSystem.playPortal();
        window.WinemakingSystem.showNotification(`✨ Viajaste a: ${target.name}`);
    }

    selectHotbarSlot(slot) {
        this.selectedSlot = slot;
        document.querySelectorAll('.hotbar-slot').forEach(el => {
            el.classList.toggle('active', parseInt(el.dataset.slot, 10) === slot);
        });
        const current = this.hotbarSlots[slot - 1];
        if (current) {
            document.getElementById('current-block-name').textContent = current.name;
        }
    }

    breakBlock() {
        const hit = this.physics.raycast(this.camera.position, this.camera.getWorldDirection(new THREE.Vector3()));
        if (!hit.hit) return;

        const { x, y, z } = hit.position;
        const blockId = hit.blockId;
        const blockData = Blocks.get(blockId);

        window.SoundSystem.playBreak();
        this.spawnBlockParticles(x + 0.5, y + 0.5, z + 0.5, blockId);

        if (blockData.drop && blockData.drop.isGrape) {
            window.WinemakingSystem.addGrape(blockData.drop.id, blockData.drop.count);
        }

        this.world.setBlock(x, y, z, 0);
        this.saveWorldDebounced();
    }

    interactOrPlaceBlock() {
        const hit = this.physics.raycast(this.camera.position, this.camera.getWorldDirection(new THREE.Vector3()));
        if (!hit.hit) return;

        const { x, y, z } = hit.position;
        const blockId = hit.blockId;

        // 1. Atril de información de la bodega
        if (blockId === Blocks.PEDESTAL_INFO) {
            window.WinemakingSystem.showPedestalInfo(x, y, z);
            return;
        }

        // 2. Vasija de Hormigón
        if (blockId === Blocks.HORMIGON_VASJA) {
            window.WinemakingSystem.interactVatOrBarrel(x, y, z, true);
            return;
        }

        // 3. Barrica de Roble
        if (blockId === Blocks.BARREL_ROBLE) {
            window.WinemakingSystem.interactVatOrBarrel(x, y, z, false);
            return;
        }

        // 4. Prensa / Lagar
        if (blockId === Blocks.PRESS_LAGAR) {
            window.WinemakingSystem.interactPress(x, y, z);
            return;
        }

        // 5. Colocar bloque en cara adyacente
        const targetX = x + hit.normal.x;
        const targetY = y + hit.normal.y;
        const targetZ = z + hit.normal.z;

        if (this.physics.intersectsPlayer(this.player, targetX, targetY, targetZ)) {
            return;
        }

        const slotData = this.hotbarSlots[this.selectedSlot - 1];
        if (slotData) {
            this.world.setBlock(targetX, targetY, targetZ, slotData.blockId);
            window.SoundSystem.playPlace();
            this.saveWorldDebounced();
        }
    }

    spawnBlockParticles(x, y, z, blockId) {
        const mat = window.TextureManager.getBlockMaterials(blockId);
        const pGeo = new THREE.BoxGeometry(0.12, 0.12, 0.12);
        for (let i = 0; i < 8; i++) {
            const mesh = new THREE.Mesh(pGeo, Array.isArray(mat) ? mat[0] : mat);
            mesh.position.set(x, y, z);
            const vel = new THREE.Vector3(
                (Math.random() - 0.5) * 4,
                Math.random() * 4 + 1,
                (Math.random() - 0.5) * 4
            );
            this.scene.add(mesh);
            this.particles.push({ mesh, vel, life: 0.6 });
        }
    }

    updateParticles(delta) {
        for (let i = this.particles.length - 1; i >= 0; i--) {
            const p = this.particles[i];
            p.life -= delta;
            p.vel.y -= 12 * delta;
            p.mesh.position.addScaledVector(p.vel, delta);
            if (p.life <= 0) {
                this.scene.remove(p.mesh);
                p.mesh.geometry.dispose();
                this.particles.splice(i, 1);
            }
        }
    }

    updateRegionHUD() {
        const badge = document.getElementById('region-badge');
        if (!badge) return;

        const x = this.player.position.x;
        const z = this.player.position.z;

        if (z < -10) {
            if (x < 3) {
                badge.innerHTML = `🕊️ <b>PerSe Vines</b> — Gualtallary Alto (1.450 msnm)<br><small>Dueños: Edy Del Pópolo & David Bonomi | Caliza La Craie</small>`;
                badge.style.borderColor = '#ffffff';
            } else {
                badge.innerHTML = `🌱 <b>Sitio La Estocada</b> — Gualtallary (1.350 msnm)<br><small>Dueño: Matías Michelini | Biodinámica & Cal Restaurante Michelin</small>`;
                badge.style.borderColor = '#449633';
            }
        } else if (z >= -10 && z <= 10) {
            badge.innerHTML = `❄️ <b>Cru de Montaña</b> — San Pablo (1.420 msnm)<br><small>Creadores: Rodrigo Calderón & Manolo Pelegrina | Viento Glaciar</small>`;
            badge.style.borderColor = '#3a76b8';
        } else {
            badge.innerHTML = `🏛️ <b>Zuccardi Valle de Uco</b> — Paraje Altamira (1.100 msnm)<br><small>Director: Sebastián Zuccardi | Finca Piedra Infinita (100 Pts)</small>`;
            badge.style.borderColor = '#cda845';
        }
    }

    updateHighlightBox() {
        const hit = this.physics.raycast(this.camera.position, this.camera.getWorldDirection(new THREE.Vector3()));
        if (hit.hit) {
            this.highlightBox.position.set(hit.position.x + 0.5, hit.position.y + 0.5, hit.position.z + 0.5);
            this.highlightBox.visible = true;
        } else {
            this.highlightBox.visible = false;
        }
    }

    saveWorldDebounced() {
        clearTimeout(this.saveTimer);
        this.saveTimer = setTimeout(() => {
            this.saveWorldToServer();
        }, 1200);
    }

    async saveWorldToServer() {
        const modified = [];
        for (const [key, blockId] of this.world.modifiedBlocks.entries()) {
            const [x, y, z] = key.split(',').map(Number);
            modified.push({ x, y, z, blockId });
        }

        const payload = {
            modifiedBlocks: modified,
            playerPosition: {
                x: this.player.position.x,
                y: this.player.position.y,
                z: this.player.position.z
            },
            inventory: window.WinemakingSystem.inventory
        };

        try {
            await fetch('/api/save_world', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
        } catch (e) {}
    }

    async loadSavedWorld() {
        try {
            const res = await fetch('/api/load_world');
            if (!res.ok) return;
            const data = await res.json();
            if (data.modifiedBlocks && data.modifiedBlocks.length > 0) {
                data.modifiedBlocks.forEach(b => {
                    this.world.setBlock(b.x, b.y, b.z, b.blockId, false);
                });
                this.world.rebuildAllChunks();
            }
            if (data.inventory) {
                window.WinemakingSystem.inventory = { ...window.WinemakingSystem.inventory, ...data.inventory };
                window.WinemakingSystem.updateInventoryUI();
            }
        } catch (e) {
            console.log("Iniciando nuevo mapa de Mendoza.");
        }
    }

    animate() {
        requestAnimationFrame(this.animate);
        const delta = Math.min(this.clock.getDelta(), 0.1);

        if (this.controls.isLocked) {
            this.physics.update(this.player, this.keys, delta);
        }

        this.updateHighlightBox();
        this.updateParticles(delta);
        this.updateRegionHUD();

        this.renderer.render(this.scene, this.camera);
    }
}

window.addEventListener('DOMContentLoaded', () => {
    window.app = new WineCraftApp();
});
