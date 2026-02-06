// ─────────────────────────────────────────────────
// Charts — velocity & position graphs using Chart.js
// ─────────────────────────────────────────────────
const Graphs = (() => {
    let velChart = null;
    let posChart = null;
    const MAX_POINTS = 100;
    const timestamps = [];
    const velLinear = [];
    const velAngular = [];
    const posX = [];
    const posY = [];
    let counter = 0;

    const chartDefaults = {
        animation: false,
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                labels: { color: '#aaa', font: { size: 10 } }
            }
        },
        scales: {
            x: {
                ticks: { color: '#666', maxTicksLimit: 5 },
                grid: { color: 'rgba(255,255,255,0.05)' }
            },
            y: {
                ticks: { color: '#666' },
                grid: { color: 'rgba(255,255,255,0.05)' }
            }
        }
    };

    function init() {
        const velCanvas = document.getElementById('chart-velocity');
        const posCanvas = document.getElementById('chart-position');
        if (!velCanvas || !posCanvas) return;

        velChart = new Chart(velCanvas, {
            type: 'line',
            data: {
                labels: timestamps,
                datasets: [
                    { label: 'Linear (m/s)', data: velLinear, borderColor: '#00d4ff', borderWidth: 1.5, pointRadius: 0, fill: false },
                    { label: 'Angular (rad/s)', data: velAngular, borderColor: '#ff6600', borderWidth: 1.5, pointRadius: 0, fill: false }
                ]
            },
            options: { ...chartDefaults }
        });

        posChart = new Chart(posCanvas, {
            type: 'line',
            data: {
                labels: timestamps,
                datasets: [
                    { label: 'X (m)', data: posX, borderColor: '#66ff66', borderWidth: 1.5, pointRadius: 0, fill: false },
                    { label: 'Y (m)', data: posY, borderColor: '#ff66ff', borderWidth: 1.5, pointRadius: 0, fill: false }
                ]
            },
            options: { ...chartDefaults }
        });
    }

    function pushVelocity(linear, angular) {
        counter++;
        timestamps.push(counter);
        velLinear.push(linear);
        velAngular.push(angular);

        if (timestamps.length > MAX_POINTS) {
            timestamps.shift();
            velLinear.shift();
            velAngular.shift();
        }

        if (velChart) velChart.update();
    }

    function pushPosition(x, y) {
        if (posX.length < timestamps.length) {
            posX.push(x);
            posY.push(y);
        } else {
            posX[posX.length - 1] = x;
            posY[posY.length - 1] = y;
        }

        if (posX.length > MAX_POINTS) {
            posX.shift();
            posY.shift();
        }

        if (posChart) posChart.update();
    }

    return { init, pushVelocity, pushPosition };
})();
