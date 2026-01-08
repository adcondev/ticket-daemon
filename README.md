# 🎫 Ticket Daemon

A Windows Service that bridges Web POS applications with physical thermal printers via WebSocket.

## Features

- **WebSocket Server** on port 8766
- **Job Queue** with 100-slot buffer for burst traffic
- **Windows Service** integration via `go-svc`
- **File Logging** with automatic rotation
- **Poster Integration** for ESC/POS printing

## Quick Start

### Development Mode

```bash
# Build and run locally
task run

# Open test client
# Navigate to http://localhost:8766
```

### Windows Service Installation

```powershell
# Build for production
task build-prod

# Install service (run as Administrator)
task install-prod

# Start service
sc start TicketDaemon

# View logs
task logs
```

## WebSocket Protocol

### Endpoints

- `ws://localhost:8766/ws` - WebSocket endpoint
- `http://localhost:8766/health` - Health check

### Message Types

| Direction | Type     | Description             |
|-----------|----------|-------------------------|
| C → S     | `ticket` | Submit print job        |
| C → S     | `status` | Request queue status    |
| C → S     | `ping`   | Ping server             |
| S → C     | `ack`    | Job queued successfully |
| S → C     | `result` | Job completed/failed    |
| S → C     | `error`  | Validation error        |
| S → C     | `pong`   | Ping response           |

### Example: Submit Print Job

```json
{
  "tipo": "ticket",
  "id": "order-12345",
  "datos": {
    "version": "1.0",
    "profile": {
      "model": "80mm EC-PM-80250"
    },
    "commands": [
      {
        "type": "text",
        "data": {
          "content": {
            "text": "Hello!"
          }
        }
      },
      {
        "type": "cut",
        "data": {
          "mode": "partial"
        }
      }
    ]
  }
}
```

## Project Structure

```
ticket-daemon/
├── cmd/ticketd/          # Entry point
├── internal/
│   ├── daemon/           # Windows Service logic
│   └── server/           # WebSocket server
├── web/                  # Test client
└── Taskfile.yml          # Build tasks
```

## Configuration

Environment is set at build time:

| Environment | Listen Address   | Verbose | Service Name     |
|-------------|------------------|---------|------------------|
| `prod`      | `0.0.0.0:8766`   | false   | TicketDaemon     |
| `test`      | `localhost:8766` | true    | TicketDaemonTest |

## Logs

Logs are stored in:

- Production: `%PROGRAMDATA%\TicketDaemon\TicketDaemon.log`
- Test: `%PROGRAMDATA%\TicketDaemonTest\TicketDaemonTest.log`

## License

MIT © adcondev - RED 2000