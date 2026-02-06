// ─────────────────────────────────────────────────
// Virtual Joystick — touch & mouse support
// ─────────────────────────────────────────────────
const Joystick = (() => {
    let canvas, ctx;
    const SIZE = 180;
    const CENTER = SIZE / 2;
    const OUTER_R = 75;
    const INNER_R = 25;

    let knobX = CENTER, knobY = CENTER;
    let active = false;
    let sendInterval = null;

    function init() {
        canvas = document.getElementById('joystick-canvas');
        if (!canvas) return;
        ctx = canvas.getContext('2d');
        canvas.width = SIZE;
        canvas.height = SIZE;

        // Mouse events
        canvas.addEventListener('mousedown', onStart);
        canvas.addEventListener('mousemove', onMove);
        canvas.addEventListener('mouseup', onEnd);
        canvas.addEventListener('mouseleave', onEnd);

        // Touch events
        canvas.addEventListener('touchstart', onTouchStart, { passive: false });
        canvas.addEventListener('touchmove', onTouchMove, { passive: false });
        canvas.addEventListener('touchend', onEnd);
        canvas.addEventListener('touchcancel', onEnd);

        draw();
    }

    function getValues() {
        const dx = (knobX - CENTER) / OUTER_R;
        const dy = -(knobY - CENTER) / OUTER_R; // Invert Y: up = positive
        return {
            linearX: Math.max(-1, Math.min(1, dy)),  // forward/backward
            angularZ: Math.max(-1, Math.min(1, -dx))  // left/right
        };
    }

    function clampKnob(x, y) {
        const dx = x - CENTER;
        const dy = y - CENTER;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist > OUTER_R) {
            knobX = CENTER + (dx / dist) * OUTER_R;
            knobY = CENTER + (dy / dist) * OUTER_R;
        } else {
            knobX = x;
            knobY = y;
        }
    }

    function onStart(e) {
        active = true;
        const rect = canvas.getBoundingClientRect();
        clampKnob(e.clientX - rect.left, e.clientY - rect.top);
        startSending();
        draw();
    }

    function onMove(e) {
        if (!active) return;
        const rect = canvas.getBoundingClientRect();
        clampKnob(e.clientX - rect.left, e.clientY - rect.top);
        draw();
    }

    function onEnd() {
        active = false;
        knobX = CENTER;
        knobY = CENTER;
        stopSending();
        WS.sendStop();
        draw();
    }

    function onTouchStart(e) {
        e.preventDefault();
        const touch = e.touches[0];
        const rect = canvas.getBoundingClientRect();
        active = true;
        clampKnob(touch.clientX - rect.left, touch.clientY - rect.top);
        startSending();
        draw();
    }

    function onTouchMove(e) {
        e.preventDefault();
        if (!active) return;
        const touch = e.touches[0];
        const rect = canvas.getBoundingClientRect();
        clampKnob(touch.clientX - rect.left, touch.clientY - rect.top);
        draw();
    }

    function startSending() {
        if (sendInterval) return;
        sendInterval = setInterval(() => {
            if (active) {
                const v = getValues();
                WS.sendJoystick(v.linearX, v.angularZ);
            }
        }, 50); // 20 Hz
    }

    function stopSending() {
        if (sendInterval) {
            clearInterval(sendInterval);
            sendInterval = null;
        }
    }

    function draw() {
        ctx.clearRect(0, 0, SIZE, SIZE);

        // Outer ring
        ctx.beginPath();
        ctx.arc(CENTER, CENTER, OUTER_R, 0, Math.PI * 2);
        ctx.strokeStyle = 'rgba(255,255,255,0.2)';
        ctx.lineWidth = 2;
        ctx.stroke();

        // Cross hair
        ctx.beginPath();
        ctx.moveTo(CENTER - OUTER_R, CENTER);
        ctx.lineTo(CENTER + OUTER_R, CENTER);
        ctx.moveTo(CENTER, CENTER - OUTER_R);
        ctx.lineTo(CENTER, CENTER + OUTER_R);
        ctx.strokeStyle = 'rgba(255,255,255,0.08)';
        ctx.lineWidth = 1;
        ctx.stroke();

        // Dead zone
        ctx.beginPath();
        ctx.arc(CENTER, CENTER, 8, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(255,255,255,0.1)';
        ctx.fill();

        // Knob
        const gradient = ctx.createRadialGradient(knobX, knobY, 5, knobX, knobY, INNER_R);
        gradient.addColorStop(0, active ? 'rgba(0, 212, 255, 0.9)' : 'rgba(100, 100, 120, 0.8)');
        gradient.addColorStop(1, active ? 'rgba(0, 150, 200, 0.5)' : 'rgba(60, 60, 80, 0.4)');

        ctx.beginPath();
        ctx.arc(knobX, knobY, INNER_R, 0, Math.PI * 2);
        ctx.fillStyle = gradient;
        ctx.fill();
        ctx.strokeStyle = active ? 'rgba(0, 212, 255, 0.8)' : 'rgba(150,150,170,0.5)';
        ctx.lineWidth = 2;
        ctx.stroke();

        // Values display
        if (active) {
            const v = getValues();
            ctx.font = '10px monospace';
            ctx.fillStyle = 'rgba(255,255,255,0.6)';
            ctx.textAlign = 'center';
            ctx.fillText(`V: ${v.linearX.toFixed(2)}`, CENTER, SIZE - 8);
            ctx.fillText(`ω: ${v.angularZ.toFixed(2)}`, CENTER, 14);
        }
    }

    return { init, getValues };
})();
