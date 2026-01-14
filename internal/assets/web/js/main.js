/* ==============================================================
   INICIALIZACIÓN Y EVENT HANDLERS
   ============================================================== */

// Inicialización al cargar el DOM
document.addEventListener('DOMContentLoaded', () => {
  initElements();
  init();
});

// Función de inicialización principal
function init() {
  el.jsonEditor.value = JSON.stringify(TEMPLATES.simple, null, 2);
  validateJSON();
  connectWebSocket();
  state.pollTimer = setInterval(fetchHealth, CONFIG.POLL_INTERVAL);
  fetchHealth();
  addLog('INFO', '🚀 Dashboard inicializado');
  
  setupEventListeners();
}

// Configurar todos los event listeners
function setupEventListeners() {
  // Cargar Plantilla
  document.getElementById('btnLoad').addEventListener('click', () => {
    const key = el.templateSelect.value;
    const template = TEMPLATES[key];
    if (template) {
      el.jsonEditor.value = JSON.stringify(template, null, 2);
      validateJSON();
      addLog('INFO', `📄 Plantilla cargada: ${key}`);
    }
  });

  // Enviar Trabajo de Impresión
  // [GUÍA DE INTEGRACIÓN] Ejemplo de envío de un ticket
  el.btnSend.addEventListener('click', () => {
    if (!validateJSON()) {
      showToast('Corrige los errores del JSON primero', 'error');
      return;
    }

    try {
      const payload = JSON.parse(el.jsonEditor.value);
      const jobId = `job-${Date.now()}`;
      // Estructura del mensaje: { tipo: 'ticket', id: '...', datos: { ... } }
      const msg = {tipo: 'ticket', id: jobId, datos: payload};

      if (sendMessage(msg)) {
        addLog('SENT', `📤 Trabajo: ${jobId}`);
        state.jobsSent++;
        el.jobsSentVal.textContent = state.jobsSent;
      }
    } catch (e) {
      addLog('ERROR', `Envío fallido: ${e.message}`);
    }
  });

  // Modo Ráfaga
  el.btnBurst.addEventListener('click', () => {
    el.burstModal.classList.add('show');
  });

  document.getElementById('btnBurstCancel').addEventListener('click', () => {
    el.burstModal.classList.remove('show');
  });

  document.getElementById('btnBurstConfirm').addEventListener('click', () => {
    el.burstModal.classList.remove('show');

    let payload;
    try {
      payload = JSON.parse(el.jsonEditor.value);
    } catch {
      payload = TEMPLATES.burstable;
      addLog('INFO', '⚠️ Usando plantilla Ráfaga');
    }

    addLog('INFO', '🔥 RÁFAGA: Enviando 10 trabajos...');

    for (let i = 0; i < 10; i++) {
      const jobId = `burst-${Date.now()}-${i}`;
      if (sendMessage({tipo: 'ticket', id: jobId, datos: payload})) {
        state.jobsSent++;
      }
    }

    el.jobsSentVal.textContent = state.jobsSent;
    showToast('Ráfaga: ¡10 trabajos enviados!', 'warning');
    setTimeout(fetchHealth, 100);
  });

  // Ping
  document.getElementById('btnPing').addEventListener('click', () => {
    const id = `ping-${Date.now()}`;
    if (sendMessage({tipo: 'ping', id})) {
      addLog('SENT', `🏓 Ping (${id})`);
    }
  });

  // Estado
  document.getElementById('btnStatus').addEventListener('click', () => {
    if (sendMessage({tipo: 'status'})) {
      addLog('SENT', '📊 Solicitud de estado');
    }
  });

  // Actualizar (Refresh)
  document.getElementById('btnRefresh').addEventListener('click', () => {
    fetchHealth();
    showToast('Actualizado', 'info');
  });

  // Limpiar Logs
  document.getElementById('btnClearLogs').addEventListener('click', () => {
    el.logContainer.innerHTML = '';
    state.logCount = 0;
    el.logCount.textContent = '0 entradas';
  });

  // Exportar Logs
  document.getElementById('btnExportLogs').addEventListener('click', () => {
    const entries = el.logContainer.querySelectorAll('.log-entry');
    const lines = Array.from(entries).map(e => {
      const time = e.querySelector('.log-time').textContent;
      const type = e.querySelector('.log-badge').textContent;
      const msg = e.querySelector('.log-message').textContent;
      return `[${time}] ${type}: ${msg}`;
    });

    const blob = new Blob([lines.join('\n')], {type: 'text/plain'});
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `ticket-daemon-${Date.now()}.log`;
    a.click();
    URL.revokeObjectURL(url);
    showToast('Logs exportados', 'success');
  });

  // Formatear JSON
  document.getElementById('btnFormat').addEventListener('click', () => {
    try {
      const doc = JSON.parse(el.jsonEditor.value);
      el.jsonEditor.value = JSON.stringify(doc, null, 2);
      validateJSON();
      showToast('Formateado', 'success');
    } catch {
      showToast('No se puede formatear JSON inválido', 'error');
    }
  });

  // Copiar JSON
  document.getElementById('btnCopy').addEventListener('click', () => {
    navigator.clipboard.writeText(el.jsonEditor.value);
    showToast('Copiado al portapapeles', 'info');
  });

  // Toggle de Solución de Problemas
  document.getElementById('troubleshootToggle').addEventListener('click', () => {
    el.troubleshootCard.classList.toggle('open');
  });

  // Validación del editor al escribir
  el.jsonEditor.addEventListener('input', validateJSON);

  // Cerrar modal al hacer clic fuera
  el.burstModal.addEventListener('click', (e) => {
    if (e.target === el.burstModal) {
      el.burstModal.classList.remove('show');
    }
  });
}
