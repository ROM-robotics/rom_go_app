// ─────────────────────────────────────────────────
// WebSocket client — browser ↔ Go server
// ─────────────────────────────────────────────────
const WS = (() => {
    let ws = null;
    let reconnectTimer = null;
    const handlers = {};

    function connect() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${proto}//${location.host}/ws`;
        ws = new WebSocket(url);

        ws.onopen = () => {
            console.log('[ws] connected');
            if (reconnectTimer) { clearInterval(reconnectTimer); reconnectTimer = null; }
            document.getElementById('conn-badge').textContent = 'WS Connected';
            document.getElementById('conn-badge').classList.add('connected');
            // Request initial state
            send({ type: 'request_map' });
            send({ type: 'request_status' });
        };

        ws.onmessage = (ev) => {
            try {
                const msg = JSON.parse(ev.data);
                const fn = handlers[msg.type];
                if (fn) fn(msg);
            } catch (e) {
                console.warn('[ws] parse error:', e);
            }
        };

        ws.onclose = () => {
            console.log('[ws] disconnected');
            document.getElementById('conn-badge').textContent = 'Disconnected';
            document.getElementById('conn-badge').classList.remove('connected');
            scheduleReconnect();
        };

        ws.onerror = (e) => {
            console.error('[ws] error:', e);
            ws.close();
        };
    }

    function scheduleReconnect() {
        if (!reconnectTimer) {
            reconnectTimer = setInterval(() => {
                console.log('[ws] attempting reconnect...');
                connect();
            }, 3000);
        }
    }

    function send(msg) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify(msg));
        }
    }

    function on(type, callback) {
        handlers[type] = callback;
    }

    function sendJoystick(linearX, angularZ) {
        send({
            type: 'joystick',
            data: { linear_x: linearX, angular_z: angularZ }
        });
    }

    function sendStop() {
        send({ type: 'stop' });
    }

    return { connect, send, on, sendJoystick, sendStop };
})();
