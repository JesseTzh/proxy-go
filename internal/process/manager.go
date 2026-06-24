package process

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const defaultLogLimitBytes = 64 * 1024

type ManagedProcess struct {
	Name          string
	Path          string
	Args          []string
	Dir           string
	LogLimitBytes int

	mu        sync.Mutex
	cmd       *exec.Cmd
	done      chan error
	startedAt *time.Time
	lastError string
	log       *boundedLog
}

func (p *ManagedProcess) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		slog.Info("managed process already running", "name", p.Name, "path", p.Path)
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	started := time.Now()
	slog.Info("managed process starting", "name", p.Name, "path", p.Path, "args", p.Args, "dir", p.Dir)
	cmd := exec.Command(p.Path, p.Args...)
	cmd.Dir = p.Dir
	p.ensureLogLocked()
	cmd.Stdout = io.MultiWriter(os.Stdout, p.log)
	cmd.Stderr = io.MultiWriter(os.Stderr, p.log)
	if err := cmd.Start(); err != nil {
		p.lastError = err.Error()
		p.log.append("start failed: " + err.Error() + "\n")
		slog.Warn("managed process start failed", "name", p.Name, "path", p.Path, "error", err)
		return err
	}
	now := time.Now()
	p.startedAt = &now
	p.cmd = cmd
	p.done = make(chan error, 1)
	slog.Info("managed process started", "name", p.Name, "pid", cmd.Process.Pid, "elapsed", time.Since(started).String())
	done := p.done
	go func() {
		err := cmd.Wait()
		done <- err
		close(done)
		p.mu.Lock()
		defer p.mu.Unlock()
		if err != nil {
			p.lastError = err.Error()
			p.log.append("process exited: " + err.Error() + "\n")
			slog.Warn("managed process exited", "name", p.Name, "error", err)
		}
		p.cmd = nil
		p.done = nil
	}()
	return nil
}

func (p *ManagedProcess) Stop(ctx context.Context) error {
	p.mu.Lock()
	cmd := p.cmd
	done := p.done
	p.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		slog.Info("managed process already stopped", "name", p.Name)
		return nil
	}
	started := time.Now()
	slog.Info("managed process stopping", "name", p.Name, "pid", cmd.Process.Pid)
	_ = cmd.Process.Signal(os.Interrupt)
	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		slog.Warn("managed process stop timed out; killed", "name", p.Name, "elapsed", time.Since(started).String())
		return ctx.Err()
	case err := <-done:
		if errors.Is(err, exec.ErrNotFound) {
			return nil
		}
		slog.Info("managed process stopped", "name", p.Name, "elapsed", time.Since(started).String(), "waitError", err)
		return nil
	}
}

func (p *ManagedProcess) Restart(ctx context.Context) error {
	slog.Info("managed process restarting", "name", p.Name)
	if err := p.Stop(ctx); err != nil {
		return err
	}
	return p.Start(ctx)
}

func (p *ManagedProcess) Status() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	running := p.cmd != nil && p.cmd.Process != nil
	return map[string]any{"name": p.Name, "path": p.Path, "running": running, "startedAt": p.startedAt, "lastError": p.lastError}
}

func (p *ManagedProcess) Logs() []string {
	p.mu.Lock()
	p.ensureLogLocked()
	log := p.log
	p.mu.Unlock()
	return log.lines()
}

func (p *ManagedProcess) ensureLogLocked() {
	limit := p.LogLimitBytes
	if limit <= 0 {
		limit = defaultLogLimitBytes
	}
	if p.log == nil || p.log.limit != limit {
		p.log = &boundedLog{limit: limit}
	}
}

type boundedLog struct {
	mu    sync.Mutex
	limit int
	buf   []byte
}

func (l *boundedLog) Write(p []byte) (int, error) {
	l.append(string(p))
	return len(p), nil
}

func (l *boundedLog) append(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf = append(l.buf, []byte(s)...)
	if len(l.buf) > l.limit {
		l.buf = append([]byte(nil), l.buf[len(l.buf)-l.limit:]...)
	}
}

func (l *boundedLog) lines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	text := strings.TrimRight(string(l.buf), "\n")
	if text == "" {
		return []string{}
	}
	return strings.Split(text, "\n")
}
