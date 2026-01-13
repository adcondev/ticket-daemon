/* ==============================================================
   ESTADO (STATE)
   ============================================================== */
// Estado global de la aplicación
const state = {
  socket: null,
  isConnected: false,
  jobsSent: 0,
  logCount: 0,
  pollTimer: null,
  startTime: Date.now()
};

/* ==============================================================
   ELEMENTOS DEL DOM
   ============================================================== */
// Referencias a elementos DOM (se inicializan después del DOMContentLoaded)
let el = {};

function initElements() {
  el = {
    connStatus: document.getElementById('connStatus'),
    queueVal: document.getElementById('queueVal'),
    queueBar: document.getElementById('queueBar'),
    queuePct: document.getElementById('queuePct'),
    workerVal: document.getElementById('workerVal'),
    workerSub: document.getElementById('workerSub'),
    envVal: document.getElementById('envVal'),
    buildInfo: document.getElementById('buildInfo'),
    jobsSentVal: document.getElementById('jobsSentVal'),
    uptimeVal: document.getElementById('uptimeVal'),
    processedVal: document.getElementById('processedVal'),
    failedVal: document.getElementById('failedVal'),
    jsonEditor: document.getElementById('jsonEditor'),
    jsonStatus: document.getElementById('jsonStatus'),
    lineCount: document.getElementById('lineCount'),
    cmdCount: document.getElementById('cmdCount'),
    logContainer: document.getElementById('logContainer'),
    logCount: document.getElementById('logCount'),
    templateSelect: document.getElementById('templateSelect'),
    btnSend: document.getElementById('btnSend'),
    btnBurst: document.getElementById('btnBurst'),
    burstModal: document.getElementById('burstModal'),
    troubleshootCard: document.getElementById('troubleshootCard')
  };
}
