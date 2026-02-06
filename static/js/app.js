// ─────────────────────────────────────────────────
// App — main application controller
// ─────────────────────────────────────────────────
const App = (() => {
    let currentMode = 'navigation';
    let keysDown = {};

    function init() {
        MapCanvas.init();
        Joystick.init();
        Graphs.init();

        // Connect WebSocket
        WS.connect();

        // Register WebSocket handlers
        WS.on('map', (msg) => {
            MapCanvas.updateMap(msg.data);
        });

        WS.on('tf', (msg) => {
            MapCanvas.updateRobotPose(msg.data);
            updateInfoOverlay(msg.data, null);
        });

        WS.on('map_bfp', (msg) => {
            const d = msg.data;
            if (d) {
                MapCanvas.updateRobotPose({ x: d.x, y: d.y, yaw: d.theta });
                updateInfoOverlay({ x: d.x, y: d.y, yaw: d.theta }, null);
            }
        });

        WS.on('odom', (msg) => {
            const d = msg.data;
            Graphs.pushPosition(d.pos_x || d.x || 0, d.pos_y || d.y || 0);
        });

        WS.on('ctrl_odom', (msg) => {
            // Controller odom — can update graphs too
        });

        WS.on('velocity', (msg) => {
            const d = msg.data;
            Graphs.pushVelocity(d.linear_x || d.LinearX || 0, d.angular_z || d.AngularZ || 0);
            updateVelocityInfo(d);
        });

        WS.on('laser', (msg) => {
            MapCanvas.updateLaser(msg.data);
        });

        WS.on('status', (msg) => {
            updateStatusBadge(msg.data);
        });

        WS.on('robot_added', () => {
            refreshRobotList();
            updateRobotCount();
            Notify.success('Robot added');
        });

        WS.on('robot_removed', () => {
            refreshRobotList();
            updateRobotCount();
            Notify.info('Robot removed');
        });

        WS.on('robot_connected', (msg) => {
            Notify.success(`Robot ${msg.robot_id} connected`);
            refreshRobotList();
            updateRobotCount();
            updateConnBadge(true);
        });

        WS.on('robot_disconnected', (msg) => {
            Notify.warn(`Robot ${msg.robot_id} disconnected`);
            refreshRobotList();
            updateConnBadge(false);
        });

        WS.on('robot_switched', () => {
            refreshRobotList();
            WS.send({ type: 'request_map' });
            WS.send({ type: 'request_status' });
            refreshNavPoints();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', onKeyDown);
        document.addEventListener('keyup', onKeyUp);

        // Periodic status poll
        setInterval(() => {
            WS.send({ type: 'request_status' });
        }, 5000);
    }

    function updateInfoOverlay(tf, odom) {
        if (tf) {
            const x = tf.x || tf.X || 0;
            const y = tf.y || tf.Y || 0;
            const yaw = tf.yaw || tf.Yaw || tf.bfp_yaw || 0;
            setEl('info-x', x.toFixed(3));
            setEl('info-y', y.toFixed(3));
            setEl('info-theta', (yaw * 180 / Math.PI).toFixed(1) + '°');
        }
    }

    function updateVelocityInfo(v) {
        const lx = v.linear_x || v.LinearX || 0;
        const az = v.angular_z || v.AngularZ || 0;
        setEl('info-vel', lx.toFixed(3) + ' m/s');
        setEl('info-omega', az.toFixed(3) + ' rad/s');
    }

    function updateStatusBadge(data) {
        if (!data) return;
        updateConnBadge(data.connected);
        const freq = document.getElementById('freq-badge');
        if (freq && data.tf_hz !== undefined) {
            const hz = data.tf_hz || data.odom_hz || 0;
            freq.textContent = `TF:${data.tf_hz || 0} Odom:${data.odom_hz || 0} Map:${data.map_hz || 0}`;
        }
    }

    function updateConnBadge(connected) {
        const badge = document.getElementById('conn-badge');
        if (badge) {
            badge.textContent = connected ? 'Connected' : 'Disconnected';
            badge.classList.toggle('connected', connected);
        }
    }

    function updateRobotCount() {
        fetch('/api/robots')
            .then(r => r.json())
            .then(data => {
                const count = Array.isArray(data) ? data.length : 0;
                setEl('robot-count', `${count} robot${count !== 1 ? 's' : ''}`);
            })
            .catch(() => {});
    }

    function setEl(id, text) {
        const el = document.getElementById(id);
        if (el) el.textContent = text;
    }

    // ──────────── Mode switching ────────────

    function setMode(mode) {
        currentMode = mode;
        setEl('mode-label', mode.charAt(0).toUpperCase() + mode.slice(1));

        // Highlight active mode button
        document.querySelectorAll('.top-bar-center .btn').forEach(b => {
            b.classList.remove('btn-mode-active');
        });
        const modeBtn = document.getElementById(`btn-${mode === 'navigation' ? 'navi' : mode}`);
        if (modeBtn) modeBtn.classList.add('btn-mode-active');

        // Call server to switch robot mode
        if (mode !== 'mapediting' && mode !== 'settings') {
            fetch(`/api/mode/${mode}`, { method: 'POST' })
                .then(r => r.json())
                .then(data => {
                    if (data.error) {
                        Notify.error(data.error);
                    } else {
                        Notify.info(`Mode: ${mode}`);
                    }
                })
                .catch(err => Notify.error('Mode switch failed'));
        }

        // Show/hide right sidebar sections based on mode
        if (mode === 'settings') {
            showSection('settings');
        } else {
            showSection('nav');
        }
    }

    // ──────────── Section tabs ────────────

    function showSection(name) {
        ['nav', 'settings', 'graphs', 'speech'].forEach(s => {
            const el = document.getElementById(`section-${s}`);
            const tab = document.getElementById(`tab-${s}`);
            if (el) el.classList.toggle('hidden', s !== name);
            if (tab) tab.classList.toggle('active', s === name);
        });

        // Refresh content
        if (name === 'settings') {
            htmx.ajax('GET', '/partial/settings', { target: '#settings-content', swap: 'innerHTML' });
        } else if (name === 'nav') {
            refreshNavPoints();
        }
    }

    // ──────────── Robot actions ────────────

    function switchRobot(id) {
        fetch(`/api/robots/switch?id=${id}`, { method: 'POST' })
            .then(() => {
                WS.send({ type: 'switch_robot', data: { id } });
                WS.send({ type: 'request_map', robot_id: id });
                refreshRobotList();
                refreshNavPoints();
            });
    }

    function refreshRobotList() {
        htmx.ajax('GET', '/partial/robots', { target: '#robot-list', swap: 'innerHTML' });
    }

    function refreshNavPoints() {
        htmx.ajax('GET', '/partial/nav_points', { target: '#nav-points-content', swap: 'innerHTML' });
    }

    // ──────────── Map actions ────────────

    function openMap(name) {
        fetch('/api/maps/open', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name })
        })
        .then(r => r.json())
        .then(data => {
            hideDialog();
            if (data.error) {
                Notify.error(data.error);
            } else {
                Notify.success(`Map "${name}" opened`);
                // Request new map data
                WS.send({ type: 'request_map' });
            }
        })
        .catch(err => {
            hideDialog();
            Notify.error('Failed to open map');
        });
    }

    function fetchMapList() {
        return fetch('/api/maps').then(r => r.json());
    }

    // ──────────── Settings ────────────

    function saveSettings() {
        const lr = document.getElementById('setting-linear-ratio')?.value || '1.0';
        const ar = document.getElementById('setting-angular-ratio')?.value || '1.0';
        const radius = document.getElementById('setting-radius')?.value || '0.30';

        fetch('/api/robots/settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `linear_vel_ratio=${lr}&angular_vel_ratio=${ar}&radius=${radius}`
        })
        .then(r => r.json())
        .then(data => {
            if (data.error) Notify.error(data.error);
            else Notify.success('Settings saved');
        });
    }

    // ──────────── Placement mode (for map toolbar) ────────────

    function setPlacementMode(mode) {
        MapCanvas.setPlacementMode(mode);
    }

    // ──────────── Key bindings ────────────

    function onKeyDown(e) {
        // Ignore if user is typing in an input/textarea
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;

        if (keysDown[e.key]) return; // Prevent key repeat spam
        keysDown[e.key] = true;

        switch (e.key) {
            case 'w': case 'ArrowUp':
                WS.sendJoystick(0.3, 0); e.preventDefault(); break;
            case 's': case 'ArrowDown':
                WS.sendJoystick(-0.3, 0); e.preventDefault(); break;
            case 'a': case 'ArrowLeft':
                WS.sendJoystick(0, 0.5); e.preventDefault(); break;
            case 'd': case 'ArrowRight':
                WS.sendJoystick(0, -0.5); e.preventDefault(); break;
            case ' ':
                WS.sendStop();
                e.preventDefault();
                break;
            case 'Escape':
                setPlacementMode(null);
                hideDialog();
                break;
        }
    }

    function onKeyUp(e) {
        delete keysDown[e.key];
        // Stop robot when movement key is released
        const movementKeys = ['w', 's', 'a', 'd', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'];
        if (movementKeys.includes(e.key)) {
            // Check if any other movement key is still held
            const stillHeld = movementKeys.some(k => keysDown[k]);
            if (!stillHeld) {
                WS.sendStop();
            }
        }
    }

    // ──────────── Zoom/view delegates ────────────

    function zoomIn()    { MapCanvas.zoomIn(); }
    function zoomOut()   { MapCanvas.zoomOut(); }
    function resetView() { MapCanvas.resetView(); }

    return {
        init, setMode, showSection, switchRobot, openMap, saveSettings,
        setPlacementMode, zoomIn, zoomOut, resetView, refreshNavPoints,
        fetchMapList, updateRobotCount
    };
})();

// ──────────── Dialog helpers ────────────

function showDialog() {
    document.getElementById('dialog-overlay').classList.remove('hidden');
}

function hideDialog() {
    const overlay = document.getElementById('dialog-overlay');
    overlay.classList.add('hidden');
    overlay.innerHTML = '';
}

// ──────────── Global mode setter (called from top bar buttons) ────────────
function setMode(mode) {
    App.setMode(mode);
}

// ──────────── Bootstrap ────────────
document.addEventListener('DOMContentLoaded', App.init);
