/* ==============================================================
   CONFIGURACION
   ============================================================== */
// [GUIA DE INTEGRACION] URL del WebSocket. Puerto por defecto 8766.
const CONFIG = {
  WS_URL: `ws://${window.location.hostname}:${window.location.port || 8766}/ws`,
  HEALTH_URL: `http://${window.location.hostname}:${window.location.port || 8766}/health`,
  POLL_INTERVAL: 2000,
  MAX_LOGS: 200,
  RECONNECT_DELAY: 3000
};

/* ==============================================================
   PLANTILLAS (TEMPLATES)
   [GUIA DE INTEGRACION] Estructura requerida para los documentos de impresion
   ============================================================== */
const TEMPLATES = {
  simple: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {
        type: "text",
        data: {content: {text: "PRUEBA DE CONEXION", align: "center", content_style: {bold: true, size: "2x1"}}}
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
      {type: "text", data: {content: {text: "TOTAL: $60.00", align: "right", content_style: {bold: true}}}},
      {type: "feed", data: {lines: 1}},
      {type: "text", data: {content: {text: "Gracias por su compra!", align: "center"}}},
      {type: "feed", data: {lines: 3}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  barcode: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {type: "text", data: {content: {text: "PRUEBA CODIGO BARRAS", align: "center", content_style: {bold: true}}}},
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
    profile: {
      model: "58mm PT-210",
      paper_width: 58,
      code_table: "WPC1252",
      dpi: 203,
      has_qr: true
    },
    commands: [
      {
        type: "text",
        data: {
          content: {text: "P", align: "center", content_style: {bold: true, size: "4x4"}}
        }
      },
      {
        type: "text",
        data: {
          content: {text: "CENTRO PLAZA", align: "center", content_style: {bold: true, size: "1x1"}}
        }
      },
      {type: "text", data: {content: {text: "Av. Reforma 222, CDMX", align: "center"}}},
      {type: "feed", data: {lines: 1}},
      {type: "separator", data: {char: "-", length: 32}},
      {
        type: "table",
        data: {
          definition: {
            columns: [
              {name: "Concepto", width: 14, align: "left"},
              {name: "Dato", width: 16, align: "right"}
            ]
          },
          show_headers: false,
          rows: [
            ["Entrada:", "16/01 14:30"],
            ["Placa:", "XK-99-22"],
            ["Folio:", "#902102"]
          ],
          options: {column_spacing: 1}
        }
      },
      {type: "separator", data: {char: "-", length: 32}},
      {type: "feed", data: {lines: 1}},
      {
        type: "qr",
        data: {
          data: "https://pagos.centroplaza.com/t/902102",
          human_text: "ESCANEAR PARA PAGAR",
          pixel_width: 240,
          correction: "H",
          align: "center",
          logo: "PLACEHOLDER_IMAGEN_BASE64_LOGO_AQUI...",
          circle_shape: false
        }
      },
      {type: "feed", data: {lines: 1}},
      {
        type: "text",
        data: {
          content: {
            text: "¡Evita filas! Paga desde tu celular escaneando el código QR.",
            align: "center",
            content_style: {size: "1x1"}
          }
        }
      },
      {type: "feed", data: {lines: 1}},
      {
        type: "text",
        data: {
          content: {text: "Horario: 24 hrs", align: "center", content_style: {bold: true}}
        }
      },
      {type: "feed", data: {lines: 3}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  table: {
    version: "1.0",
    profile: {
      model: "58mm PT-210",
      paper_width: 58,
      code_table: "WPC1252",
      dpi: 203
    },
    commands: [
      {
        type: "text",
        data: {
          content: {text: "== ORDEN DE COCINA ==", align: "center", content_style: {bold: true, size: "1x1"}}
        }
      },
      {type: "separator", data: {char: "=", length: 32}},
      {
        type: "table",
        data: {
          definition: {
            columns: [
              {name: "No.", width: 4, align: "left"},
              {name: "ITEM", width: 18, align: "left"},
              {name: "PRECIO", width: 8, align: "right"}
            ]
          },
          show_headers: true,
          rows: [
            ["001", "Pizza Fam 16\"", "$250.00"],
            ["", " |_ Extra queso", "$30.00"],
            ["", " |_ Doble carne", "$50.00"],
            ["", " |_ Bebida 2L", "$25.00"],
            ["", "-------", ""],
            ["002", "Hamburguesa DX", "$120.00"],
            ["", " |_ Con tocino", "$20.00"],
            ["", " |_ Papas XL", "$35.00"],
            ["", "-------", ""],
            ["003", "Ensalada Caesar", "$85.00"],
            ["", " |_ Sin crutones", "$0.00"]
          ],
          options: {header_bold: true, word_wrap: true, column_spacing: 1}
        }
      },
      {type: "separator", data: {char: "-", length: 32}},
      {
        type: "table",
        data: {
          definition: {
            columns: [
              {name: "", width: 22, align: "left"},
              {name: "", width: 8, align: "right"}
            ]
          },
          show_headers: false,
          rows: [
            ["Subtotal:", "$615.00"],
            ["IVA (16%):", "$98.40"],
            ["TOTAL:", "$713.40"]
          ],
          options: {header_bold: false, column_spacing: 1}
        }
      },
      {type: "feed", data: {lines: 1}},
      {
        type: "barcode",
        data: {
          symbology: "code128",
          data: "2024011601",
          width: 2,
          height: 60,
          hri_position: "below",
          align: "center"
        }
      },
      {type: "feed", data: {lines: 2}},
      {type: "cut", data: {mode: "partial"}}
    ]
  },
  raw: {
    version: "1.0",
    profile: {model: "58mm PT-210", paper_width: 58},
    commands: [
      {type: "raw", data: {hex: "1B 40", comment: "Inicializar impresora", safe_mode: true}},
      {type: "text", data: {content: {text: "Comando RAW ejecutado!", align: "center"}}},
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

// Petición de lista de impresoras
function requestPrinters() {
  if (sendMessage({tipo: 'get_printers'})) {
    addLog('SENT', '🖨️ Requesting printer list');
  }
}