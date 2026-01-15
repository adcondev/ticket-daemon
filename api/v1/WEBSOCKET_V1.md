# Ticket Daemon - WebSocket API v1.0.0

## Descripción General

Ticket Daemon expone un servidor WebSocket para recibir trabajos de impresión desde aplicaciones POS y otros clientes. El protocolo es bidireccional y basado en mensajes JSON.

---

## Endpoints

| Protocolo | Endpoint                    | Descripción                      |
|-----------|-----------------------------|----------------------------------|
| WebSocket | `ws://{host}:8766/ws`       | Conexión principal para trabajos |
| HTTP GET  | `http://{host}:8766/health` | Health check (monitoreo)         |
| HTTP GET  | `http://{host}:8766/`       | Dashboard de diagnóstico (HTML)  |

### Configuración por Ambiente

| Ambiente       | Host Bind   | Puerto | Servicio Windows     |
|----------------|-------------|--------|----------------------|
| **Producción** | `0.0.0.0`   | 8766   | `TicketServicio`     |
| **Test/Dev**   | `localhost` | 8766   | `TicketServicioTest` |

---

## Protocolo WebSocket

### Ciclo de Vida de Conexión

```
┌─────────────────────────────────────────────────────────────────┐
│  1. Cliente conecta a ws://host:8766/ws                         │
│  2. Servidor envía mensaje "info" de bienvenida                 │
│  3. Cliente envía mensajes (ticket, status, ping)               │
│  4. Servidor responde (ack, result, error, pong, status)        │
│  5. Conexión se mantiene abierta hasta cierre explícito         │
└─────────────────────────────────────────────────────────────────┘
```

### Flujo de Trabajo de Impresión

```
Cliente                          Servidor                        Impresora
   │                                │                                │
   │──── ticket (job-001) ─────────>│                                │
   │                                │── Validar JSON                 │
   │                                │── Encolar trabajo              │
   │<─── ack (queued, pos: 1) ──────│                                │
   │                                │                                │
   │                                │── Worker toma trabajo          │
   │                                │───────── ESC/POS ─────────────>│
   │                                │<──────── OK ───────────────────│
   │<─── result (success) ──────────│                                │
   │                                │                                │
```

---

## Mensajes del Cliente → Servidor

### 1. `ticket` - Enviar Trabajo de Impresión

Encola un documento para impresión.

**Estructura:**

```json
{
  "tipo": "ticket",
  "id": "pos1-20260115-001",
  "datos": {
    "version": "1.0",
    "profile": { "model": "58mm PT-210", "paper_width": 58 },
    "commands": [...]
  }
}
```

| Campo   | Tipo   | Requerido | Descripción                                           |
|---------|--------|-----------|-------------------------------------------------------|
| `tipo`  | string | ✓         | Debe ser `"ticket"`                                   |
| `id`    | string |           | ID del trabajo (si se omite, el servidor genera UUID) |
| `datos` | object | ✓         | Documento de impresión (ver `document.schema.json`)   |

**Respuestas Posibles:**

| Escenario              | Respuesta                     |
|------------------------|-------------------------------|
| Trabajo encolado       | `ack` con status `"queued"`   |
| Campo `datos` faltante | `error` inmediato             |
| Cola llena             | `error` con mensaje de retry  |
| Impresión exitosa      | `result` con status `success` |
| Impresión fallida      | `result` con status `error`   |

---

### 2. `status` - Consultar Estado de Cola

Solicita estadísticas actuales de la cola de impresión.

**Estructura:**

```json
{
  "tipo": "status"
}
```

| Campo  | Tipo   | Requerido | Descripción         |
|--------|--------|-----------|---------------------|
| `tipo` | string | ✓         | Debe ser `"status"` |

---

### 3. `ping` - Verificar Conectividad

Envía un ping para verificar que el servidor responde.

**Estructura:**

```json
{
  "tipo": "ping",
  "id":  "ping-1736956800000"
}
```

| Campo  | Tipo   | Requerido | Descripción                            |
|--------|--------|-----------|----------------------------------------|
| `tipo` | string | ✓         | Debe ser `"ping"`                      |
| `id`   | string |           | ID opcional para correlacionar el pong |

---

## Mensajes del Servidor → Cliente

### 1. `info` - Mensaje Informativo

Enviado al conectar (bienvenida) o para notificaciones generales.

```json
{
  "tipo": "info",
  "status": "connected",
  "mensaje": "Connected to Ticket Daemon print server"
}
```

| Campo     | Tipo   | Descripción                  |
|-----------|--------|------------------------------|
| `tipo`    | string | Siempre `"info"`             |
| `status`  | string | Estado de conexión           |
| `mensaje` | string | Mensaje legible para humanos |

---

### 2. `ack` - Confirmación de Encolamiento

Indica que el trabajo fue aceptado y está en cola.

```json
{
  "tipo": "ack",
  "id":  "pos1-20260115-001",
  "status":  "queued",
  "current": 3,
  "capacity":  100,
  "mensaje":  "Job queued for printing"
}
```

| Campo      | Tipo    | Descripción                              |
|------------|---------|------------------------------------------|
| `tipo`     | string  | Siempre `"ack"`                          |
| `id`       | string  | ID del trabajo                           |
| `status`   | string  | Siempre `"queued"`                       |
| `current`  | integer | Trabajos actualmente en cola             |
| `capacity` | integer | Capacidad máxima de la cola              |
| `mensaje`  | string  | Confirmación legible                     |

> ⚠️ **Nota:** `ack` solo confirma encolamiento en servicio, NO impresión. Espera el mensaje `result` para confirmar que el Windows Spooler registró el trabajo.

---

### 3. `result` - Resultado de Impresión

Enviado cuando un trabajo termina de procesarse (éxito o error).

**Éxito:**

```json
{
  "tipo": "result",
  "id": "pos1-20260115-001",
  "status": "success",
  "mensaje": "Print completed in 245ms"
}
```

**Error:**

```json
{
  "tipo": "result",
  "id": "pos1-20260115-001",
  "status": "error",
  "mensaje": "PRINTER: Cannot connect - check if printer is installed"
}
```

| Campo     | Tipo   | Descripción                                  |
|-----------|--------|----------------------------------------------|
| `tipo`    | string | Siempre `"result"`                           |
| `id`      | string | ID del trabajo procesado                     |
| `status`  | string | `"success"` o `"error"`                      |
| `mensaje` | string | Descripción del resultado o detalle de error |

---

### 4. `error` - Error Inmediato

Enviado cuando hay un error de validación o la cola está llena (antes de encolar).

```json
{
  "tipo": "error",
  "id": "job-123",
  "status": "error",
  "mensaje": "Queue full, please retry in a few seconds"
}
```

| Campo     | Tipo   | Descripción                |
|-----------|--------|----------------------------|
| `tipo`    | string | Siempre `"error"`          |
| `id`      | string | ID del trabajo (si aplica) |
| `status`  | string | Siempre `"error"`          |
| `mensaje` | string | Descripción del error      |

**Errores Comunes:**

| Mensaje                                           | Causa                              |
|---------------------------------------------------|------------------------------------|
| `Field 'datos' is required for type 'ticket'`     | Falta el campo `datos`             |
| `Queue full, please retry in a few seconds`       | Cola saturada (100 trabajos)       |
| `Unknown message type: xxx`                       | Tipo de mensaje no reconocido      |

---

### 5. `pong` - Respuesta a Ping

```json
{
  "tipo": "pong",
  "id": "ping-1736956800000",
  "status": "ok"
}
```

| Campo    | Tipo   | Descripción                        |
|----------|--------|------------------------------------|
| `tipo`   | string | Siempre `"pong"`                   |
| `id`     | string | ID del ping original (si se envió) |
| `status` | string | Siempre `"ok"`                     |

---

### 6. `status` - Estado de Cola

```json
{
  "tipo": "status",
  "status": "ok",
  "current": 5,
  "capacity": 100,
  "mensaje": "Queue: 5/100"
}
```

| Campo      | Tipo    | Descripción                    |
|------------|---------|--------------------------------|
| `tipo`     | string  | Siempre `"status"`             |
| `status`   | string  | Siempre `"ok"`                 |
| `current`  | integer | Trabajos en cola               |
| `capacity` | integer | Capacidad máxima               |
| `mensaje`  | string  | Estado formateado              |

---

## HTTP Endpoints

### GET `/health`

Endpoint de health check para sistemas de monitoreo.

**Request:**

```http
GET /health HTTP/1.1
Host: localhost:8766
```

**Response:**

```json
{
  "status": "ok",
  "queue": {
    "current": 2,
    "capacity": 100,
    "utilization": 2.0
  },
  "worker": {
    "running": true,
    "jobs_processed": 1547,
    "jobs_failed": 3
  },
  "build": {
    "env": "prod",
    "date": "2026-01-15",
    "time": "10:30:00"
  },
  "uptime_seconds": 86400
}
```

**Headers de Respuesta:**

```
Content-Type: application/json
Access-Control-Allow-Origin: *
```

---

## Categorías de Error

Los mensajes de error en `result` siguen un formato prefijado para facilitar el parsing:

| Prefijo       | Descripción                       | Ejemplos                                       |
|---------------|-----------------------------------|------------------------------------------------|
| `VALIDATION:` | Error de estructura del documento | Missing 'version' field                        |
| `PRINTER:`    | Error de conexión con impresora   | Cannot connect - check if printer is installed |
| `QR:`         | Error generando código QR         | Data cannot be empty                           |
| `BARCODE:`    | Error en código de barras         | Symbology type is required                     |
| `TABLE:`      | Error en renderizado de tabla     | Columns exceed paper width                     |
| `RAW:`        | Error en comando raw              | Blocked by safe_mode                           |
| `IMAGE:`      | Error procesando imagen           | Invalid or corrupted base64 data               |
| `COMMAND:`    | Error de comando desconocido      | Unknown command type                           |
| `JSON:`       | Error de parsing JSON             | Invalid document structure                     |
| `EXECUTION:`  | Error durante ejecución           | (varios)                                       |
| `ERROR:`      | Error genérico                    | (fallback)                                     |

---

## Ejemplos de Integración

### JavaScript (Browser)

```javascript
const ws = new WebSocket('ws://localhost:8766/ws');

ws.onopen = () => {
  console.log('Conectado');
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  
  switch (msg.tipo) {
    case 'info':
      console.log('Bienvenida:', msg.mensaje);
      break;
    case 'ack':
      console.log(`Trabajo ${msg.id} encolado (${msg.current}/${msg.capacity})`);
      break;
    case 'result':
      if (msg.status === 'success') {
        console.log(`✅ ${msg.id}: ${msg.mensaje}`);
      } else {
        console.error(`❌ ${msg.id}: ${msg.mensaje}`);
      }
      break;
    case 'error':
      console.error('Error:', msg.mensaje);
      break;
  }
};

// Enviar trabajo de impresión
function print(document) {
  const message = {
    tipo: 'ticket',
    id: `job-${Date.now()}`,
    datos: document
  };
  ws.send(JSON.stringify(message));
}

// Ejemplo de documento
print({
  version: '1.0',
  profile: { model: '58mm PT-210', paper_width: 58 },
  commands: [
    { type: 'text', data: { content: { text: 'Hola Mundo', align: 'center' } } },
    { type: 'cut', data: { mode: 'partial' } }
  ]
});
```

### Go

```go
package main

import (
	"context"
	"encoding/json"
	"log"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Message struct {
	Tipo  string          `json:"tipo"`
	ID    string          `json:"id,omitempty"`
	Datos json.RawMessage `json:"datos,omitempty"`
}

type Response struct {
	Tipo    string `json:"tipo"`
	ID      string `json:"id,omitempty"`
	Status  string `json:"status,omitempty"`
	Mensaje string `json:"mensaje,omitempty"`
}

func main() {
	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, "ws://localhost:8766/ws", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enviar trabajo
	doc := `{"version":"1.0","profile":{"model":"58mm PT-210"},"commands":[{"type":"beep","data":{"times":1}}]}`
	msg := Message{
		Tipo:  "ticket",
		ID:    "go-job-001",
		Datos: json.RawMessage(doc),
	}

	if err := wsjson.Write(ctx, conn, msg); err != nil {
		log.Fatal(err)
	}

	// Leer respuestas
	for {
		var resp Response
		if err := wsjson.Read(ctx, conn, &resp); err != nil {
			break
		}
		log.Printf("[%s] %s: %s", resp.Tipo, resp.ID, resp.Mensaje)

		if resp.Tipo == "result" {
			break
		}
	}
}
```

### cURL (Health Check)

```bash
curl -s http://localhost:8766/health | jq .
```

---

## Límites y Consideraciones

| Parámetro                | Valor   | Descripción                                    |
|--------------------------|---------|------------------------------------------------|
| Capacidad de cola        | 100     | Trabajos máximos en espera                     |
| Timeout de notificación  | 5s      | Tiempo máximo para notificar resultado         |
| Reconexión recomendada   | 3s      | Delay sugerido antes de reconectar             |
| Tamaño máximo de mensaje | ~10MB   | Limitado por memoria (imágenes base64)         |

### Backpressure

Cuando la cola está llena (100 trabajos), el servidor rechaza inmediatamente nuevos trabajos con un mensaje `error`:

```json
{
  "tipo": "error",
  "id": "job-xxx",
  "status": "error",
  "mensaje": "Queue full, please retry in a few seconds"
}
```

**Recomendación:** Implementar retry con backoff exponencial (1s, 2s, 4s...).

---

## Versionado

- **Protocolo actual:** v1.0
- **Documento de impresión:** v1.0 (ver `document.schema.json`)
- **Compatibilidad:** El servidor valida `version` en el documento pero actualmente solo soporta `1.x`