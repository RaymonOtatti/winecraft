// Textures Generator: Texturas procedimentales pixel-art para el Valle de Uco, Mendoza

const TextureManager = {
    cache: {},

    init() {
        this.materials = {};
    },

    createCanvas(width = 32, height = 32) {
        const canvas = document.createElement('canvas');
        canvas.width = width;
        canvas.height = height;
        const ctx = canvas.getContext('2d');
        ctx.imageSmoothingEnabled = false;
        return { canvas, ctx };
    },

    randomNoise(ctx, width, height, alpha = 0.08) {
        for (let x = 0; x < width; x++) {
            for (let y = 0; y < height; y++) {
                if (Math.random() > 0.4) {
                    const val = Math.random() > 0.5 ? 255 : 0;
                    ctx.fillStyle = `rgba(${val},${val},${val},${Math.random() * alpha})`;
                    ctx.fillRect(x, y, 1, 1);
                }
            }
        }
    },

    generateTexture(type) {
        if (this.cache[type]) return this.cache[type];
        const { canvas, ctx } = this.createCanvas(32, 32);

        switch (type) {
            case 'dirt_altamira': // Suelo de Paraje Altamira: cantos rodados con calcáreo blanco
                ctx.fillStyle = '#7a543e';
                ctx.fillRect(0, 0, 32, 32);
                // Cantos rodados cubiertos de cal blanco
                const altamiraPebbles = [[6, 8, 4], [20, 12, 5], [10, 22, 3], [24, 24, 4], [16, 4, 3]];
                altamiraPebbles.forEach(([px, py, r]) => {
                    ctx.fillStyle = '#e8e6df'; // Capa de calcáreo blanco
                    ctx.beginPath();
                    ctx.arc(px, py, r, 0, Math.PI * 2);
                    ctx.fill();
                    ctx.fillStyle = '#8f887c'; // Núcleo de piedra andina
                    ctx.beginPath();
                    ctx.arc(px, py, r - 1.2, 0, Math.PI * 2);
                    ctx.fill();
                });
                this.randomNoise(ctx, 32, 32, 0.12);
                break;

            case 'dirt_gualtallary': // Suelo de Gualtallary: arena volcánica y grava calcárea
                ctx.fillStyle = '#8c765c';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#f0ede6';
                for (let i = 0; i < 24; i++) {
                    const x = (i * 7 + 2) % 30;
                    const y = (i * 11 + 4) % 30;
                    ctx.fillRect(x, y, 2, 2);
                }
                this.randomNoise(ctx, 32, 32, 0.14);
                break;

            case 'dirt_san_pablo': // Suelo de San Pablo: altura fría, piedras oscuras y jarilla
                ctx.fillStyle = '#594b3f';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#3f3830';
                for (let i = 0; i < 14; i++) {
                    ctx.fillRect((i * 13) % 28, (i * 9) % 28, 3, 2);
                }
                ctx.fillStyle = '#687834'; // Puntos de jarilla andina
                for (let i = 0; i < 8; i++) {
                    ctx.fillRect((i * 15 + 3) % 30, (i * 7 + 5) % 30, 2, 1);
                }
                this.randomNoise(ctx, 32, 32, 0.1);
                break;

            case 'grass_uco_top': // Pasto de altura con matices dorados y verdes secos
                ctx.fillStyle = '#7a8d42';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#97aa4e';
                for (let i = 0; i < 35; i++) {
                    ctx.fillRect((i * 11) % 32, (i * 7) % 32, 2, 2);
                }
                ctx.fillStyle = '#5f6f2e';
                for (let i = 0; i < 20; i++) {
                    ctx.fillRect((i * 13) % 32, (i * 5) % 32, 2, 1);
                }
                this.randomNoise(ctx, 32, 32, 0.08);
                break;

            case 'grass_uco_side':
                ctx.fillStyle = '#7a543e';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#7a8d42';
                ctx.fillRect(0, 0, 32, 7);
                for (let x = 0; x < 32; x += 3) {
                    const drop = 3 + (x % 5);
                    ctx.fillRect(x, 7, 2, drop);
                }
                this.randomNoise(ctx, 32, 32, 0.1);
                break;

            case 'stone_andes': // Roca andina de la Cordillera
                ctx.fillStyle = '#4c4c58';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#656577';
                for (let i = 0; i < 18; i++) {
                    ctx.fillRect((i * 9) % 28, (i * 13) % 28, 4, 3);
                }
                this.randomNoise(ctx, 32, 32, 0.14);
                break;

            case 'caliza_lacraie': // Caliza tiza blanca pura (PerSe - La Craie)
                ctx.fillStyle = '#f4f2ec';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#dedad0';
                for (let i = 0; i < 20; i++) {
                    ctx.fillRect((i * 7) % 30, (i * 11) % 30, 3, 2);
                }
                this.randomNoise(ctx, 32, 32, 0.07);
                break;

            case 'grapevine_malbec_altamira': // Malbec de Zuccardi (uvas moradas intensas)
                ctx.fillStyle = '#4a3220';
                ctx.fillRect(13, 0, 6, 32);
                ctx.fillStyle = '#2d5e24';
                ctx.fillRect(4, 2, 24, 16);
                ctx.fillStyle = '#3e7d32';
                ctx.fillRect(6, 4, 20, 12);
                // Racimos Malbec Altamira
                [[6, 17], [18, 16], [12, 20]].forEach(([cx, cy]) => {
                    ctx.fillStyle = '#1e092b';
                    ctx.fillRect(cx, cy, 6, 8);
                    ctx.fillStyle = '#4a1564';
                    ctx.fillRect(cx + 1, cy + 1, 2, 2);
                    ctx.fillRect(cx + 3, cy + 3, 2, 2);
                    ctx.fillStyle = '#dcd8cf'; // Pizca de polvillo calcáreo en el hollejo
                    ctx.fillRect(cx + 2, cy + 5, 2, 1);
                });
                break;

            case 'grapevine_lacraie': // Vid PerSe La Craie (Malbec y Cabernet Franc de alta ladera)
                ctx.fillStyle = '#422a18';
                ctx.fillRect(13, 0, 6, 32);
                ctx.fillStyle = '#23501c';
                ctx.fillRect(4, 2, 24, 16);
                ctx.fillStyle = '#37732d';
                ctx.fillRect(6, 4, 20, 12);
                // Racimos pequeños y concentrados de gran altura
                [[7, 16], [19, 17], [13, 21]].forEach(([cx, cy]) => {
                    ctx.fillStyle = '#170624';
                    ctx.fillRect(cx, cy, 5, 7);
                    ctx.fillStyle = '#3c0f4f';
                    ctx.fillRect(cx + 1, cy + 1, 2, 2);
                    ctx.fillRect(cx + 2, cy + 3, 2, 2);
                });
                break;

            case 'grapevine_biodinamico': // Vid Biodinámica de Sitio La Estocada (Matías Michelini)
                ctx.fillStyle = '#452d1c';
                ctx.fillRect(13, 0, 6, 32);
                ctx.fillStyle = '#2e6b22';
                ctx.fillRect(4, 2, 24, 16);
                ctx.fillStyle = '#449633';
                ctx.fillRect(6, 4, 20, 12);
                // Racimos vivos y flores silvestres
                [[7, 16], [18, 17], [12, 20]].forEach(([cx, cy]) => {
                    ctx.fillStyle = '#3a0c20';
                    ctx.fillRect(cx, cy, 5, 7);
                    ctx.fillStyle = '#69183d';
                    ctx.fillRect(cx + 1, cy + 1, 2, 2);
                });
                // Pequeñas flores silvestres amarillas de la biodinámica
                ctx.fillStyle = '#f1c40f';
                ctx.fillRect(5, 5, 2, 2);
                ctx.fillRect(23, 8, 2, 2);
                break;

            case 'grapevine_san_pablo': // Cabernet Franc de Cru de Montaña (Rodrigo Calderón)
                ctx.fillStyle = '#3d2616';
                ctx.fillRect(13, 0, 6, 32);
                ctx.fillStyle = '#26541f';
                ctx.fillRect(4, 2, 24, 16);
                ctx.fillStyle = '#3a7630';
                ctx.fillRect(6, 4, 20, 12);
                // Racimos azulados de Cabernet Franc de montaña
                [[6, 16], [18, 16], [12, 19]].forEach(([cx, cy]) => {
                    ctx.fillStyle = '#141829';
                    ctx.fillRect(cx, cy, 6, 8);
                    ctx.fillStyle = '#283759';
                    ctx.fillRect(cx + 1, cy + 1, 2, 2);
                    ctx.fillRect(cx + 3, cy + 3, 2, 2);
                });
                break;

            case 'hormigon_vasija': // Vasija/Pileta de concreto sin epoxi (Zuccardi / Michelini)
                ctx.fillStyle = '#8b8a87';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#9e9d99';
                for (let i = 0; i < 25; i++) {
                    ctx.fillRect((i * 11) % 30, (i * 7) % 30, 3, 3);
                }
                ctx.fillStyle = '#6f6e6b';
                ctx.fillRect(0, 0, 32, 2);
                ctx.fillRect(0, 30, 32, 2);
                ctx.fillRect(0, 0, 2, 32);
                ctx.fillRect(30, 0, 2, 32);
                // Válvula de acero inoxidable en el centro
                ctx.fillStyle = '#dcdfe3';
                ctx.fillRect(14, 22, 4, 4);
                this.randomNoise(ctx, 32, 32, 0.1);
                break;

            case 'barrel_roble_side': // Barrica de roble
                ctx.fillStyle = '#874d25';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#6f3c1b';
                for (let x = 0; x < 32; x += 8) ctx.fillRect(x, 0, 1, 32);
                ctx.fillStyle = '#3a3a40';
                ctx.fillRect(0, 5, 32, 4);
                ctx.fillRect(0, 23, 32, 4);
                this.randomNoise(ctx, 32, 32, 0.08);
                break;

            case 'barrel_roble_top':
                ctx.fillStyle = '#7a4420';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#5a3014';
                ctx.beginPath();
                ctx.arc(16, 16, 14, 0, Math.PI * 2);
                ctx.stroke();
                ctx.fillStyle = '#9c2438';
                ctx.fillRect(13, 13, 6, 6);
                break;

            case 'piedra_bodega': // Muro de piedras de canto rodado (Zuccardi Piedra Infinita)
                ctx.fillStyle = '#78736a';
                ctx.fillRect(0, 0, 32, 32);
                const stones = [
                    [2, 2, 13, 8], [17, 2, 13, 8],
                    [2, 12, 10, 8], [14, 12, 16, 8],
                    [2, 22, 14, 8], [18, 22, 12, 8]
                ];
                stones.forEach(([sx, sy, sw, sh]) => {
                    ctx.fillStyle = '#8f887c';
                    ctx.fillRect(sx, sy, sw, sh);
                    ctx.fillStyle = '#d6d2c7'; // Borde calcáreo
                    ctx.strokeRect(sx, sy, sw, sh);
                });
                this.randomNoise(ctx, 32, 32, 0.1);
                break;

            case 'press_lagar': // Prensa de madera
                ctx.fillStyle = '#5c351c';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#3a200f';
                for (let x = 4; x < 28; x += 4) ctx.fillRect(x, 6, 1, 20);
                ctx.fillStyle = '#404040';
                ctx.fillRect(14, 0, 4, 32);
                ctx.fillRect(6, 2, 20, 4);
                ctx.fillStyle = '#640c23';
                ctx.fillRect(10, 26, 12, 6);
                break;

            case 'water_deshielo': // Acequia del Río Tunuyán
                ctx.fillStyle = '#339ad4';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#6bcbf7';
                for (let y = 4; y < 32; y += 8) {
                    ctx.fillRect(2, y, 12, 2);
                    ctx.fillRect(18, y + 4, 10, 2);
                }
                break;

            case 'pedestal_info': // Atril de bodega con placa dorada
                ctx.fillStyle = '#4c4b54';
                ctx.fillRect(0, 0, 32, 32);
                // Placa dorada con información
                ctx.fillStyle = '#cda845';
                ctx.fillRect(4, 4, 24, 24);
                ctx.fillStyle = '#e8c96b';
                ctx.fillRect(6, 6, 20, 20);
                ctx.fillStyle = '#3d2b07';
                ctx.fillRect(8, 10, 16, 2);
                ctx.fillRect(8, 14, 12, 2);
                ctx.fillRect(8, 18, 14, 2);
                break;

            case 'wood_planks':
                ctx.fillStyle = '#945e36';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#704221';
                for (let y = 0; y < 32; y += 8) ctx.fillRect(0, y, 32, 1);
                this.randomNoise(ctx, 32, 32, 0.08);
                break;

            case 'white_stone': // Muro blanco encalado (Restaurante Cal / Monasterio)
                ctx.fillStyle = '#f0eee9';
                ctx.fillRect(0, 0, 32, 32);
                ctx.fillStyle = '#dedad2';
                for (let y = 0; y < 32; y += 8) {
                    ctx.fillRect(0, y, 32, 1);
                }
                this.randomNoise(ctx, 32, 32, 0.06);
                break;

            default:
                ctx.fillStyle = '#ff00ff';
                ctx.fillRect(0, 0, 32, 32);
        }

        const texture = new THREE.CanvasTexture(canvas);
        texture.magFilter = THREE.NearestFilter;
        texture.minFilter = THREE.NearestFilter;
        this.cache[type] = texture;
        return texture;
    },

    getBlockMaterials(typeId) {
        if (this.materials[typeId]) return this.materials[typeId];

        let mats;
        switch (typeId) {
            case 1: // DIRT_ALTAMIRA
                const matAltamira = new THREE.MeshLambertMaterial({ map: this.generateTexture('dirt_altamira') });
                mats = [matAltamira, matAltamira, matAltamira, matAltamira, matAltamira, matAltamira];
                break;
            case 2: // DIRT_GUALTALLARY
                const matGual = new THREE.MeshLambertMaterial({ map: this.generateTexture('dirt_gualtallary') });
                mats = [matGual, matGual, matGual, matGual, matGual, matGual];
                break;
            case 3: // DIRT_SAN_PABLO
                const matSP = new THREE.MeshLambertMaterial({ map: this.generateTexture('dirt_san_pablo') });
                mats = [matSP, matSP, matSP, matSP, matSP, matSP];
                break;
            case 4: // GRASS_UCO
                const gTop = new THREE.MeshLambertMaterial({ map: this.generateTexture('grass_uco_top') });
                const gSide = new THREE.MeshLambertMaterial({ map: this.generateTexture('grass_uco_side') });
                const gBot = new THREE.MeshLambertMaterial({ map: this.generateTexture('dirt_altamira') });
                mats = [gSide, gSide, gTop, gBot, gSide, gSide];
                break;
            case 5: // STONE_ANDES
                const sAndes = new THREE.MeshLambertMaterial({ map: this.generateTexture('stone_andes') });
                mats = [sAndes, sAndes, sAndes, sAndes, sAndes, sAndes];
                break;
            case 6: // CALIZA_LACRAIE
                const sCraie = new THREE.MeshLambertMaterial({ map: this.generateTexture('caliza_lacraie') });
                mats = [sCraie, sCraie, sCraie, sCraie, sCraie, sCraie];
                break;
            case 7: // VINE_MALBEC_ALTAMIRA
                const vAltamira = new THREE.MeshLambertMaterial({ map: this.generateTexture('grapevine_malbec_altamira'), transparent: true });
                mats = [vAltamira, vAltamira, vAltamira, vAltamira, vAltamira, vAltamira];
                break;
            case 8: // VINE_LACRAIE
                const vCraie = new THREE.MeshLambertMaterial({ map: this.generateTexture('grapevine_lacraie'), transparent: true });
                mats = [vCraie, vCraie, vCraie, vCraie, vCraie, vCraie];
                break;
            case 9: // VINE_BIODINAMICO
                const vBio = new THREE.MeshLambertMaterial({ map: this.generateTexture('grapevine_biodinamico'), transparent: true });
                mats = [vBio, vBio, vBio, vBio, vBio, vBio];
                break;
            case 10: // VINE_SAN_PABLO
                const vSP = new THREE.MeshLambertMaterial({ map: this.generateTexture('grapevine_san_pablo'), transparent: true });
                mats = [vSP, vSP, vSP, vSP, vSP, vSP];
                break;
            case 11: // HORMIGON_VASJA
                const hVat = new THREE.MeshLambertMaterial({ map: this.generateTexture('hormigon_vasija') });
                mats = [hVat, hVat, hVat, hVat, hVat, hVat];
                break;
            case 12: // BARREL_ROBLE
                const bSide = new THREE.MeshLambertMaterial({ map: this.generateTexture('barrel_roble_side') });
                const bTop = new THREE.MeshLambertMaterial({ map: this.generateTexture('barrel_roble_top') });
                mats = [bSide, bSide, bTop, bTop, bSide, bSide];
                break;
            case 13: // PIEDRA_BODEGA
                const pBodega = new THREE.MeshLambertMaterial({ map: this.generateTexture('piedra_bodega') });
                mats = [pBodega, pBodega, pBodega, pBodega, pBodega, pBodega];
                break;
            case 14: // PRESS_LAGAR
                const pLagar = new THREE.MeshLambertMaterial({ map: this.generateTexture('press_lagar') });
                mats = [pLagar, pLagar, pLagar, pLagar, pLagar, pLagar];
                break;
            case 15: // WATER_DESHIELO
                const water = new THREE.MeshLambertMaterial({ map: this.generateTexture('water_deshielo'), transparent: true, opacity: 0.82 });
                mats = [water, water, water, water, water, water];
                break;
            case 16: // PEDESTAL_INFO
                const ped = new THREE.MeshLambertMaterial({ map: this.generateTexture('pedestal_info') });
                mats = [ped, ped, ped, ped, ped, ped];
                break;
            case 17: // WOOD_PLANKS
                const planks = new THREE.MeshLambertMaterial({ map: this.generateTexture('wood_planks') });
                mats = [planks, planks, planks, planks, planks, planks];
                break;
            case 18: // WHITE_STONE
                const wStone = new THREE.MeshLambertMaterial({ map: this.generateTexture('white_stone') });
                mats = [wStone, wStone, wStone, wStone, wStone, wStone];
                break;
            default:
                const fallback = new THREE.MeshBasicMaterial({ color: 0x999999 });
                mats = [fallback, fallback, fallback, fallback, fallback, fallback];
        }

        this.materials[typeId] = mats;
        return mats;
    }
};

window.TextureManager = TextureManager;
