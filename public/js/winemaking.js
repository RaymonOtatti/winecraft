// Winemaking Mechanics, Concrete Vats & Iconic Mendoza Wines

class WinemakingSystem {
    constructor() {
        this.inventory = {
            // Uvas de las 4 bodegas
            'uva_malbec_zuccardi': { name: 'Racimo Malbec Altamira (Zuccardi)', bodega: 'Zuccardi', count: 0, icon: '🍇' },
            'uva_lacraie': { name: 'Racimo La Craie (PerSe)', bodega: 'PerSe', count: 0, icon: '🍇' },
            'uva_estocada': { name: 'Racimo Biodinámico (La Estocada)', bodega: 'La Estocada', count: 0, icon: '🍇' },
            'uva_san_pablo': { name: 'Racimo Cabernet Franc (Cru de Montaña)', bodega: 'Cru de Montaña', count: 0, icon: '🍇' },

            // Mostos
            'mosto_altamira': { name: 'Mosto Malbec Piedra Infinita', count: 0, icon: '🧃' },
            'mosto_lacraie': { name: 'Mosto Calcáreo La Craie', count: 0, icon: '🧃' },
            'mosto_estocada': { name: 'Mosto Biodinámico Camino de los Europeos', count: 0, icon: '🧃' },
            'mosto_sanpablo': { name: 'Mosto de Altura San Pablo', count: 0, icon: '🧃' },

            // Vinos Iconos de Mendoza
            'vino_piedra_infinita': { name: 'Zuccardi Finca Piedra Infinita 2026', bodega: 'Zuccardi', count: 0, icon: '🍷' },
            'vino_per_se_lacraie': { name: 'PerSe La Craie 2026', bodega: 'PerSe', count: 0, icon: '🍷' },
            'vino_camino_europeos': { name: 'Sitio La Estocada "Camino de los Europeos" 2026', bodega: 'La Estocada', count: 0, icon: '🍷' },
            'vino_cru_san_pablo': { name: 'Cru de Montaña "Cru de San Pablo" 2026', bodega: 'Cru de Montaña', count: 0, icon: '🍷' }
        };

        this.activeVats = new Map();
    }

    addGrape(grapeId, count = 2) {
        if (this.inventory[grapeId]) {
            this.inventory[grapeId].count += count;
            window.SoundSystem.playHarvest();
            this.showNotification(`+${count} ${this.inventory[grapeId].name} cosechado!`);
            this.updateInventoryUI();
            return true;
        }
        return false;
    }

    interactPress(x, y, z) {
        window.SoundSystem.playPress();

        const grapeMap = [
            { grape: 'uva_malbec_zuccardi', must: 'mosto_altamira', name: 'Malbec Altamira (Zuccardi)' },
            { grape: 'uva_lacraie', must: 'mosto_lacraie', name: 'La Craie Gualtallary (PerSe)' },
            { grape: 'uva_estocada', must: 'mosto_estocada', name: 'Biodinámico (La Estocada)' },
            { grape: 'uva_san_pablo', must: 'mosto_sanpablo', name: 'Cabernet Franc (Cru de Montaña)' }
        ];

        let pressed = false;
        for (const item of grapeMap) {
            if (this.inventory[item.grape].count >= 2) {
                this.inventory[item.grape].count -= 2;
                this.inventory[item.must].count += 1;
                this.showNotification(`🍇 ¡Prensaste ${item.name}! Obtuviste 1x ${this.inventory[item.must].name}`);
                this.updateInventoryUI();
                pressed = true;
                break;
            }
        }

        if (!pressed) {
            this.showNotification(`⚠️ Necesitas al menos 2 racimos de uva cosechada para prensar mosto.`);
        }
    }

    interactVatOrBarrel(x, y, z, isConcrete = false) {
        const key = `${x},${y},${z}`;
        let vat = this.activeVats.get(key);

        const vatName = isConcrete ? "Vasija de Hormigón" : "Barrica de Roble Francés";

        if (!vat) {
            const recipeMap = [
                { must: 'mosto_altamira', wine: 'vino_piedra_infinita', name: 'Zuccardi Finca Piedra Infinita 2026' },
                { must: 'mosto_lacraie', wine: 'vino_per_se_lacraie', name: 'PerSe La Craie 2026' },
                { must: 'mosto_estocada', wine: 'vino_camino_europeos', name: 'Sitio La Estocada Camino de los Europeos 2026' },
                { must: 'mosto_sanpablo', wine: 'vino_cru_san_pablo', name: 'Cru de Montaña Cru de San Pablo 2026' }
            ];

            let loaded = false;
            for (const item of recipeMap) {
                if (this.inventory[item.must].count >= 1) {
                    this.inventory[item.must].count -= 1;
                    this.activeVats.set(key, {
                        wineTarget: item.wine,
                        wineName: item.name,
                        vessel: vatName,
                        startTime: Date.now(),
                        duration: 7500
                    });
                    window.SoundSystem.playPlace();
                    this.showNotification(`🪵 ¡Cargaste ${vatName} con ${item.name}! Fermentando y criando...`);
                    this.updateInventoryUI();
                    loaded = true;
                    break;
                }
            }

            if (!loaded) {
                this.showNotification(`ℹ️ Esta ${vatName} está vacía. Prensa uvas en el lagar para obtener mosto.`);
            }
        } else {
            const elapsed = Date.now() - vat.startTime;
            if (elapsed >= vat.duration) {
                this.inventory[vat.wineTarget].count += 1;
                this.activeVats.delete(key);
                window.SoundSystem.playToast();
                this.showNotification(`🍷 ¡Vino listo! Embotellaste 1x ${vat.wineName}`);
                this.updateInventoryUI();
                this.showTastingNotesModal(vat.wineTarget);
            } else {
                const remaining = Math.ceil((vat.duration - elapsed) / 1000);
                this.showNotification(`⏳ Crianza en ${vat.vessel}: listo en ${remaining}s...`);
            }
        }
    }

    showPedestalInfo(x, y, z) {
        window.SoundSystem.playPlace();

        // Determinar qué bodega corresponde según coordenadas z y x
        let info = null;

        if (z < -10) {
            if (x < 0) {
                // PerSe
                info = {
                    title: "PerSe Vines",
                    subtitle: "Gualtallary Alto, Tupungato | 1.450 - 1.500 msnm",
                    owners: "Edgardo 'Edy' Del Pópolo & David Bonomi",
                    soil: "Ladera escarpada con caliza pura ('La Craie'), arenas volcánicas y calcáreo blanco.",
                    grapes: "Malbec, Cabernet Franc y Pinot Noir.",
                    philosophy: "Vinos de mínima intervención nacidos junto al Monasterio del Cristo Orante. Elaboración de partidas exclusivas en vasijas de concreto y barricas usadas. 'PerSe' significa por sí mismo: el suelo habla sin maquillaje.",
                    iconic: "La Craie (100 Puntos Parker), Iubileus, Inseparable, Uní."
                };
            } else {
                // Sitio La Estocada
                info = {
                    title: "Sitio La Estocada",
                    subtitle: "Gualtallary, Tupungato | 1.350 msnm",
                    owners: "Matías Michelini y familia",
                    soil: "Suelo calcáreo con arcillas y arenas andinas de gran frescura.",
                    grapes: "Cabernet Franc, Malbec, Chardonnay biodinámico.",
                    philosophy: "Finca familiar de 4 hectáreas con vitivinicultura ecológica y biodinámica. Las vides conviven con animales de granja, huerta orgánica y frutales. Alberga el restaurante 'Cal' (Guía MICHELIN).",
                    iconic: "Línea 'Camino de los Europeos', vinos de club privado y crianza en huevos de hormigón."
                };
            }
        } else if (z >= -10 && z <= 10) {
            // Cru de Montaña
            info = {
                title: "Cru de Montaña",
                subtitle: "San Pablo, Tunuyán | 1.400 - 1.470 msnm",
                owners: "Rodrigo Calderón (Sommelier) & Manolo Pelegrina",
                soil: "Gravas y suelo aluvial de alta montaña, expuesto a los vientos glaciares del Cordón del Plata.",
                grapes: "Cabernet Franc y Malbec.",
                philosophy: "Proyecto de autor con acidez natural crujiente y perfil austero de montaña. Crianzas cuidadas en roble borgoñón y reposo en concreto. Expresión pura de parcelas extremas.",
                iconic: "Cru de San Pablo, Cru de Gualtallary, Días Perfectos."
            };
        } else {
            // Zuccardi
            info = {
                title: "Zuccardi Valle de Uco — Finca Piedra Infinita",
                subtitle: "Paraje Altamira, San Carlos | 1.100 msnm",
                owners: "Sebastián Zuccardi y Familia Zuccardi",
                soil: "Cono aluvial del Río Tunuyán, repleto de cantos rodados cubiertos de cal blanco.",
                grapes: "Malbec de Altamira y Cabernet Franc.",
                philosophy: "'No buscamos vinos perfectos, sino vinos que expresen el lugar'. Elegida #1 World's Best Vineyard. Arquitectura monolítica de piedra y concreto del lugar, vinificación por gravedad en piletas cónicas de hormigón sin epoxi.",
                iconic: "Finca Piedra Infinita (100 Puntos Parker), Supercal, Gravascal."
            };
        }

        const modal = document.getElementById('tasting-modal');
        const content = document.getElementById('tasting-content');
        if (!modal || !content) return;

        content.innerHTML = `
            <div class="tasting-card">
                <div class="tasting-header">
                    <h2>🏰 ${info.title}</h2>
                    <span class="tasting-score">Valle de Uco</span>
                </div>
                <h3>${info.subtitle}</h3>
                <p><strong>👤 Creadores / Dueños:</strong> ${info.owners}</p>
                <p><strong>🌍 Terroir & Suelo:</strong> ${info.soil}</p>
                <p><strong>🍇 Cepas Principales:</strong> ${info.grapes}</p>
                <div class="tasting-box" style="margin: 14px 0;">
                    <strong>📖 Filosofía de Elaboración:</strong>
                    <p>${info.philosophy}</p>
                </div>
                <p><strong>⭐ Vinos Icónicos:</strong> ${info.iconic}</p>
                <button class="tasting-close-btn" style="margin-top: 15px;" onclick="document.getElementById('tasting-modal').style.display='none'">Cerrar Ficha</button>
            </div>
        `;

        modal.style.display = 'flex';
        if (document.exitPointerLock) document.exitPointerLock();
    }

    showTastingNotesModal(wineId) {
        const tastings = {
            'vino_piedra_infinita': {
                title: 'Zuccardi Finca Piedra Infinita Malbec 2026',
                origin: 'Paraje Altamira, San Carlos, Valle de Uco (1.100 msnm)',
                creator: 'Sebastián Zuccardi',
                terroir: 'Suelos aluviales extremos con cantos rodados cubiertos de carbonato de calcio blanco.',
                visual: 'Rojo violáceo brillante de capa profunda y límpida, ribetes azulados.',
                aromas: 'Ciruelas negras frescas, moras de montaña, cáscara de naranja, violetas y un marcado fondo de tiza húmeda y pólvora.',
                palate: 'Textura calcárea única en el paladar, taninos que agarran con finura de tiza, acidez vibrante y final interminable.',
                score: '100 Puntos - The Wine Advocate (Robert Parker)',
                pairing: 'Ojo de bife madurado a las brasas de quebracho, mollejas crujientes al limón y verduras al rescoldo.'
            },
            'vino_per_se_lacraie': {
                title: 'PerSe La Craie 2026',
                origin: 'Gualtallary Alto, Tupungato, Valle de Uco (1.450 msnm)',
                creator: 'Edgardo Del Pópolo & David Bonomi',
                terroir: 'Ladera extrema junto al Monasterio del Cristo Orante; suelo de tiza blanca pura (La Craie).',
                visual: 'Rojo rubí oscuro translúcido con destellos granate.',
                aromas: 'Aromas etéreos a hierbas andinas, jarilla, frutos rojos silvestres, pimienta blanca y polvo de roca.',
                palate: 'Una tensión eléctrica y mineral asombrosa, cuerpo medio pero de una profundidad monumental, taninos sedosos y salinos.',
                score: '100 Puntos - Luis Gutiérrez / Robert Parker',
                pairing: 'Chivito andino estofado al caldero, lomo a la pimienta verde o risotto de hongos de pino.'
            },
            'vino_camino_europeos': {
                title: 'Sitio La Estocada "Camino de los Europeos" 2026',
                origin: 'Gualtallary, Tupungato, Valle de Uco (1.350 msnm)',
                creator: 'Matías Michelini',
                terroir: 'Finca biodinámica de 4 ha; suelo de arena y carbonato de calcio, flora silvestre y biodiversidad.',
                visual: 'Rojo carmesí vivo con tintes rubí brillantes.',
                aromas: 'Pimientos asados dulces, grosellas rojas, flores de montaña, tomillo silvestre y notas minerales vivas.',
                palate: 'Fresco, directo, vibrante y jugoso; taninos tensos y una acidez natural deslumbrante que invita a beber sin cesar.',
                score: '97 Puntos - Guía Descorchados / Cal Restaurante Michelin',
                pairing: 'Platos del restaurante Cal: trucha de montaña curada, pato confitado con puré de topinambur y quesos de campo.'
            },
            'vino_cru_san_pablo': {
                title: 'Cru de Montaña "Cru de San Pablo" 2026',
                origin: 'San Pablo, Tunuyán, Valle de Uco (1.420 msnm)',
                creator: 'Rodrigo Calderón (Sommelier) & Manolo Pelegrina',
                terroir: 'Alta montaña azotada por los vientos glaciares del Cordón del Plata; noches gélidas y suelos pedregosos.',
                visual: 'Púrpura intenso con reflejos violáceos oscuros.',
                aromas: 'Frutos negros salvajes, moras, notas de eucalipto andino, grafito y hierbas autóctonas.',
                palate: 'Filoso y vertical, acidez punzante y gran concentración de fruta de clima frío; taninos finos y firmes.',
                score: '96 Puntos - Tim Atkin MW',
                pairing: 'Entraña a la leña, empanadas mendocinas con abundante cebolla y comino, charqui y quesos curados.'
            }
        };

        const data = tastings[wineId];
        if (!data) return;

        const modal = document.getElementById('tasting-modal');
        const content = document.getElementById('tasting-content');
        if (!modal || !content) return;

        content.innerHTML = `
            <div class="tasting-card">
                <div class="tasting-header">
                    <h2>🍷 Ficha de Degustación Oficial</h2>
                    <span class="tasting-score">${data.score}</span>
                </div>
                <h3>${data.title}</h3>
                <p class="tasting-origin"><strong>📍 Origen:</strong> ${data.origin}</p>
                <p><strong>👤 Enólogo / Creador:</strong> ${data.creator}</p>
                <p class="tasting-terroir"><strong>🌍 Terruño:</strong> ${data.terroir}</p>
                <div class="tasting-grid">
                    <div class="tasting-box">
                        <strong>👁️ Fase Visual</strong>
                        <p>${data.visual}</p>
                    </div>
                    <div class="tasting-box">
                        <strong>👃 Fase Olfativa</strong>
                        <p>${data.aromas}</p>
                    </div>
                    <div class="tasting-box">
                        <strong>👅 Fase Gustativa</strong>
                        <p>${data.palate}</p>
                    </div>
                    <div class="tasting-box">
                        <strong>🍽️ Maridaje de Mendoza</strong>
                        <p>${data.pairing}</p>
                    </div>
                </div>
                <button class="tasting-close-btn" onclick="document.getElementById('tasting-modal').style.display='none'">¡Salud! (Cerrar Ficha)</button>
            </div>
        `;

        modal.style.display = 'flex';
        if (document.exitPointerLock) document.exitPointerLock();
    }

    showNotification(text) {
        const notif = document.getElementById('notification');
        if (!notif) return;
        notif.textContent = text;
        notif.style.opacity = '1';
        clearTimeout(this.notifTimeout);
        this.notifTimeout = setTimeout(() => {
            notif.style.opacity = '0';
        }, 3500);
    }

    updateInventoryUI() {
        const countSpan = document.getElementById('harvest-summary');
        if (!countSpan) return;

        const totalGrapes = this.inventory.uva_malbec_zuccardi.count +
            this.inventory.uva_lacraie.count +
            this.inventory.uva_estocada.count +
            this.inventory.uva_san_pablo.count;

        const totalWines = this.inventory.vino_piedra_infinita.count +
            this.inventory.vino_per_se_lacraie.count +
            this.inventory.vino_camino_europeos.count +
            this.inventory.vino_cru_san_pablo.count;

        countSpan.innerHTML = `🍇 Racimos de Uco: <b>${totalGrapes}</b> | 🍷 Botellas Icono: <b>${totalWines}</b>`;
    }
}

window.WinemakingSystem = new WinemakingSystem();
