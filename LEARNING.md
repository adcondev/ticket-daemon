# 📚 Ticket Daemon - Technical Summary

---

## 🎯 Project Overview

**Ticket Daemon** is a production-grade **Windows Service** designed to bridge web-based Point of Sale (POS)
applications with physical thermal printers. The service acts as a middleware layer that:

- Receives print job documents via **WebSocket** connections from multiple web clients
- Queues and serializes jobs for ordered execution
- Renders and sends ESC/POS commands to thermal printers via the Windows Print Spooler
- Provides real-time feedback to clients about job status

This project demonstrates enterprise-level software engineering for retail/POS environments, combining network
programming, concurrent systems design, and Windows system integration.

---

## 🛠️ Tech Stack and Key Technologies

### Programming Language

| Technology      | Version | Usage                        |
|-----------------|---------|------------------------------|
| **Go (Golang)** | 1.24+   | Primary development language |

### Frameworks & Platforms

| Technology           | Purpose                                               |
|----------------------|-------------------------------------------------------|
| **Windows Services** | Native OS integration for background daemon execution |
| **ESC/POS Protocol** | Thermal printer command language                      |
| **WebSocket**        | Real-time bidirectional communication protocol        |
| **HTTP/REST**        | Health check and static file serving endpoints        |

### Build & Automation Tools

| Tool               | Purpose                                     |
|--------------------|---------------------------------------------|
| **Task (go-task)** | Build automation and service management     |
| **sc.exe**         | Windows Service Control Manager integration |
| **PowerShell**     | Log management and system automation        |

### Development Environment

| Component        | Details                                             |
|------------------|-----------------------------------------------------|
| **OS Target**    | Windows 10/11, Windows Server                       |
| **Architecture** | x86_64                                              |
| **Build Flags**  | `-ldflags` for compile-time configuration injection |

---

## 📦 Notable Libraries

| Library                          | Purpose                  | Problem Solved                                                                           |
|----------------------------------|--------------------------|------------------------------------------------------------------------------------------|
| `github.com/judwhite/go-svc`     | Windows Service wrapper  | Abstracts Windows SCM integration, signal handling, and service lifecycle management     |
| `nhooyr.io/websocket`            | WebSocket implementation | Production-ready WebSocket server with JSON marshaling support                           |
| `github.com/google/uuid`         | UUID generation          | Generates unique identifiers for print jobs when clients don't provide IDs               |
| `github.com/adcondev/poster`     | ESC/POS rendering engine | Custom library for document parsing, command composition, and printer profile management |
| `github.com/yeqown/go-qrcode/v2` | QR code generation       | Generates QR codes for receipts (invoices, verification links)                           |
| `github.com/fogleman/gg`         | 2D graphics              | Image rendering for receipt logos and graphics                                           |

---

## 🏆 Major Achievements and Skills Demonstrated

### System Architecture & Design

- **Designed a producer-consumer architecture** for handling concurrent print requests from multiple POS terminals
- **Implemented a buffered job queue** (100 slots) with non-blocking enqueue and backpressure handling
- **Created a modular service architecture** separating WebSocket handling, job processing, and printer integration

### Concurrent Programming

- **Developed a single-worker goroutine model** that serializes printer access while supporting concurrent client
  connections
- **Implemented thread-safe client registry** using `sync.RWMutex` for managing active WebSocket connections
- **Designed graceful shutdown mechanisms** with `sync.WaitGroup` and context cancellation

### Windows System Integration

- **Built a native Windows Service** using the `go-svc` library with proper lifecycle management (Init, Start, Stop)
- **Implemented structured logging** to `%PROGRAMDATA%` with automatic log rotation (5MB threshold, keeps 1000 lines)
- **Created build-time environment configuration** for production vs. test deployments

### Network Programming

- **Implemented a WebSocket server** with connection lifecycle management and client broadcasting capabilities
- **Designed a JSON-based messaging protocol** with typed messages (ticket, status, ping) and responses (ack, result,
  error)
- **Built health check endpoints** returning JSON status for monitoring integration

### API Design

- **Defined a structured document schema** for print jobs with support for text, images, tables, barcodes, and QR codes
- **Implemented printer profile system** supporting multiple thermal printer models (58mm, 80mm)
- **Created dynamic document executor** that interprets JSON commands into ESC/POS byte sequences

### DevOps & Tooling

- **Configured Taskfile.yml automation** for build, deploy, install, and service management operations
- **Implemented multi-environment builds** (prod/test) with compile-time flag injection
- **Created embedded HTML test client** for debugging and manual testing

### Error Handling & Resilience

- **Designed comprehensive error propagation** from printer errors back to originating clients
- **Implemented queue overflow protection** with immediate feedback to clients
- **Built filtered logging system** that reduces verbosity in production mode

### Windows System Integration

- **Built a native Windows Service** using the `go-svc` library with proper lifecycle management (Init, Start, Stop)
- **Implemented structured logging** to `%PROGRAMDATA%` with automatic log rotation (5MB threshold, keeps 1000 lines)
- **Created build-time environment configuration** for production vs. test deployments
- **Implemented printer discovery** using Windows `EnumPrinters` API via syscall, with automatic classification of
  thermal vs. virtual printers

---

## 💡 Skills Gained/Reinforced

### Core Technical Skills

- [x] **Go (Golang) Development** - Idiomatic Go patterns, package organization, error handling
- [x] **Concurrent Programming** - Goroutines, channels, mutexes, wait groups
- [x] **WebSocket Protocol** - Bidirectional real-time communication
- [x] **Windows Service Development** - SCM integration, system-level programming
- [x] **Interface Segregation** - Decoupling packages (Server/Worker) using interface abstraction (`ClientNotifier`) to
  prevent circular dependencies.
- [x] **Go Embed** - Bundling static HTML/JS assets into the binary for a single-file distribution.
- [x] **Context Management** - Using `context.WithTimeout` for controlling I/O deadlines during client notifications.
- [x] **Windows API Integration** - Direct syscall to `winspool.drv` for `EnumPrinters`, `GetDefaultPrinter`
- [x] **Caching Patterns** - Implemented thread-safe cache with TTL and double-check locking optimization

### Architecture & Design

- [x] **Producer-Consumer Pattern** - Job queuing and worker-based processing
- [x] **Modular Architecture** - Separation of concerns across packages
- [x] **Protocol Design** - JSON messaging schema and response types
- [x] **Non-Blocking Channel Operations** - Implementing traffic shaping using `select` with a `default` case to handle
  queue saturation instantly.

### DevOps & Operations

- [x] **Build Automation** - Taskfile/Makefile patterns for Go projects
- [x] **Service Management** - Windows service installation, control, monitoring
- [x] **Logging Best Practices** - Structured logging, rotation, filtering

### Domain Knowledge

- [x] **POS Systems** - Point of Sale terminal integration patterns
- [x] **Thermal Printing** - ESC/POS command language and printer profiles
- [x] **Retail Software** - Multi-client, high-availability service requirements

---

## 📝 CV-Ready Bullet Points

> Copy-paste ready descriptions for resume/CV:

- Designed and implemented a **Windows Service** in Go for real-time thermal printer integration, handling concurrent
  WebSocket connections from multiple POS terminals
- Built a **producer-consumer job queue** with backpressure handling, supporting 100+ concurrent print requests with
  ordered serial execution
- Developed a **JSON-based document schema** and executor for ESC/POS thermal printer commands including text, images,
  barcodes, and QR codes
- Implemented **graceful shutdown** and **automatic log rotation** for production-grade service reliability
- Created **multi-environment build system** with compile-time configuration injection using Go build flags
- Integrated with Windows Print Spooler via custom Go library for **cross-printer-model compatibility**
- Implemented **non-blocking backpressure** mechanisms using Go channels to protect the service from traffic spikes,
  providing immediate feedback to clients when queues are full.
- Decoupled architecture using **Dependency Injection** interfaces, allowing independent testing of the worker logic
  without a live WebSocket server.
- Developed a **self-contained binary** using `embed`, bundling the diagnostic dashboard UI directly into the Windows
  Service executable.
- Implemented **Windows printer discovery** using direct syscall to `EnumPrinters` API with intelligent classification
  of thermal vs. virtual printers and 30-second caching for performance
