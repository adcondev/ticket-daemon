/* ==============================================================
   FUNCIONES DE INTERFAZ DE USUARIO
   ============================================================== */

// Actualizar UI de conexión
function updateConnectionUI(connected) {
  el.connStatus.className = 'conn-badge ' + (connected ? 'online' : 'offline');
  el.connStatus.innerHTML = `<span class="conn-dot"></span><span>${connected ? 'En Línea' : 'Desconectado'}</span>`;
  el.btnSend.disabled = !connected;
  el.btnBurst.disabled = !connected;
}

/* ==============================================================
   LOGGING
   ============================================================== */
function addLog(type, message, textClass = '') {
  const time = new Date().toLocaleTimeString('es-MX', {hour12: false});
  const entry = document.createElement('div');
  entry.className = 'log-entry';

  const badgeClass = type.toLowerCase();
  const msgClass = textClass ? `log-message ${textClass}-text` : 'log-message';

  entry.innerHTML = `
    <span class="log-time">${time}</span>
    <span class="log-badge ${badgeClass}">${type}</span>
    <span class="${msgClass}">${escapeHtml(message)}</span>
  `;

  el.logContainer.appendChild(entry);
  el.logContainer.scrollTop = el.logContainer.scrollHeight;

  state.logCount++;
  el.logCount.textContent = `${state.logCount} entradas`;

  while (el.logContainer.children.length > CONFIG.MAX_LOGS) {
    el.logContainer.removeChild(el.logContainer.firstChild);
  }
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

/* ==============================================================
   TOASTS (Notificaciones)
   ============================================================== */
function showToast(message, type = 'info') {
  const container = document.getElementById('toastContainer');
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;

  const icons = {success: '✅', error: '❌', warning: '⚠️', info: 'ℹ️'};
  toast.innerHTML = `
    <span class="toast-icon">${icons[type]}</span>
    <span class="toast-message">${escapeHtml(message)}</span>
  `;

  container.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateX(100px)';
    setTimeout(() => toast.remove(), 300);
  }, 3500);
}

/* ==============================================================
   VALIDACIÓN JSON
   ============================================================== */
function validateJSON() {
  const input = el.jsonEditor.value;

  try {
    const doc = JSON.parse(input);
    const errors = [];

    if (!doc.version) errors.push('Falta versión');
    if (!doc.profile?.model) errors.push('Falta profile.model');
    if (!doc.commands?.length) errors.push('No hay comandos');

    if (errors.length > 0) {
      el.jsonStatus.className = 'json-status invalid';
      el.jsonStatus.innerHTML = `<span>⚠️</span><span>${errors[0]}</span>`;
      return false;
    }

    el.jsonStatus.className = 'json-status valid';
    el.jsonStatus.innerHTML = '<span>✅</span><span>Válido</span>';

    el.lineCount.textContent = `${input.split('\n').length} líneas`;
    el.cmdCount.textContent = `${doc.commands?.length || 0} comandos`;

    return true;
  } catch {
    el.jsonStatus.className = 'json-status invalid';
    el.jsonStatus.innerHTML = '<span>❌</span><span>JSON Inválido</span>';
    return false;
  }
}

/* ==============================================================
   ACTUALIZACIÓN DE VISTA DE COLA
   ============================================================== */
function updateQueueDisplay(current, capacity) {
  const pct = Math.round((current / capacity) * 100);
  el.queueVal.textContent = `${current} / ${capacity}`;
  el.queuePct.textContent = `${pct}%`;
  el.queueBar.style.width = `${pct}%`;
  el.queueBar.classList.toggle('warning', pct > 70);
}

/* ==============================================================
   FORMATO DE TIEMPO
   ============================================================== */
function formatUptime(seconds) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

/* ==============================================================
   POLLING DE SALUD (HEALTH)
   ============================================================== */
async function fetchHealth() {
  try {
    const res = await fetch(CONFIG.HEALTH_URL);
    if (!res.ok) throw new Error('Health falló');
    const data = await res.json();

    // Queue (Cola)
    const current = data.queue?.current || 0;
    const capacity = data.queue?.capacity || 100;
    updateQueueDisplay(current, capacity);

    // Worker (Procesador)
    const running = data.worker?.running;
    el.workerVal.textContent = running ? '🟢 Activo' : '🔴 Detenido';
    el.workerVal.className = 'metric-value ' + (running ? 'success' : 'error');

    const processed = data.worker?.jobs_processed || 0;
    const failed = data.worker?.jobs_failed || 0;
    el.workerSub.textContent = `${processed} procesados, ${failed} fallidos`;
    el.processedVal.textContent = processed;
    el.failedVal.textContent = failed;

    // Environment (Entorno)
    el.envVal.textContent = (data.build?.env || 'desconocido').toUpperCase();
    el.buildInfo.textContent = `Build: ${data.build?.date || '--'} ${data.build?.time || ''}`;

    // Uptime
    const uptime = data.uptime_seconds || 0;
    el.uptimeVal.textContent = formatUptime(uptime);

  } catch {
    el.workerVal.textContent = '⚫ Offline';
    el.workerVal.className = 'metric-value muted';
    el.workerSub.textContent = 'No se puede contactar al daemon';
  }
}
