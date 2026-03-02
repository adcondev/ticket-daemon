# 📚 Ticket Daemon - Technical Summary

---

## 🎯 Project Overview

**Ticket Daemon** is a production-grade **Windows Service** built in Go that bridges web-based Point of Sale (POS)
applications with physical thermal printers. It receives structured JSON documents via WebSocket, queues and serializes
them through a producer-consumer pipeline, renders ESC/POS commands via the custom **Poster** library, and delivers
real-time feedback to clients about job status — all while running as a background daemon managed by the Windows
Service Control Manager.

---

## 🛠️ Tech Stack & Infrastructure

### Language & Runtime

| Technology      | Version | Usage                                              |
|-----------------|---------|----------------------------------------------------|
| **Go (Golang)** | 1.24+   | Primary language — backend, service, and dashboard |

### Protocols & Platforms

| Technology           | Purpose                                                   |
|----------------------|-----------------------------------------------------------|
| **Windows Services** | Native SCM integration for background daemon execution    |
| **ESC/POS Protocol** | Thermal printer command language for POS hardware         |
| **WebSocket**        | Real-time bidirectional communication with POS clients    |
| **HTTP**             | Health check endpoint, static dashboard, login/auth flows |

### CI/CD & DevOps

| Tool / Service             | Purpose                                                         |
|----------------------------|-----------------------------------------------------------------|
| **GitHub Actions**         | Multi-job CI pipeline (test, lint, build, benchmarks)           |
| **CodeQL**                 | Static Application Security Testing (SAST) on every PR and push |
| **Codecov**                | Test coverage tracking and reporting                            |
| **golangci-lint**          | 16+ linters (govet, errcheck, staticcheck, gosec, etc.)         |
| **Semantic PR Validation** | Enforces conventional commit prefixes on PR titles              |
| **PR Automation**          | Auto-labeling (size/XS–XL), conflict detection, auto-assignment |
| **Task (go-task)**         | Build automation with `.env` loading and ldflags injection      |
| **PR Status Dashboard**    | Scheduled weekly report of open PRs needing review              |

### Development Environment

| Component        | Details                                                            |
|------------------|--------------------------------------------------------------------|
| **OS Target**    | Windows 10/11, Windows Server                                      |
| **Architecture** | x86_64 (cross-compiled from Linux in CI via `GOOS=windows`)        |
| **Build Flags**  | `-ldflags` for compile-time injection of secrets and configuration |

---

## 📦 Notable Libraries

| Library                       | Purpose                  | Problem Solved                                                                           |
|-------------------------------|--------------------------|------------------------------------------------------------------------------------------|
| `github.com/coder/websocket`  | WebSocket implementation | Production-ready WebSocket server with JSON marshaling (`wsjson`) and graceful close     |
| `github.com/judwhite/go-svc`  | Windows Service wrapper  | Abstracts Windows SCM integration, signal handling, and service lifecycle management     |
| `github.com/google/uuid`      | UUID generation          | Generates unique identifiers for print jobs when clients don't provide IDs               |
| `golang.org/x/crypto`         | Cryptographic utilities  | Bcrypt password hashing for dashboard authentication with constant-time comparison       |
| `github.com/adcondev/poster`  | ESC/POS rendering engine | Custom library for document parsing, command composition, and printer profile management |
| `github.com/yeqown/go-qrcode` | QR code generation       | Generates QR codes for receipts (invoices, payment links, verification URLs)             |
| `github.com/fogleman/gg`      | 2D graphics              | Image rendering and dithering for receipt logos and graphics on thermal paper            |

---

## 🏆 CV-Ready Achievements

> Copy-paste ready bullet points for resume/CV — each begins with a strong action verb.

- **Architected** a producer-consumer Windows Service in Go that bridges WebSocket-connected POS terminals with thermal
  printers, processing concurrent print requests through a buffered channel queue (50–100 slots) with non-blocking
  backpressure handling and ordered serial execution.

- **Engineered** a dual-layer authentication system combining per-message token validation on WebSocket submissions with
  bcrypt-hashed session-based dashboard login, including brute-force protection (IP lockout after 5 failed attempts) and
  cryptographically random 256-bit session tokens with automatic expiry cleanup.

- **Developed** a comprehensive JSON-based document schema and ESC/POS executor supporting 11 command types (text,
  image, barcode, QR, table, separator, feed, cut, raw, pulse, beep) with full printer profile management across
  multiple thermal printer models (58mm/80mm).

- **Implemented** a robust CI/CD pipeline on GitHub Actions encompassing unit tests with race detection, Codecov
  coverage reporting, golangci-lint with 16+ linters, CodeQL SAST scanning, performance benchmark comparison on PRs, and
  cross-compiled Windows builds via Taskfile automation.

- **Optimized** printer discovery by implementing a thread-safe cache with 30-second TTL and double-check locking
  pattern, backed by Windows `EnumPrinters` syscall integration with intelligent classification of thermal vs. virtual
  printers.

- **Built** a self-contained single-binary deployment using Go's `embed` package to bundle the diagnostic dashboard
  (HTML/CSS/JS) directly into the service executable, with Go template injection for runtime token configuration.

- **Designed** a per-client rate limiter (30 jobs/minute) using a sliding-window algorithm to protect the print queue
  from traffic spikes, combined with structured audit logging for security events (login attempts, token rejections,
  rate
  limit violations).

---

## 💡 Skills Demonstrated

Concurrent Programming, WebSocket Protocol Design, Windows Service Development, Producer-Consumer Architecture,
Go Channels & Goroutines, Thread-Safe Data Structures (sync.RWMutex), Graceful Shutdown Patterns, Bcrypt Authentication,
Session Management, Rate Limiting, ESC/POS Thermal Printing, JSON Schema Design, API Documentation,
CI/CD Pipeline Design (GitHub Actions), Static Analysis & SAST (CodeQL, golangci-lint), Code Coverage (Codecov),
Go Embed, Compile-Time Configuration Injection (ldflags), Structured Logging with Rotation, Interface Segregation /
Dependency Injection, Windows API Syscall Integration, Build Automation (Taskfile)
