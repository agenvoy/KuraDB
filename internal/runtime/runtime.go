package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

const (
	stopGraceWindow = 5 * time.Second
	stopPollGap     = 100 * time.Millisecond
)

var ErrAlreadyRunning = errors.New("kuradb daemon already running")

type Runtime struct {
	UID       string `json:"uid"`
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
}

func Path(configDir string) string {
	return filepath.Join(configDir, "runtime.uid")
}

func Read(configDir string) (*Runtime, error) {
	r, err := go_pkg_filesystem.ReadJSON[Runtime](Path(configDir))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func write(configDir string, r *Runtime) error {
	if err := go_pkg_filesystem.WriteJSON(Path(configDir), r, true); err != nil {
		return fmt.Errorf("go_pkg_filesystem.WriteJSON: %w", err)
	}
	return nil
}

func IsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if pid == os.Getpid() {
		return true
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func Init(configDir string) (*Runtime, error) {
	if existing, err := Read(configDir); err == nil && existing != nil {
		if existing.PID != os.Getpid() && IsAlive(existing.PID) {
			return existing, ErrAlreadyRunning
		}
	}
	r := &Runtime{
		UID:       go_pkg_utils.UUID(),
		PID:       os.Getpid(),
		StartedAt: time.Now().Format(time.RFC3339),
	}
	if err := write(configDir, r); err != nil {
		return nil, err
	}
	return r, nil
}

func Stop(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("os.FindProcess: %w", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("proc.SIGTERM: %w", err)
	}
	deadline := time.Now().Add(stopGraceWindow)
	for time.Now().Before(deadline) {
		if !IsAlive(pid) {
			return nil
		}
		time.Sleep(stopPollGap)
	}
	if err := proc.Signal(syscall.SIGKILL); err != nil {
		return fmt.Errorf("proc.SIGKILL: %w", err)
	}
	return nil
}

func Clear(configDir string) error {
	p := Path(configDir)
	if !go_pkg_filesystem_reader.Exists(p) {
		return nil
	}
	if err := os.Remove(p); err != nil {
		return fmt.Errorf("os.Remove: %w", err)
	}
	return nil
}

func IsCurrent(configDir string) bool {
	r, err := Read(configDir)
	if err != nil || r == nil {
		return false
	}
	return IsAlive(r.PID)
}