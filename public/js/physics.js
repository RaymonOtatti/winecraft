// Physics Engine: Movimiento fluido, teclas duales (WASD + Flechas) y auto-escalón (Auto-Step)

class PhysicsEngine {
    constructor(world) {
        this.world = world;
        this.gravity = -24.0;
        this.playerHeight = 1.75;
        this.playerRadius = 0.32;
        this.jumpVelocity = 8.6;
        this.walkSpeed = 6.2;
        this.runSpeed = 10.0;
        this.stepHeight = 1.05; // Altura máxima de auto-escalón (subir 1 bloque sin saltar)
    }

    update(player, keys, delta) {
        // Movimiento dual: WASD + Teclas de Flechas
        const moveVec = new THREE.Vector3();
        if (keys['KeyW'] || keys['ArrowUp']) moveVec.z -= 1;
        if (keys['KeyS'] || keys['ArrowDown']) moveVec.z += 1;
        if (keys['KeyA'] || keys['ArrowLeft']) moveVec.x -= 1;
        if (keys['KeyD'] || keys['ArrowRight']) moveVec.x += 1;

        if (moveVec.lengthSq() > 0) {
            moveVec.normalize();
        }

        const isRunning = keys['ShiftLeft'] || keys['ShiftRight'];
        const targetSpeed = isRunning ? this.runSpeed : this.walkSpeed;

        // Transformar vector según la rotación horizontal de la cámara
        const cameraEuler = new THREE.Euler().setFromQuaternion(player.camera.quaternion, 'YXZ');
        const forward = new THREE.Vector3(-Math.sin(cameraEuler.y), 0, -Math.cos(cameraEuler.y));
        const right = new THREE.Vector3(Math.cos(cameraEuler.y), 0, -Math.sin(cameraEuler.y));

        const targetVelX = (right.x * moveVec.x + forward.x * moveVec.z) * targetSpeed;
        const targetVelZ = (right.z * moveVec.x + forward.z * moveVec.z) * targetSpeed;

        // Inercia y aceleración suave
        const accel = player.onGround ? 16 : 8;
        player.velocity.x += (targetVelX - player.velocity.x) * Math.min(1.0, delta * accel);
        player.velocity.z += (targetVelZ - player.velocity.z) * Math.min(1.0, delta * accel);

        // Salto con barra espaciadora
        if (keys['Space'] && player.onGround) {
            player.velocity.y = this.jumpVelocity;
            player.onGround = false;
            window.SoundSystem.playStep(false);
        }

        // Gravedad
        player.velocity.y += this.gravity * delta;
        if (player.velocity.y < -30) player.velocity.y = -30;

        // Desplazamiento horizontal con Auto-Step (subir bloques de 1 de altura sin tener que saltar)
        this.moveHorizontalWithStep(player, player.velocity.x * delta, player.velocity.z * delta);

        // Desplazamiento vertical (gravedad / salto)
        this.moveVertical(player, player.velocity.y * delta);

        // Sonido rítmico de pasos
        const horizontalSpeed = Math.hypot(player.velocity.x, player.velocity.z);
        if (player.onGround && horizontalSpeed > 0.8) {
            player.stepTimer = (player.stepTimer || 0) + delta * (isRunning ? 1.7 : 1.1);
            if (player.stepTimer > 0.38) {
                player.stepTimer = 0;
                const blockBelow = this.world.getBlock(Math.floor(player.position.x), Math.floor(player.position.y - 0.1), Math.floor(player.position.z));
                const isStone = [5, 6, 11, 13, 16, 18].includes(blockBelow);
                window.SoundSystem.playStep(isStone);
            }
        }

        // Sincronizar posición de cámara (altura de los ojos)
        player.camera.position.set(
            player.position.x,
            player.position.y + this.playerHeight * 0.85,
            player.position.z
        );
    }

    moveHorizontalWithStep(player, dx, dz) {
        if (dx === 0 && dz === 0) return;

        // Intentar mover directamente
        const originalY = player.position.y;
        const collidedX = this.tryMoveAxis(player, 'x', dx);
        const collidedZ = this.tryMoveAxis(player, 'z', dz);

        // Si colisionó en el suelo, intentar auto-escalón (subir hasta 1 bloque)
        if ((collidedX || collidedZ) && player.onGround) {
            // Verificar si elevar la posición permite pasar el escalón
            const testY = originalY + this.stepHeight;
            player.position.y = testY;

            // Probar si a esa altura hay espacio
            if (!this.checkCollision(player)) {
                // Mover horizontalmente arriba del escalón
                this.tryMoveAxis(player, 'x', dx);
                this.tryMoveAxis(player, 'z', dz);

                // Bajar suavemente hasta tocar el suelo del escalón
                this.moveVertical(player, -this.stepHeight);
            } else {
                // No pudo escalar, restaurar altura
                player.position.y = originalY;
            }
        }
    }

    tryMoveAxis(player, axis, amount) {
        if (amount === 0) return false;
        player.position[axis] += amount;

        if (this.checkCollision(player)) {
            // Revertir y frenar en este eje
            player.position[axis] -= amount;
            player.velocity[axis] = 0;
            return true;
        }
        return false;
    }

    moveVertical(player, amount) {
        if (amount === 0) return;
        player.position.y += amount;

        const pMinX = player.position.x - this.playerRadius;
        const pMaxX = player.position.x + this.playerRadius;
        const pMinY = player.position.y;
        const pMaxY = player.position.y + this.playerHeight;
        const pMinZ = player.position.z - this.playerRadius;
        const pMaxZ = player.position.z + this.playerRadius;

        const minBX = Math.floor(pMinX);
        const maxBX = Math.floor(pMaxX);
        const minBY = Math.floor(pMinY);
        const maxBY = Math.floor(pMaxY);
        const minBZ = Math.floor(pMinZ);
        const maxBZ = Math.floor(pMaxZ);

        for (let bx = minBX; bx <= maxBX; bx++) {
            for (let by = minBY; by <= maxBY; by++) {
                for (let bz = minBZ; bz <= maxBZ; bz++) {
                    const blockId = this.world.getBlock(bx, by, bz);
                    if (blockId !== 0 && Blocks.get(blockId).solid) {
                        if (amount > 0) {
                            // Chocó con el techo
                            player.position.y = by - this.playerHeight - 0.001;
                            player.velocity.y = 0;
                        } else {
                            // Aterrizó en el suelo
                            player.position.y = by + 1;
                            player.velocity.y = 0;
                            player.onGround = true;
                        }
                        return;
                    }
                }
            }
        }

        if (amount < 0) {
            player.onGround = false;
        }
    }

    checkCollision(player) {
        const pMinX = player.position.x - this.playerRadius;
        const pMaxX = player.position.x + this.playerRadius;
        const pMinY = player.position.y;
        const pMaxY = player.position.y + this.playerHeight;
        const pMinZ = player.position.z - this.playerRadius;
        const pMaxZ = player.position.z + this.playerRadius;

        const minBX = Math.floor(pMinX);
        const maxBX = Math.floor(pMaxX);
        const minBY = Math.floor(pMinY);
        const maxBY = Math.floor(pMaxY);
        const minBZ = Math.floor(pMinZ);
        const maxBZ = Math.floor(pMaxZ);

        for (let bx = minBX; bx <= maxBX; bx++) {
            for (let by = minBY; by <= maxBY; by++) {
                for (let bz = minBZ; bz <= maxBZ; bz++) {
                    const blockId = this.world.getBlock(bx, by, bz);
                    if (blockId !== 0 && Blocks.get(blockId).solid) {
                        return true;
                    }
                }
            }
        }
        return false;
    }

    intersectsPlayer(player, bx, by, bz) {
        const pMinX = player.position.x - this.playerRadius;
        const pMaxX = player.position.x + this.playerRadius;
        const pMinY = player.position.y;
        const pMaxY = player.position.y + this.playerHeight;
        const pMinZ = player.position.z - this.playerRadius;
        const pMaxZ = player.position.z + this.playerRadius;

        return (
            bx < pMaxX && (bx + 1) > pMinX &&
            by < pMaxY && (by + 1) > pMinY &&
            bz < pMaxZ && (bz + 1) > pMinZ
        );
    }

    raycast(origin, direction, maxDistance = 6.8) {
        let x = Math.floor(origin.x);
        let y = Math.floor(origin.y);
        let z = Math.floor(origin.z);

        const dx = direction.x;
        const dy = direction.y;
        const dz = direction.z;

        const stepX = dx > 0 ? 1 : -1;
        const stepY = dy > 0 ? 1 : -1;
        const stepZ = dz > 0 ? 1 : -1;

        const tDeltaX = dx !== 0 ? Math.abs(1 / dx) : Infinity;
        const tDeltaY = dy !== 0 ? Math.abs(1 / dy) : Infinity;
        const tDeltaZ = dz !== 0 ? Math.abs(1 / dz) : Infinity;

        let tMaxX = dx > 0 ? (x + 1 - origin.x) * tDeltaX : (origin.x - x) * tDeltaX;
        let tMaxY = dy > 0 ? (y + 1 - origin.y) * tDeltaY : (origin.y - y) * tDeltaY;
        let tMaxZ = dz > 0 ? (z + 1 - origin.z) * tDeltaZ : (origin.z - z) * tDeltaZ;

        let distance = 0;
        let normal = { x: 0, y: 0, z: 0 };

        while (distance <= maxDistance) {
            const blockId = this.world.getBlock(x, y, z);
            if (blockId !== 0 && Blocks.get(blockId).solid) {
                return {
                    hit: true,
                    blockId,
                    position: { x, y, z },
                    normal: { ...normal },
                    distance
                };
            }

            if (tMaxX < tMaxY) {
                if (tMaxX < tMaxZ) {
                    distance = tMaxX;
                    x += stepX;
                    tMaxX += tDeltaX;
                    normal = { x: -stepX, y: 0, z: 0 };
                } else {
                    distance = tMaxZ;
                    z += stepZ;
                    tMaxZ += tDeltaZ;
                    normal = { x: 0, y: 0, z: -stepZ };
                }
            } else {
                if (tMaxY < tMaxZ) {
                    distance = tMaxY;
                    y += stepY;
                    tMaxY += tDeltaY;
                    normal = { x: 0, y: -stepY, z: 0 };
                } else {
                    distance = tMaxZ;
                    z += stepZ;
                    tMaxZ += tDeltaZ;
                    normal = { x: 0, y: 0, z: -stepZ };
                }
            }
        }

        return { hit: false };
    }
}

window.PhysicsEngine = PhysicsEngine;
