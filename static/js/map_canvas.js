// ─────────────────────────────────────────────────
// Map Canvas — renders OccupancyGrid + robot + points
// ─────────────────────────────────────────────────
const MapCanvas = (() => {
    let canvas, ctx;
    let mapImage = null;         // ImageData for the OccupancyGrid
    let mapInfo = null;          // { width, height, resolution, originX, originY }
    let robotPose = null;        // { x, y, theta }
    let laserPoints = [];        // [{x,y}, ...]
    let navPoints = {            // keyed by type
        waypoint: [],
        service_point: [],
        patrol_point: [],
        path_point: [],
        wall: []
    };

    // View transform
    let viewX = 0, viewY = 0, viewScale = 1;
    let isDragging = false, dragStartX = 0, dragStartY = 0;

    // Placement mode
    let placementMode = null;    // null | 'waypoint' | 'service_point' | ...

    // Colors
    const COLORS = {
        background: '#1a1a2e',
        free: '#2d2d44',
        occupied: '#e0e0e0',
        unknown: '#16213e',
        robot: '#00d4ff',
        robotDir: '#00ff88',
        laser: 'rgba(255, 100, 100, 0.4)',
        waypoint: '#ffcc00',
        service_point: '#00ccff',
        patrol_point: '#ff6600',
        path_point: '#66ff66',
        wall: '#ff3366',
        grid: 'rgba(255,255,255,0.05)'
    };

    function init() {
        canvas = document.getElementById('map-canvas');
        ctx = canvas.getContext('2d');

        resize();
        window.addEventListener('resize', resize);

        // Mouse events
        canvas.addEventListener('mousedown', onMouseDown);
        canvas.addEventListener('mousemove', onMouseMove);
        canvas.addEventListener('mouseup', onMouseUp);
        canvas.addEventListener('wheel', onWheel, { passive: false });
        canvas.addEventListener('click', onClick);

        // Touch events
        canvas.addEventListener('touchstart', onTouchStart, { passive: false });
        canvas.addEventListener('touchmove', onTouchMove, { passive: false });
        canvas.addEventListener('touchend', onTouchEnd);

        requestAnimationFrame(renderLoop);
    }

    function resize() {
        const container = document.getElementById('map-container');
        canvas.width = container.clientWidth;
        canvas.height = container.clientHeight;
    }

    // ──────────── Map data ────────────

    function updateMap(mapData) {
        if (!mapData || !mapData.width || !mapData.height) return;

        mapInfo = {
            width: mapData.width,
            height: mapData.height,
            resolution: mapData.resolution || 0.05,
            originX: mapData.origin_x || 0,
            originY: mapData.origin_y || 0
        };

        // Create image from occupancy grid
        const imgData = ctx.createImageData(mapData.width, mapData.height);
        const data = mapData.data || [];

        for (let i = 0; i < data.length; i++) {
            const val = data[i];
            const idx = i * 4;

            if (val === -1 || val === 255) {
                // Unknown
                imgData.data[idx]     = 22;
                imgData.data[idx + 1] = 33;
                imgData.data[idx + 2] = 62;
                imgData.data[idx + 3] = 255;
            } else if (val === 0) {
                // Free
                imgData.data[idx]     = 45;
                imgData.data[idx + 1] = 45;
                imgData.data[idx + 2] = 68;
                imgData.data[idx + 3] = 255;
            } else {
                // Occupied (higher = more certain)
                const brightness = Math.min(255, val * 2.5);
                imgData.data[idx]     = brightness;
                imgData.data[idx + 1] = brightness;
                imgData.data[idx + 2] = brightness;
                imgData.data[idx + 3] = 255;
            }
        }

        // Create offscreen canvas for the map image
        const offscreen = document.createElement('canvas');
        offscreen.width = mapData.width;
        offscreen.height = mapData.height;
        offscreen.getContext('2d').putImageData(imgData, 0, 0);
        mapImage = offscreen;

        // Auto-center on first map
        if (viewScale === 1 && viewX === 0 && viewY === 0) {
            autoFit();
        }
    }

    function autoFit() {
        if (!mapInfo) return;
        const mapPixelW = mapInfo.width;
        const mapPixelH = mapInfo.height;
        const scaleX = canvas.width / mapPixelW;
        const scaleY = canvas.height / mapPixelH;
        viewScale = Math.min(scaleX, scaleY) * 0.9;
        viewX = (canvas.width - mapPixelW * viewScale) / 2;
        viewY = (canvas.height - mapPixelH * viewScale) / 2;
    }

    function updateRobotPose(tf) {
        if (!tf) return;
        robotPose = {
            x: tf.x || 0,
            y: tf.y || 0,
            theta: tf.yaw || 0
        };
    }

    function updateLaser(laser) {
        if (!laser || !laser.ranges || !mapInfo || !robotPose) return;
        const points = [];
        const angleMin = laser.angle_min || 0;
        const angleInc = laser.angle_increment || 0;
        const ranges = laser.ranges || [];

        for (let i = 0; i < ranges.length; i++) {
            const r = ranges[i];
            if (!isFinite(r) || r <= 0) continue;
            const angle = angleMin + i * angleInc + robotPose.theta;
            points.push({
                x: robotPose.x + r * Math.cos(angle),
                y: robotPose.y + r * Math.sin(angle)
            });
        }
        laserPoints = points;
    }

    function updateNavPoints(type, points) {
        navPoints[type] = points || [];
    }

    // ──────────── World ↔ Pixel conversions ────────────

    function worldToMap(wx, wy) {
        if (!mapInfo) return { x: 0, y: 0 };
        const mx = (wx - mapInfo.originX) / mapInfo.resolution;
        const my = mapInfo.height - (wy - mapInfo.originY) / mapInfo.resolution;
        return { x: mx, y: my };
    }

    function mapToWorld(mx, my) {
        if (!mapInfo) return { x: 0, y: 0 };
        const wx = mx * mapInfo.resolution + mapInfo.originX;
        const wy = (mapInfo.height - my) * mapInfo.resolution + mapInfo.originY;
        return { x: wx, y: wy };
    }

    function screenToMap(sx, sy) {
        return {
            x: (sx - viewX) / viewScale,
            y: (sy - viewY) / viewScale
        };
    }

    // ──────────── Rendering ────────────

    function renderLoop() {
        render();
        requestAnimationFrame(renderLoop);
    }

    function render() {
        ctx.fillStyle = COLORS.background;
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        ctx.save();
        ctx.translate(viewX, viewY);
        ctx.scale(viewScale, viewScale);

        // Draw map
        if (mapImage) {
            ctx.imageSmoothingEnabled = false;
            ctx.drawImage(mapImage, 0, 0);
        }

        // Draw laser points
        if (laserPoints.length > 0) {
            ctx.fillStyle = COLORS.laser;
            for (const p of laserPoints) {
                const mp = worldToMap(p.x, p.y);
                ctx.fillRect(mp.x - 0.5, mp.y - 0.5, 1, 1);
            }
        }

        // Draw navigation points
        drawNavPoints('wall', COLORS.wall);
        drawNavPoints('path_point', COLORS.path_point);
        drawNavPoints('patrol_point', COLORS.patrol_point);
        drawNavPoints('service_point', COLORS.service_point);
        drawNavPoints('waypoint', COLORS.waypoint);

        // Draw robot
        if (robotPose && mapInfo) {
            const rp = worldToMap(robotPose.x, robotPose.y);
            const radius = 0.3 / mapInfo.resolution; // robot radius in pixels

            // Robot circle
            ctx.beginPath();
            ctx.arc(rp.x, rp.y, radius, 0, Math.PI * 2);
            ctx.fillStyle = 'rgba(0, 212, 255, 0.3)';
            ctx.fill();
            ctx.strokeStyle = COLORS.robot;
            ctx.lineWidth = 1.5 / viewScale;
            ctx.stroke();

            // Direction arrow
            const dirLen = radius * 1.5;
            const angle = -robotPose.theta; // canvas Y is inverted
            ctx.beginPath();
            ctx.moveTo(rp.x, rp.y);
            ctx.lineTo(rp.x + dirLen * Math.cos(angle), rp.y + dirLen * Math.sin(angle));
            ctx.strokeStyle = COLORS.robotDir;
            ctx.lineWidth = 2 / viewScale;
            ctx.stroke();
        }

        ctx.restore();
    }

    function drawNavPoints(type, color) {
        const pts = navPoints[type];
        if (!pts || pts.length === 0) return;
        if (!mapInfo) return;

        if (type === 'wall') {
            ctx.strokeStyle = color;
            ctx.lineWidth = 2 / viewScale;
            for (const w of pts) {
                const p1 = worldToMap(w.x1 || w.X1 || 0, w.y1 || w.Y1 || 0);
                const p2 = worldToMap(w.x2 || w.X2 || 0, w.y2 || w.Y2 || 0);
                ctx.beginPath();
                ctx.moveTo(p1.x, p1.y);
                ctx.lineTo(p2.x, p2.y);
                ctx.stroke();
            }
            return;
        }

        const r = 4 / viewScale;
        for (const p of pts) {
            const wx = p.world_x_m || p.WorldXM || 0;
            const wy = p.world_y_m || p.WorldYM || 0;
            const mp = worldToMap(wx, wy);

            // Draw marker
            ctx.beginPath();
            if (type === 'waypoint') {
                // Diamond
                ctx.moveTo(mp.x, mp.y - r);
                ctx.lineTo(mp.x + r, mp.y);
                ctx.lineTo(mp.x, mp.y + r);
                ctx.lineTo(mp.x - r, mp.y);
                ctx.closePath();
            } else if (type === 'patrol_point') {
                // Triangle
                ctx.moveTo(mp.x, mp.y - r);
                ctx.lineTo(mp.x + r, mp.y + r);
                ctx.lineTo(mp.x - r, mp.y + r);
                ctx.closePath();
            } else {
                // Circle
                ctx.arc(mp.x, mp.y, r, 0, Math.PI * 2);
            }
            ctx.fillStyle = color;
            ctx.fill();

            // Draw name label
            const name = p.name || p.Name || '';
            if (name) {
                ctx.font = `${10 / viewScale}px sans-serif`;
                ctx.fillStyle = '#fff';
                ctx.textAlign = 'center';
                ctx.fillText(name, mp.x, mp.y - r - 2 / viewScale);
            }

            // Draw direction line
            const theta = p.theta || p.Theta || 0;
            if (theta !== 0) {
                const dLen = r * 2;
                ctx.beginPath();
                ctx.moveTo(mp.x, mp.y);
                ctx.lineTo(mp.x + dLen * Math.cos(-theta), mp.y + dLen * Math.sin(-theta));
                ctx.strokeStyle = color;
                ctx.lineWidth = 1 / viewScale;
                ctx.stroke();
            }
        }
    }

    // ──────────── Mouse events ────────────

    function onMouseDown(e) {
        if (placementMode) return;
        isDragging = true;
        dragStartX = e.clientX - viewX;
        dragStartY = e.clientY - viewY;
        canvas.style.cursor = 'grabbing';
    }

    function onMouseMove(e) {
        if (!isDragging) return;
        viewX = e.clientX - dragStartX;
        viewY = e.clientY - dragStartY;
    }

    function onMouseUp() {
        isDragging = false;
        canvas.style.cursor = placementMode ? 'crosshair' : 'grab';
    }

    function onWheel(e) {
        e.preventDefault();
        const factor = e.deltaY < 0 ? 1.1 : 0.9;
        const mx = e.clientX;
        const my = e.clientY;

        viewX = mx - (mx - viewX) * factor;
        viewY = my - (my - viewY) * factor;
        viewScale *= factor;
    }

    function onClick(e) {
        if (!placementMode || !mapInfo) return;

        const rect = canvas.getBoundingClientRect();
        const sx = e.clientX - rect.left;
        const sy = e.clientY - rect.top;
        const mp = screenToMap(sx, sy);
        const wp = mapToWorld(mp.x, mp.y);

        // Open the add nav point dialog with pre-filled coordinates
        const type = placementMode;
        fetch(`/dialog/add_nav_point?type=${type}`)
            .then(r => r.text())
            .then(html => {
                document.getElementById('dialog-overlay').innerHTML = html;
                showDialog();
                // Pre-fill coordinates
                const xInput = document.getElementById('pt-x');
                const yInput = document.getElementById('pt-y');
                if (xInput) xInput.value = wp.x.toFixed(3);
                if (yInput) yInput.value = wp.y.toFixed(3);
            });
    }

    // ──────────── Touch events ────────────

    let lastTouchDist = 0;

    function onTouchStart(e) {
        if (e.touches.length === 1 && !placementMode) {
            isDragging = true;
            dragStartX = e.touches[0].clientX - viewX;
            dragStartY = e.touches[0].clientY - viewY;
        } else if (e.touches.length === 2) {
            isDragging = false;
            lastTouchDist = getTouchDist(e.touches);
        }
        e.preventDefault();
    }

    function onTouchMove(e) {
        if (e.touches.length === 1 && isDragging) {
            viewX = e.touches[0].clientX - dragStartX;
            viewY = e.touches[0].clientY - dragStartY;
        } else if (e.touches.length === 2) {
            const dist = getTouchDist(e.touches);
            const factor = dist / lastTouchDist;
            const cx = (e.touches[0].clientX + e.touches[1].clientX) / 2;
            const cy = (e.touches[0].clientY + e.touches[1].clientY) / 2;
            viewX = cx - (cx - viewX) * factor;
            viewY = cy - (cy - viewY) * factor;
            viewScale *= factor;
            lastTouchDist = dist;
        }
        e.preventDefault();
    }

    function onTouchEnd() {
        isDragging = false;
    }

    function getTouchDist(touches) {
        const dx = touches[0].clientX - touches[1].clientX;
        const dy = touches[0].clientY - touches[1].clientY;
        return Math.sqrt(dx * dx + dy * dy);
    }

    // ──────────── Public API ────────────

    return {
        init,
        updateMap,
        updateRobotPose,
        updateLaser,
        updateNavPoints,
        autoFit,

        zoomIn()  { viewScale *= 1.2; },
        zoomOut() { viewScale /= 1.2; },
        resetView() { autoFit(); },

        setPlacementMode(mode) {
            placementMode = mode;
            canvas.style.cursor = mode ? 'crosshair' : 'grab';
            // Update toolbar buttons
            document.querySelectorAll('.tool-btn').forEach(b => b.classList.remove('active'));
            const btnId = mode ? `tool-${mode === 'service_point' ? 'sp' : mode === 'patrol_point' ? 'pp' : mode === 'path_point' ? 'path' : mode}` : 'tool-pan';
            const btn = document.getElementById(btnId);
            if (btn) btn.classList.add('active');
        },

        getPlacementMode() { return placementMode; }
    };
})();
