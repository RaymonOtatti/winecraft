// Web Audio API Synthesizer: Efectos y Música procedimental para WineCraft

class SoundSystem {
    constructor() {
        this.ctx = null;
        this.musicPlaying = false;
        this.muted = false;
        this.musicTimer = null;
    }

    init() {
        if (!this.ctx) {
            const AudioCtx = window.AudioContext || window.webkitAudioContext;
            this.ctx = new AudioCtx();
        }
        if (this.ctx.state === 'suspended') {
            this.ctx.resume();
        }
    }

    playStep(isStone = false) {
        if (this.muted) return;
        this.init();
        const now = this.ctx.currentTime;
        const osc = this.ctx.createOscillator();
        const gain = this.ctx.createGain();

        osc.type = isStone ? 'triangle' : 'sine';
        osc.frequency.setValueAtTime(isStone ? 120 : 80, now);
        osc.frequency.exponentialRampToValueAtTime(30, now + 0.08);

        gain.gain.setValueAtTime(0.08, now);
        gain.gain.exponentialRampToValueAtTime(0.001, now + 0.08);

        osc.connect(gain);
        gain.connect(this.ctx.destination);

        osc.start(now);
        osc.stop(now + 0.08);
    }

    playPlace() {
        if (this.muted) return;
        this.init();
        const now = this.ctx.currentTime;
        const osc = this.ctx.createOscillator();
        const gain = this.ctx.createGain();

        osc.type = 'triangle';
        osc.frequency.setValueAtTime(220, now);
        osc.frequency.exponentialRampToValueAtTime(110, now + 0.12);

        gain.gain.setValueAtTime(0.25, now);
        gain.gain.exponentialRampToValueAtTime(0.001, now + 0.12);

        osc.connect(gain);
        gain.connect(this.ctx.destination);

        osc.start(now);
        osc.stop(now + 0.12);
    }

    playBreak() {
        if (this.muted) return;
        this.init();
        const now = this.ctx.currentTime;
        // White noise burst con filtro pasa-altos
        const bufferSize = this.ctx.sampleRate * 0.12;
        const buffer = this.ctx.createBuffer(1, bufferSize, this.ctx.sampleRate);
        const data = buffer.getChannelData(0);
        for (let i = 0; i < bufferSize; i++) {
            data[i] = Math.random() * 2 - 1;
        }

        const noise = this.ctx.createBufferSource();
        noise.buffer = buffer;

        const filter = this.ctx.createBiquadFilter();
        filter.type = 'bandpass';
        filter.frequency.value = 600;

        const gain = this.ctx.createGain();
        gain.gain.setValueAtTime(0.3, now);
        gain.gain.exponentialRampToValueAtTime(0.001, now + 0.12);

        noise.connect(filter);
        filter.connect(gain);
        gain.connect(this.ctx.destination);

        noise.start(now);
    }

    playHarvest() {
        if (this.muted) return;
        this.init();
        const now = this.ctx.currentTime;
        // Sonido de recolectar uvas: dos tonos brillantes
        [523.25, 659.25, 783.99].forEach((freq, idx) => {
            const osc = this.ctx.createOscillator();
            const gain = this.ctx.createGain();
            osc.type = 'sine';
            osc.frequency.setValueAtTime(freq, now + idx * 0.05);

            gain.gain.setValueAtTime(0.18, now + idx * 0.05);
            gain.gain.exponentialRampToValueAtTime(0.001, now + idx * 0.05 + 0.18);

            osc.connect(gain);
            gain.connect(this.ctx.destination);

            osc.start(now + idx * 0.05);
            osc.stop(now + idx * 0.05 + 0.18);
        });
    }

    playPress() {
        if (this.muted) return;
        this.init();
        const now = this.ctx.currentTime;
        // Sonido crujido de prensa de madera
        const osc = this.ctx.createOscillator();
        const gain = this.ctx.createGain();
        osc.type = 'sawtooth';
        osc.frequency.setValueAtTime(90, now);
        osc.frequency.linearRampToValueAtTime(140, now + 0.25);

        gain.gain.setValueAtTime(0.15, now);
        gain.gain.exponentialRampToValueAtTime(0.01, now + 0.25);

        osc.connect(gain);
        gain.connect(this.ctx.destination);
        osc.start(now);
        osc.stop(now + 0.25);
    }

    playToast() {
        if (this.muted) return;
        this.init();
        const now = this.ctx.currentTime;
        // Brindis de copa: campaneo cristalino de 2 copas (F#6 / 1479 Hz y G#6 / 1661 Hz)
        [1479.98, 1661.22].forEach((freq, i) => {
            const osc = this.ctx.createOscillator();
            const gain = this.ctx.createGain();
            osc.type = 'sine';
            osc.frequency.setValueAtTime(freq, now + i * 0.04);

            gain.gain.setValueAtTime(0.2, now + i * 0.04);
            gain.gain.exponentialRampToValueAtTime(0.001, now + i * 0.04 + 1.2);

            osc.connect(gain);
            gain.connect(this.ctx.destination);

            osc.start(now + i * 0.04);
            osc.stop(now + i * 0.04 + 1.2);
        });
    }

    playPortal() {
        if (this.muted) return;
        this.init();
        const now = this.ctx.currentTime;
        const osc = this.ctx.createOscillator();
        const gain = this.ctx.createGain();

        osc.type = 'sine';
        osc.frequency.setValueAtTime(150, now);
        osc.frequency.exponentialRampToValueAtTime(900, now + 0.8);
        osc.frequency.exponentialRampToValueAtTime(300, now + 1.4);

        gain.gain.setValueAtTime(0.01, now);
        gain.gain.linearRampToValueAtTime(0.2, now + 0.6);
        gain.gain.exponentialRampToValueAtTime(0.001, now + 1.4);

        osc.connect(gain);
        gain.connect(this.ctx.destination);

        osc.start(now);
        osc.stop(now + 1.4);
    }

    toggleMusic() {
        if (this.musicPlaying) {
            this.stopMusic();
        } else {
            this.startMusic();
        }
        return this.musicPlaying;
    }

    startMusic() {
        this.init();
        this.musicPlaying = true;
        // Acordes acústicos suaves inspirados en atardeceres de viñedo (Re menor, Sol menor, La7, Fa mayor)
        const progression = [
            [293.66, 349.23, 440.00], // Dm (D4, F4, A4)
            [261.63, 329.63, 392.00], // C  (C4, E4, G4)
            [220.00, 277.18, 329.63], // A  (A3, C#4, E4)
            [174.61, 220.00, 261.63], // F  (F3, A3, C4)
            [196.00, 246.94, 293.66]  // G  (G3, B3, D4)
        ];

        let chordIdx = 0;
        const playNextChord = () => {
            if (!this.musicPlaying || this.muted) return;
            const now = this.ctx.currentTime;
            const chord = progression[chordIdx % progression.length];
            chordIdx++;

            chord.forEach((freq, noteIdx) => {
                const osc = this.ctx.createOscillator();
                const gain = this.ctx.createGain();
                osc.type = 'triangle';
                const startTime = now + noteIdx * 0.25;
                osc.frequency.setValueAtTime(freq, startTime);

                gain.gain.setValueAtTime(0.04, startTime);
                gain.gain.exponentialRampToValueAtTime(0.001, startTime + 2.8);

                osc.connect(gain);
                gain.connect(this.ctx.destination);

                osc.start(startTime);
                osc.stop(startTime + 2.8);
            });

            this.musicTimer = setTimeout(playNextChord, 4000);
        };

        playNextChord();
    }

    stopMusic() {
        this.musicPlaying = false;
        if (this.musicTimer) {
            clearTimeout(this.musicTimer);
            this.musicTimer = null;
        }
    }
}

window.SoundSystem = new SoundSystem();
