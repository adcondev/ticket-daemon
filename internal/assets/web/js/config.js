/* ==============================================================
   CONFIGURACIÓN
   ============================================================== */
// [GUÍA DE INTEGRACIÓN] URL del WebSocket. Puerto por defecto 8766.
const CONFIG = {
  WS_URL: `ws://${window.location.hostname}:${window.location.port || 8766}/ws`,
  HEALTH_URL: `http://${window.location.hostname}:${window.location.port || 8766}/health`,
  POLL_INTERVAL: 2000,
  MAX_LOGS: 200,
  RECONNECT_DELAY: 3000
};

/* ==============================================================
   PLANTILLAS (TEMPLATES)
   [GUÍA DE INTEGRACIÓN] Estructura requerida para los documentos de impresión
   ============================================================== */
const TEMPLATES = {
  simple: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {
        type: "text",
        data: {content: {text: "PRUEBA DE CONEXIÓN", align: "center", content_style: {bold: true, size: "2x1"}}}
      },
      {type: "text", data: {content: {text: new Date().toLocaleString(), align: "center"}}},
      {type: "feed", data: {lines: 3}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  receipt: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {type: "text", data: {content: {text: "MI TIENDA", align: "center", content_style: {bold: true, size: "2x2"}}}},
      {type: "text", data: {content: {text: "Av. Principal 123", align: "center"}}},
      {type: "separator", data: {char: "-", length: 32}},
      {type: "text", data: {label: {text: "Fecha"}, content: {text: new Date().toLocaleDateString()}}},
      {type: "text", data: {label: {text: "Hora"}, content: {text: new Date().toLocaleTimeString()}}},
      {type: "separator", data: {char: ".", length: 32}},
      {
        type: "table",
        data: {
          definition: {columns: [{name: "Item", width: 16}, {name: "Precio", width: 10, align: "right"}]},
          rows: [["Cafe", "$35.00"], ["Muffin", "$25.00"]],
          options: {header_bold: true}
        }
      },
      {type: "separator", data: {char: "-", length: 32}},
      {type: "text", data: {content: {text: "TOTAL:  $60.00", align: "right", content_style: {bold: true}}}},
      {type: "feed", data: {lines: 1}},
      {type: "text", data: {content: {text: "¡Gracias por su compra!", align: "center"}}},
      {type: "feed", data: {lines: 3}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  barcode: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {type: "text", data: {content: {text: "PRUEBA CÓDIGO BARRAS", align: "center", content_style: {bold: true}}}},
      {type: "feed", data: {lines: 1}},
      {
        type: "barcode",
        data: {symbology: "code128", data: "ABC123456", height: 60, hri_position: "below", align: "center"}
      },
      {type: "feed", data: {lines: 3}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  qr: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58, has_qr: true},
    commands: [
      {type: "text", data: {content: {text: "PRUEBA CÓDIGO QR", align: "center", content_style: {bold: true}}}},
      {type: "feed", data: {lines: 1}},
      {
        type: "qr",
        data: {
          data: "https://github.com/adcondev/ticket-daemon",
          human_text: "Escanear Docs",
          pixel_width: 150,
          align: "center"
        }
      },
      {type: "feed", data: {lines: 3}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  table: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {type: "text", data: {content: {text: "PRUEBA DE TABLA", align: "center", content_style: {bold: true}}}},
      {type: "separator", data: {char: "=", length: 32}},
      {
        type: "table",
        data: {
          definition: {
            columns: [{name: "ID", width: 4}, {name: "Prod", width: 14}, {
              name: "Cant",
              width: 4,
              align: "right"
            }]
          },
          rows: [["01", "Pieza A", "5"], ["02", "Pieza B", "10"], ["03", "Pieza X", "3"]],
          options: {header_bold: true, column_spacing: 1}
        }
      },
      {type: "separator", data: {char: "=", length: 32}},
      {type: "feed", data: {lines: 3}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  raw: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {type: "raw", data: {hex: "1B 40", comment: "Inicializar impresora", safe_mode: true}},
      {type: "text", data: {content: {text: "¡Comando RAW ejecutado!", align: "center"}}},
      {type: "beep", data: {times: 2, lapse: 1}},
      {type: "feed", data: {lines: 3}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  burstable: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {type: "beep", data: {times: 1, lapse: 1}}
    ]
  }
};
