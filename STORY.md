# WineCraft Story (B8.0)

## Premise
WineCraft is a collaborative open‑world game set in the **Valle de Uco**. Players explore a realistic valley, gather real‑world agricultural ingredients, and ultimately craft wine using historically accurate tools. The experience blends light‑hearted adventure with educational content about viticulture, geology, and Argentinian culture.

## Mentor
**Don César** is the in‑game mentor. He is a **role‑play character**, not a real person, who guides the player through the story via a side‑chat panel. Don César speaks in Spanish, offering clues, historical anecdotes, and short tutorials. All mentor dialogue is scripted and stored on the server; the client only renders the text.

## Story Tasks (≥ 10)
The story is a linear quest chain that teaches core mechanics and leads to the construction of a **crafting bench** and its accompanying winemaking tools. Each task is a distinct step that unlocks the next, and each is accompanied by a short mentor message (sent as a `proto.Chat`).

| Step | Trigger / Action | Mentor Message | Goal Unlock |
|------|------------------|-----------------|-------------|
| 1 | Harvest **6 bunches of grapes** (world.ItemGrapes) | "¡Bien hecho! Has cosechado uvas. Ahora, necesitamos un **banco de arena** para seguir." | Enables building a SandBank (world.SandBank) |
| 2 | Place a **SandBank** on the object layer (world.SandBank) | "¡Has construido el banco de arena! Con él podrás mezclar los ingredientes básicos." | Unlocks **Plank** craft recipe |
| 3 | Craft **4 planks** (`world.ItemPlank`) using the crafting panel | "¡Excelente! Las tablas son la base de cualquier estructura. Ahora construyamos un **banco de trabajo**." | Enables building a Workbench (world.Workbench) |
| 4 | Build a **Workbench** (world.Workbench) in the valley | "¡El banco de trabajo está listo! Con él podrás ensamblar herramientas más complejas." | Unlocks **Stone Wall** recipe |
| 5 | Craft **2 stone walls** (`world.ItemStoneWall`) | "Las paredes de piedra mantendrán tu bodega estable. Construye la **bodega** para proteger la cosecha." | Enables building a Cellar (world.Cellar) |
| 6 | Build a **Cellar** (world.Cellar) | "¡Tu bodega está en pie! Ahora almacena tus uvas y prepáralas para el proceso de fermentación." | Unlocks **Press** recipe |
| 7 | Craft a **Wine Press** (`world.ItemPress`) | "La prensa es esencial para extraer el jugo. Colócala en la bodega para iniciar la fermentación." | Enables building a Press (world.Press) |
| 8 | Place the **Press** in the Cellar | "¡La prensa está lista! Ahora vamos a fermentar el mosto." | Unlocks **Fermentation Barrel** recipe |
| 9 | Craft a **Fermentation Barrel** (`world.ItemBarrel`) | "El barril permite que el jugo se convierta en vino. Necesitarás **Levadura** que ya tienes en tu inventario." | Enables building a Barrel (world.Barrel) |
| 10 | Place the **Barrel** and **Start Fermentation** (interact with the barrel) | "¡Felicidades! Has completado la primera fase de la elaboración del vino. Próximamente, aprenderás a embotellar y vender tu producción." | Marks story completion; unlocks **Market** UI for selling wine |

## Sources & Fact‑Checking
All in‑game items, recipes, and historical notes are based on publicly available resources:
- Argentine viticulture basics – *Instituto Nacional de Vitivinicultura* (2024) – https://invit.org
- Traditional winemaking tools – *Wine Spectator* article “Ancient Winemaking Techniques” (2023) – https://wine...spectator.com/ancient-tools
- Geography of Valle de Uco – *Wikipedia* entry (accessed 2026‑09‑30) – https://en.wikipedia.org/wiki/Valle_de_Uco
- Spanish terminology – *Real Academia Española* (2022) – https://rae.es

## Review Checklist (for Franco)
- [ ] Story premise aligns with product vision.
- [ ] Mentor tone appropriate and consistent.
- [ ] ≥ 10 tasks, each clearly linked to a game mechanic.
- [ ] All fact sources cited and verifiable.
- [ ] No references to real‑world persons; mentor is a fictional role.
- [ ] Tasks progress naturally toward the crafting bench and winemaking tools.

Once Franco approves, the story will be integrated into the server quest engine (B8.2) and the chat panel (B8.1).