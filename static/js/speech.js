// ─────────────────────────────────────────────────
// Speech — record from microphone, send for transcription
// ─────────────────────────────────────────────────
const Speech = (() => {
    let recording = false;
    let mediaRecorder = null;
    let chunks = [];

    async function toggle() {
        if (recording) {
            stop();
        } else {
            await start();
        }
    }

    async function start() {
        try {
            const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            mediaRecorder = new MediaRecorder(stream, { mimeType: 'audio/webm' });
            chunks = [];

            mediaRecorder.ondataavailable = (e) => {
                if (e.data.size > 0) chunks.push(e.data);
            };

            mediaRecorder.onstop = async () => {
                stream.getTracks().forEach(t => t.stop());
                const blob = new Blob(chunks, { type: 'audio/webm' });
                await transcribe(blob);
            };

            mediaRecorder.start();
            recording = true;
            updateUI();
        } catch (err) {
            console.error('[speech] mic error:', err);
            Notify.error('Microphone access denied');
        }
    }

    function stop() {
        if (mediaRecorder && mediaRecorder.state !== 'inactive') {
            mediaRecorder.stop();
        }
        recording = false;
        updateUI();
    }

    async function transcribe(blob) {
        const statusEl = document.getElementById('speech-status');
        const resultEl = document.getElementById('speech-result');
        if (statusEl) statusEl.textContent = 'Transcribing...';

        const form = new FormData();
        form.append('audio', blob, 'recording.webm');

        try {
            const resp = await fetch('/api/speech/transcribe', {
                method: 'POST',
                body: form
            });
            const data = await resp.json();

            if (data.error) {
                if (statusEl) statusEl.textContent = 'Error';
                Notify.error(data.error);
            } else {
                if (statusEl) statusEl.textContent = 'Done';
                if (resultEl) resultEl.textContent = data.text || '(empty)';
                if (data.text) {
                    Notify.success(`Voice: "${data.text}"`);
                }
            }
        } catch (err) {
            if (statusEl) statusEl.textContent = 'Error';
            console.error('[speech] transcribe error:', err);
        }
    }

    function updateUI() {
        const btn = document.getElementById('btn-record');
        const statusEl = document.getElementById('speech-status');
        if (btn) {
            btn.textContent = recording ? '⏹ Stop' : '🎤 Record';
            btn.classList.toggle('recording', recording);
        }
        if (statusEl) {
            statusEl.textContent = recording ? 'Recording...' : 'Ready';
        }
    }

    return { toggle, start, stop };
})();
