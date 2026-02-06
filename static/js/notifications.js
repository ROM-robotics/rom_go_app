// ─────────────────────────────────────────────────
// Notifications — toast-style messages
// ─────────────────────────────────────────────────
const Notify = (() => {
    function show(message, level = 'info', duration = 4000) {
        const container = document.getElementById('notification-container');
        if (!container) return;

        const el = document.createElement('div');
        el.className = `notification notification-${level}`;
        el.innerHTML = `
            <span class="notification-text">${escapeHtml(message)}</span>
            <button class="notification-close" onclick="this.parentElement.remove()">✕</button>
        `;
        container.appendChild(el);

        // Animate in
        requestAnimationFrame(() => el.classList.add('show'));

        // Auto dismiss
        if (duration > 0) {
            setTimeout(() => {
                el.classList.remove('show');
                setTimeout(() => el.remove(), 300);
            }, duration);
        }
    }

    function success(msg) { show(msg, 'success'); }
    function error(msg)   { show(msg, 'error', 6000); }
    function warn(msg)    { show(msg, 'warn', 5000); }
    function info(msg)    { show(msg, 'info'); }

    function escapeHtml(s) {
        const div = document.createElement('div');
        div.textContent = s;
        return div.innerHTML;
    }

    return { show, success, error, warn, info };
})();
