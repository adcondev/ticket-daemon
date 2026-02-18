// Package daemon contiene la lógica del servicio de Windows.
package daemon

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Log configuration
const maxLogSize = 5 * 1024 * 1024 // 5MB

// Logger state
var (
	logConfig    = struct{ Verbose bool }{Verbose: true}
	logConfigMux sync.RWMutex
	logFilePath  string
	logFile      *os.File
	logFileMu    sync.Mutex // Protege operaciones de archivo (write, flush, rotate)
)

// Non-critical prefixes (filtered when verbose=false)
var nonCriticalPrefixes = []string{
	"[>] Job enviado",
	"[i] Iniciando escucha",
	"[i] Terminando escucha",
	"[+] Cliente conectado",
	"[-] Cliente desconectado",
	"[~] Queue status",
}

// FilteredLogger implements io.Writer with filtering
type FilteredLogger struct{}

// Write filters log messages based on verbosity
func (l *FilteredLogger) Write(p []byte) (n int, err error) {
	logConfigMux.RLock()
	verbose := logConfig.Verbose
	logConfigMux.RUnlock()

	if !verbose {
		msg := string(p)
		for _, prefix := range nonCriticalPrefixes {
			if strings.Contains(msg, prefix) {
				return len(p), nil // Discard silently
			}
		}
	}

	// Usar mutex global
	logFileMu.Lock()
	defer logFileMu.Unlock()

	if logFile == nil {
		return 0, fmt.Errorf("log file not initialized")
	}
	return logFile.Write(p)
}

// InitLogger initializes the file logger with rotation
func InitLogger(path string, verbose bool) error {
	logFileMu.Lock()
	defer logFileMu.Unlock()

	logFilePath = path

	// Set verbosity
	logConfigMux.Lock()
	logConfig.Verbose = verbose
	logConfigMux.Unlock()

	// Auto-rotate if exceeds 5MB
	if err := rotateLogIfNeeded(path); err != nil {
		fmt.Printf("[!] Error en rotación de logs: %v\n", err)
	}

	// Open log file
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600) //nolint:gosec
	if err != nil {
		return err
	}
	logFile = f

	// Use filtered logger
	log.SetOutput(&FilteredLogger{})
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	return nil
}

// SetVerbose changes the verbosity level at runtime
func SetVerbose(v bool) {
	logConfigMux.Lock()
	logConfig.Verbose = v
	logConfigMux.Unlock()
	log.Printf("[OK] Verbosidad de logs: %v", v)
}

// GetVerbose returns current verbosity level
func GetVerbose() bool {
	logConfigMux.RLock()
	defer logConfigMux.RUnlock()
	return logConfig.Verbose
}

// GetLogFileSize returns current log file size
func GetLogFileSize() int64 {
	if logFilePath == "" {
		return 0
	}
	info, err := os.Stat(logFilePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// FlushLogFile keeps last 50 lines and clears the rest
func FlushLogFile() error {
	logFileMu.Lock()
	defer logFileMu.Unlock()

	if logFilePath == "" {
		return fmt.Errorf("ruta de log no configurada")
	}

	lines := readLastNLines(logFilePath, 50)
	content := ""
	if len(lines) > 0 {
		content = strings.Join(lines, "\n") + "\n"
	}

	// ningún Write() puede ocurrir
	if logFile != nil {
		err := logFile.Close()
		if err != nil {
			return err
		}
	}

	if err := os.WriteFile(logFilePath, []byte(content), 0600); err != nil {
		return err
	}

	f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600) //nolint:gosec
	if err != nil {
		return err
	}

	logFile = f
	// FilteredLogger usa la variable global
	log.Println("[OK] Logs limpiados")

	return nil
}

// rotateLogIfNeeded rotates log if exceeds max size
func rotateLogIfNeeded(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Size() < maxLogSize {
		return nil
	}

	// Rotate:  keep last 1000 lines
	lines := readLastNLines(path, 1000)
	if len(lines) == 0 {
		return nil
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0600)
}

// readLastNLines reads last N lines from file
func readLastNLines(path string, n int) []string {
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return []string{}
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Panicf("[!] Error cerrando archivo de log: %v", err)
		}
	}(file)

	stat, err := file.Stat()
	if err != nil {
		return []string{}
	}

	size := stat.Size()
	if size == 0 {
		return []string{}
	}

	// Read last 64KB max
	bufSize := int64(64 * 1024)
	if size < bufSize {
		bufSize = size
	}

	buf := make([]byte, bufSize)
	_, err = file.Seek(size-bufSize, io.SeekStart)
	if err != nil {
		return []string{}
	}

	_, err = file.Read(buf)
	if err != nil {
		return []string{}
	}

	allLines := strings.Split(string(buf), "\n")

	// Clean empty lines at end
	for len(allLines) > 0 && allLines[len(allLines)-1] == "" {
		allLines = allLines[:len(allLines)-1]
	}

	// If we started mid-line, discard first partial line
	if size > bufSize && len(allLines) > 0 {
		allLines = allLines[1:]
	}

	if len(allLines) <= n {
		return allLines
	}
	return allLines[len(allLines)-n:]
}
