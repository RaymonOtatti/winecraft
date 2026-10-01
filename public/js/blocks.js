// Blocks Catalog and Definitions for WineCraft: Mendoza Edition (Valle de Uco)

const Blocks = {
    AIR: 0,
    DIRT_ALTAMIRA: 1,      // Suelo aluvial pedregoso con carbonato de calcio
    DIRT_GUALTALLARY: 2,   // Suelo arenoso con caliza pura (La Craie)
    DIRT_SAN_PABLO: 3,     // Suelo de altura fría con jarilla y gravas
    GRASS_UCO: 4,          // Pasto andino del Valle de Uco
    STONE_ANDES: 5,        // Piedra de la Cordillera de los Andes
    CALIZA_LACRAIE: 6,     // Caliza blanca tiza de Gualtallary (PerSe)
    VINE_MALBEC_ALTAMIRA: 7, // Malbec de Zuccardi (Piedra Infinita)
    VINE_LACRAIE: 8,       // Malbec / Cabernet Franc de PerSe (Gualtallary Alto)
    VINE_BIODINAMICO: 9,   // Cabernet Franc / Chardonnay de Sitio La Estocada
    VINE_SAN_PABLO: 10,    // Cabernet Franc de Cru de Montaña
    HORMIGON_VASJA: 11,    // Vasija / Pileta cónica de hormigón sin epoxi (Zuccardi / Michelini)
    BARREL_ROBLE: 12,      // Barrica de roble francés usada
    PIEDRA_BODEGA: 13,     // Piedra autóctona labrada de bodega
    PRESS_LAGAR: 14,       // Prensa / Lagar artesanal
    WATER_DESHIELO: 15,    // Acequia de agua de deshielo del Río Tunuyán
    PEDESTAL_INFO: 16,     // Pedestal informativo del dueño y enólogo
    WOOD_PLANKS: 17,       // Tablas de madera para cata y techos
    WHITE_STONE: 18,       // Muro blanco / Cal restaurante

    data: {
        1: {
            name: 'Terruño Altamira',
            bodega: 'Zuccardi',
            description: 'Suelo aluvial pedregoso con cantos rodados cubiertos de cal blanco.',
            solid: true,
            drop: { id: 'item_dirt_altamira', count: 1 }
        },
        2: {
            name: 'Arena Calcárea Gualtallary',
            bodega: 'PerSe / La Estocada',
            description: 'Suelo de alta montaña con arena volcánica y densa tiza calcárea.',
            solid: true,
            drop: { id: 'item_dirt_gualtallary', count: 1 }
        },
        3: {
            name: 'Terruño Glaciar San Pablo',
            bodega: 'Cru de Montaña',
            description: 'Suelo de altura extrema (1450 msnm) batido por vientos del Cordón del Plata.',
            solid: true,
            drop: { id: 'item_dirt_san_pablo', count: 1 }
        },
        4: {
            name: 'Pasto del Valle de Uco',
            bodega: 'Universal',
            description: 'Vegetación autóctona de altura con tomillo silvestre y jarilla.',
            solid: true,
            drop: { id: 'item_dirt_altamira', count: 1 }
        },
        5: {
            name: 'Piedra de los Andes',
            bodega: 'Cordillera',
            description: 'Roca volcánica y andesita milenaria del Cordón del Plata.',
            solid: true,
            drop: { id: 'item_stone_andes', count: 1 }
        },
        6: {
            name: 'Caliza Blanca "La Craie"',
            bodega: 'PerSe',
            description: 'Tiza calcárea pura descubierta por Edy Del Pópolo y David Bonomi.',
            solid: true,
            drop: { id: 'item_caliza_lacraie', count: 1 }
        },
        7: {
            name: 'Vid Malbec Altamira (Zuccardi)',
            bodega: 'Zuccardi Piedra Infinita',
            description: 'Sebastián Zuccardi: taninos de tiza, ciruela negra y pureza mineral.',
            solid: true,
            drop: { id: 'uva_malbec_zuccardi', name: 'Racimo Malbec Altamira', count: 2, isGrape: true }
        },
        8: {
            name: 'Vid La Craie (PerSe)',
            bodega: 'PerSe Vines',
            description: 'Edy Del Pópolo & David Bonomi: Malbec y Cabernet Franc de 1500 msnm.',
            solid: true,
            drop: { id: 'uva_lacraie', name: 'Racimo La Craie Gualtallary', count: 2, isGrape: true }
        },
        9: {
            name: 'Vid Biodinámica (La Estocada)',
            bodega: 'Sitio La Estocada',
            description: 'Matías Michelini: viticultura natural, orgánica y biodinámica.',
            solid: true,
            drop: { id: 'uva_estocada', name: 'Racimo Biodinámico La Estocada', count: 2, isGrape: true }
        },
        10: {
            name: 'Vid Cru de San Pablo',
            bodega: 'Cru de Montaña',
            description: 'Rodrigo Calderón & Manolo Pelegrina: acidez filosa y fruta de montaña.',
            solid: true,
            drop: { id: 'uva_san_pablo', name: 'Racimo Cabernet Franc San Pablo', count: 2, isGrape: true }
        },
        11: {
            name: 'Pileta / Vasija de Hormigón',
            bodega: 'Zuccardi & La Estocada',
            description: 'Vasija de concreto sin epoxi que respeta la expresión pura del terroir.',
            solid: true,
            interactive: true,
            drop: { id: 'item_hormigon', count: 1 }
        },
        12: {
            name: 'Barrica de Roble Francés',
            bodega: 'Universal',
            description: 'Aporta microoxigenación y elegancia sin enmascarar la fruta.',
            solid: true,
            interactive: true,
            drop: { id: 'item_barrel', count: 1 }
        },
        13: {
            name: 'Piedra de Piedra Infinita',
            bodega: 'Zuccardi',
            description: 'Muro construido con las miles de piedras extraídas del viñedo.',
            solid: true,
            drop: { id: 'item_piedra_bodega', count: 1 }
        },
        14: {
            name: 'Prensa / Lagar Artesanal',
            bodega: 'Universal',
            description: 'Prensa de madera para extraer el mosto virgen de la cosecha.',
            solid: true,
            interactive: true,
            drop: { id: 'item_press', count: 1 }
        },
        15: {
            name: 'Acequia del Río Tunuyán',
            bodega: 'Universal',
            description: 'Canal de riego con agua pura de deshielo andino.',
            solid: false,
            drop: null
        },
        16: {
            name: 'Atril de Información de Bodega',
            bodega: 'Universal',
            description: 'Placa con la historia, dueños, altitud y filosofía de la bodega.',
            solid: true,
            interactive: true,
            drop: null
        },
        17: {
            name: 'Madera de Galería de Cata',
            bodega: 'Universal',
            description: 'Tablones de roble para las salas de degustación y terrazas.',
            solid: true,
            drop: { id: 'item_wood_planks', count: 1 }
        },
        18: {
            name: 'Muro Blanco Encalado',
            bodega: 'Sitio La Estocada',
            description: 'Paredes del restaurante Cal y el monasterio.',
            solid: true,
            drop: { id: 'item_white_stone', count: 1 }
        }
    },

    get(id) {
        return this.data[id] || { name: 'Desconocido', solid: true, drop: null };
    }
};

window.Blocks = Blocks;
