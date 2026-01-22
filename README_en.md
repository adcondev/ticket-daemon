# 🎫 Ticket Daemon

**Ticket Daemon** is a Windows Service that bridges Web POS applications with physical thermal printers via WebSocket.

## ✨ Features

- 🔌 **WebSocket Server** on port 8766
- 📦 **Job Queue** with 100-slot buffer for burst traffic
- 🪟 **Windows Service** with native integration
- 📝 **File Logging** with automatic rotation
- 🖨️ **Poster Integration** for ESC/POS printing
- 🌐 **Embedded Test Client** - no external files needed

---

## 🚀 Quick Start

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Task](https://taskfile.dev/installation/) (task runner)
- Windows 10/11 or Windows Server

### Installation

```powershell
# Clone the repository
git clone https://github.com/adcondev/ticket-daemon.git
cd ticket-daemon

# See available commands
task
```

### Development Mode

```powershell
# Build and run locally (console mode)
task run

# Open test client in browser
task open
# Or navigate to:  http://localhost:8766
```

### Install as Windows Service

```powershell
# ⚠️ Run PowerShell as Administrator

# Install and start service
task install

# Check status
task status

# View logs
task logs
```

---

## 📋 Available Commands

Run `task` to see all commands:

| Command          | Description                 |
|------------------|-----------------------------|
| `task build`     | Build the service binary    |
| `task run`       | Build and run locally       |
| `task install`   | Install as Windows Service  |
| `task uninstall` | Remove Windows Service      |
| `task start`     | Start the service           |
| `task stop`      | Stop the service            |
| `task restart`   | Restart the service         |
| `task status`    | Check service status        |
| `task logs`      | View logs in real-time      |
| `task health`    | Check health endpoint       |
| `task open`      | Open test client in browser |
| `task clean`     | Remove build artifacts      |

---

## 📡 WebSocket Protocol

### Endpoints

| Endpoint                       | Description          |
|--------------------------------|----------------------|
| `ws://localhost:8766/ws`       | WebSocket connection |
| `http://localhost:8766/health` | Health check (JSON)  |
| `http://localhost:8766/`       | Embedded test client |

### Message Types

| Direction       | Type           | Description             |
|-----------------|----------------|-------------------------|
| Client → Server | `ticket`       | Submit print job        |
| Client → Server | `status`       | Request queue status    |
| Client → Server | `ping`         | Ping server             |
| Client → Server | `get_printers` | List installed printers |
| Server → Client | `ack`          | Job queued              |
| Server → Client | `result`       | Job completed/failed    |
| Server → Client | `error`        | Validation error        |
| Server → Client | `printers`     | List of printers        |

### Example:  Print a Ticket

```json
{
  "tipo": "ticket",
  "id": "order-12345",
  "datos": {
    "version": "1.0",
    "profile": {
      "model": "80mm EC-PM-80250",
      "paper_width": 80
    },
    "commands": [
      {
        "type": "text",
        "data": {
          "content": {
            "text":  "Hello World! ",
            "content_style": { "bold": true, "size": "2x2" },
            "align": "center"
          }
        }
      },
      {
        "type": "cut",
        "data":  { "mode": "partial" }
      }
    ]
  }
}
```

---

## ⚙️ Configuration

Configuration is set at **build time**.  No config files needed.

| Environment | Build Command     | Port | Interface | Verbose |
|-------------|-------------------|------|-----------|---------|
| Test        | `task build`      | 8766 | localhost | Yes     |
| Production  | `task build-prod` | 8766 | 0.0.0.0   | No      |

To change defaults, edit `internal/daemon/program.go` and rebuild.

---

## 📂 Project Structure

```
ticket-daemon/
├── api/v1/                 # API documentation & JSON schemas
├── cmd/TicketServicio/     # Entry point
├── examples/json/          # Example print documents
├── internal/
│   ├── assets/web/         # Embedded dashboard client
│   ├── daemon/             # Service logic, logging & printer discovery
│   ├── printer/            # Printer type definitions
│   ├── server/             # WebSocket server
│   └── worker/             # Print worker (Poster)
├── Taskfile.yml            # Build & service management
└── README.md
```

---

## 📝 Logs

| Environment | Log Location                                              |
|-------------|-----------------------------------------------------------|
| Test        | `%PROGRAMDATA%\TicketServicioTest\TicketServicioTest.log` |
| Production  | `%PROGRAMDATA%\TicketServicio\TicketServicio.log`         |

Logs auto-rotate at 5MB, keeping the last 1000 lines.

---

## 🔗 Related Projects

- [Poster](https://github.com/adcondev/poster) - ESC/POS print engine
- [Scale Daemon](https://github.com/adcondev/scale-daemon) - Serial scale WebSocket service

---

## 📄 License

MIT © adcondev - RED 2000
