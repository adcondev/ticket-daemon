/* ==============================================================
   WEBSOCKET
   ============================================================== */
// Conexión y manejo de WebSocket
function connectWebSocket() {
  addLog('INFO', `Conectando a ${CONFIG.WS_URL}...`);
  state.socket = new WebSocket(CONFIG.WS_URL);

  state.socket.onopen = () => {
    state.isConnected = true;
    updateConnectionUI(true);
      addLog('INFO', '✅ Cliente Conectado al Ticket Daemon');
    showToast('Conectado al servicio', 'success');
    fetchHealth();
  };

  state.socket.onclose = () => {
    state.isConnected = false;
    updateConnectionUI(false);
    addLog('ERROR', '❌ Conexión perdida. Reintentando...');
    setTimeout(connectWebSocket, CONFIG.RECONNECT_DELAY);
  };

  state.socket.onerror = () => {
    addLog('ERROR', '⚠️ Error de WebSocket');
  };

  state.socket.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      handleMessage(msg);
    } catch {
      addLog('INFO', event.data);
    }
  };
}

// [GUÍA DE INTEGRACIÓN] Manejo de respuestas del servidor
function handleMessage(msg) {
  const tipo = (msg.tipo || 'info').toLowerCase();

  switch (tipo) {
    case 'info':
      addLog('INFO', msg.mensaje || JSON.stringify(msg));
      break;
    case 'ack': // Confirmación de que el trabajo entró a la cola
      addLog('ACK', `📥 Encolado: ${msg.id} (Posición: ${msg.current ?? 0}/${msg.capacity ?? 100})`);
      fetchHealth();
      break;
    case 'result': // Resultado final de la impresión (éxito o fallo)
      if (msg.status === 'success') {
        addLog('RESULT', `✅ Completado: ${msg.id} — ${msg.mensaje}`, 'success');
        showToast('¡Impresión completada!', 'success');
      } else {
        addLog('ERROR', `❌ Falló [${msg.id}]: ${msg.mensaje}`, 'error');
        showToast('Error de impresión', 'error');
      }
      fetchHealth();
      break;
    case 'error': // Error inmediato (validación, cola llena)
      if (msg.mensaje && msg.mensaje.includes('Authentication failed')) {
        addLog('ERROR', '🔒 ' + msg.mensaje, 'error');
      } else if (msg.mensaje && msg.mensaje.includes('Rate limited')) {
        addLog('ERROR', '⏳ ' + msg.mensaje, 'error');
      } else {
        addLog('ERROR', '❌ ' + msg.mensaje);
      }
      showToast(msg.mensaje, 'error');
      break;
    case 'pong':
      addLog('PONG', `🏓 Pong (id: ${msg.id})`);
      break;
    case 'status':
      const current = msg.current ?? 0;
      const capacity = msg.capacity ?? 100;
      addLog('STATUS', `📊 Cola: ${current}/${capacity}`);
      updateQueueDisplay(current, capacity);
        break;
      case 'printers':
          addLog('PRINTERS', `🖨️ Found ${msg.printers?.length || 0} printers`);

          if (msg.printers) {
              const thermal = msg.printers.filter(p => p.printer_type === 'thermal');
              // Keep detailed lines as INFO to avoid visual noise, or change to PRINTERS if you prefer
              addLog('PRINTERS', `   -> Thermal: ${thermal.length}, Virtual: ${msg.printers.length - thermal.length}`);

              thermal.forEach(p => {
                  const def = p.is_default ? '⭐' : '';
                  // Using padStart for better alignment in logs
                  addLog('PRINTERS', `      • ${p.name} [${p.port}] (${p.status})${def}`);
              });
          }
      break;
    default:
      addLog('INFO', JSON.stringify(msg));
  }
}

// [GUÍA DE INTEGRACIÓN] Función principal para enviar datos al servidor
function sendMessage(msg) {
  if (!state.socket || !state.isConnected) {
    showToast('No conectado', 'error');
    return false;
  }
  state.socket.send(JSON.stringify(msg));
  return true;
}

// Read the auth token injected by the server into the HTML template
function getAuthToken() {
  const meta = document.querySelector('meta[name="ws-auth-token"]');
  return meta ? meta.content : '';
}