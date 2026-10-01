// Codex del Sommelier: Las 4 Bodegas de Culto del Valle de Uco, Mendoza

class WineCodex {
    constructor() {
        this.isOpen = false;
        this.activeTab = 'zuccardi';
    }

    toggle() {
        if (this.isOpen) this.close();
        else this.open();
    }

    open() {
        this.isOpen = true;
        const modal = document.getElementById('codex-modal');
        if (modal) {
            modal.style.display = 'flex';
            this.renderTab(this.activeTab);
            if (document.exitPointerLock) document.exitPointerLock();
        }
    }

    close() {
        this.isOpen = false;
        const modal = document.getElementById('codex-modal');
        if (modal) modal.style.display = 'none';
    }

    setTab(tab) {
        this.activeTab = tab;
        this.renderTab(tab);
    }

    renderTab(tab) {
        const body = document.getElementById('codex-body');
        if (!body) return;

        document.querySelectorAll('.codex-tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tab);
        });

        if (tab === 'zuccardi') {
            body.innerHTML = `
                <div class="codex-section">
                    <h2>🏔️ Zuccardi Valle de Uco — Finca Piedra Infinita</h2>
                    <p class="codex-lead"><strong>Ubicación:</strong> Paraje Altamira, San Carlos (Sur del Valle de Uco, 1.100 msnm)<br>
                    <strong>Líder y Enólogo:</strong> Sebastián Zuccardi (Familia Zuccardi)</p>

                    <div class="codex-card">
                        <h3>🌍 El Terroir de Paraje Altamira</h3>
                        <p>Nacido en el cono aluvial del Río Tunuyán, este suelo es uno de los más extremos del planeta. Las capas subterráneas están repletas de grandes cantos rodados andinos cubiertos de una densa costra blanca de carbonato de calcio (calcáreo). Para plantar el viñedo, se debieron extraer más de 1.000 camiones de piedras, dando origen al nombre <em>Piedra Infinita</em>.</p>
                    </div>

                    <div class="codex-card">
                        <h3>🏛️ Arquitectura & Elaboración</h3>
                        <p>Inaugurada en 2016 y elegida múltiples veces como la <strong>Mejor Bodega del Mundo</strong> (World's Best Vineyards), su estructura fue esculpida con piedra del lugar, arena y agua del río. No utiliza madera nueva ni acero inoxidable para sus vinos de finca: fermenta y cría exclusivamente en <strong>vasijas cónicas y piletas de hormigón sin epoxi</strong>, permitiendo que la tiza y la frescura del lugar se expresen sin interferencias.</p>
                    </div>

                    <div class="codex-card">
                        <h3>⭐ Vinos Consagrados (100 Puntos Parker)</h3>
                        <p><em>Finca Piedra Infinita</em>, <em>Gravascal</em> y <em>Supercal</em> han alcanzado puntuaciones perfectas de 100 puntos en The Wine Advocate, situando al Malbec de Altamira en la cúspide mundial.</p>
                    </div>
                </div>
            `;
        } else if (tab === 'perse') {
            body.innerHTML = `
                <div class="codex-section">
                    <h2>🕊️ PerSe Vines — El Terroir Místico de la Alta Montaña</h2>
                    <p class="codex-lead"><strong>Ubicación:</strong> Gualtallary Alto, Tupungato (Norte del Valle de Uco, 1.450 - 1.500 msnm)<br>
                    <strong>Fundadores y Enólogos:</strong> Edgardo "Edy" Del Pópolo & David Bonomi</p>

                    <div class="codex-card">
                        <h3>🌿 La Búsqueda de "PerSe"</h3>
                        <p>Fundada en 2012 por dos eminencias de la viticultura argentina, PerSe significa "por sí mismo" en latín. Su objetivo es elaborar vinos de mínima intervención en micro-parcelas irrepetibles de altísima ladera, situadas junto al <strong>Monasterio del Cristo Orante</strong>.</p>
                    </div>

                    <div class="codex-card">
                        <h3>⚪ Suelo "La Craie" (Tiza Pura)</h3>
                        <p>El viñedo se asienta sobre laderas empinadas orientadas al sur, donde la erosión ha dejado al descubierto bancos de caliza pura blanca y arena volcánica. Las raíces de las vides penetran directamente en la roca calcárea, dando vinos con una tensión mineral salina, taninos de tiza y una elegancia que recuerda a los más sublimes Grands Crus de Borgoña.</p>
                    </div>

                    <div class="codex-card">
                        <h3>⭐ Vinos de Culto</h3>
                        <p><em>La Craie</em> (Malbec y Cabernet Franc cofermentados, 100 Puntos Parker), <em>Iubileus</em>, <em>Inseparable</em> y <em>Uní</em>. Producciones minúsculas de coleccionista.</p>
                    </div>
                </div>
            `;
        } else if (tab === 'estocada') {
            body.innerHTML = `
                <div class="codex-section">
                    <h2>🌱 Sitio La Estocada — Biodinámica & Cal Restaurante</h2>
                    <p class="codex-lead"><strong>Ubicación:</strong> Gualtallary, Tupungato (Valle de Uco, 1.350 msnm)<br>
                    <strong>Creador y Vigneron:</strong> Matías Michelini y familia</p>

                    <div class="codex-card">
                        <h3>🐝 Filosofía Ecológica y Biodinámica</h3>
                        <p>Una finca artesanal de 4 hectáreas concebida como un ecosistema vivo autosustentable. Las vides no están solas: conviven con colmenas de abejas, huertas orgánicas, árboles frutales, caballos y flora nativa andina. Se aplican preparados biodinámicos y se trabaja siguiendo los ritmos lunares.</p>
                    </div>

                    <div class="codex-card">
                        <h3>🍽️ Cal Restaurante (Guía MICHELIN)</h3>
                        <p>Dentro de la finca opera <strong>Cal</strong>, un restaurante íntimo con estrella y mención en la Guía Michelin donde se sirve un menú de pasos basado exclusivamente en la cosecha de la propia huerta y animales de la finca, maridado con vinos fermentados en huevos de cemento y ánforas.</p>
                    </div>

                    <div class="codex-card">
                        <h3>⭐ Vinos Exclusivos</h3>
                        <p>Línea <em>Camino de los Europeos</em> (Cabernet Franc, Malbec y Chardonnay de extrema frescura), reservados para el club privado de la bodega y visitas a la finca.</p>
                    </div>
                </div>
            `;
        } else if (tab === 'cru') {
            body.innerHTML = `
                <div class="codex-section">
                    <h2>❄️ Cru de Montaña — El Viento y la Altura Glaciar</h2>
                    <p class="codex-lead"><strong>Ubicación:</strong> San Pablo, Tunuyán (1.400 - 1.470 msnm)<br>
                    <strong>Creadores:</strong> Rodrigo Calderón (Sommelier) & Manolo Pelegrina</p>

                    <div class="codex-card">
                        <h3>🌬️ La IG San Pablo: Clima Extremo</h3>
                        <p>San Pablo es uno de los terroirs más fríos y extremos de toda América del Sur. Ubicado justo bajo las nieves eternas del Cordón del Plata, recibe una corriente constante de vientos catabáticos helados que bajan de la cordillera. Las noches son gélidas incluso en pleno verano.</p>
                    </div>

                    <div class="codex-card">
                        <h3>🍇 Visión de Sommelier</h3>
                        <p>Rodrigo Calderón concibe sus vinos desde la perspectiva del sommelier: busca acidez natural vertical, bajísimo grado alcohólico, taninos firmes y aromas a hierbas silvestres (jarilla, tomillo, mentol andino). Crianza en barricas de estilo borgoñón y reposo en piletas de concreto.</p>
                    </div>

                    <div class="codex-card">
                        <h3>⭐ Etiquetas Insignia</h3>
                        <p><em>Cru de San Pablo</em> (Cabernet Franc y Malbec), <em>Cru de Gualtallary</em> y <em>Días Perfectos</em>.</p>
                    </div>
                </div>
            `;
        } else if (tab === 'controls') {
            body.innerHTML = `
                <div class="codex-section">
                    <h2>🎮 Guía de Movimiento y Crafteo</h2>
                    <div class="codex-card">
                        <h3>🕹️ Controles de Movimiento</h3>
                        <ul>
                            <li><strong>Moverse:</strong> Teclas <strong>W, A, S, D</strong> o las <strong>Flechas del Teclado (↑, ↓, ←, →)</strong>.</li>
                            <li><strong>Auto-Escalón (Auto-Step):</strong> ¡Ya no necesitas saltar en cada bloque! El personaje sube automáticamente los desniveles de 1 bloque de altura al caminar.</li>
                            <li><strong>Saltar:</strong> Barra Espaciadora.</li>
                            <li><strong>Correr:</strong> Mantener presionado <strong>Shift</strong>.</li>
                            <li><strong>Mirar en 360°:</strong> Mueve el ratón (haz clic en la pantalla para activar el puntero).</li>
                            <li><strong>Selección de Bloques:</strong> Teclas del 1 al 9 o rueda del ratón.</li>
                        </ul>
                    </div>

                    <div class="codex-card">
                        <h3>🍇 Cómo Elaborar Vino en WineCraft</h3>
                        <ol>
                            <li><strong>Cosechar:</strong> Clic izquierdo en las vides de cada bodega para recolectar sus racimos específicos.</li>
                            <li><strong>Prensado:</strong> Clic derecho en el <strong>Lagar / Prensa de Madera</strong> con 2 o más racimos para obtener Mosto virgen.</li>
                            <li><strong>Crianza:</strong> Clic derecho con el mosto en una <strong>Vasija de Hormigón</strong> o en una <strong>Barrica de Roble</strong>. Espera 7 segundos y haz clic nuevamente para embotellar y abrir la <strong>Ficha de Degustación Oficial (100 pts)</strong>.</li>
                            <li><strong>Atriles de Información:</strong> Haz clic derecho en los pedestales dorados ubicados en cada bodega para leer la historia completa de sus dueños y enólogos.</li>
                        </ol>
                    </div>
                </div>
            `;
        }
    }
}

window.WineCodex = new WineCodex();
