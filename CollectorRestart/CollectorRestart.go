package collectorrestart

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type CollectorManager struct {
	Cmd *exec.Cmd
}

func (m *CollectorManager) Start() error {
	cmd := exec.Command("./collector/collector", "--config", "config/baseconfig.yaml")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}
	m.Cmd = cmd

	fmt.Println("Collector started successfully", m.Cmd == nil || m.Cmd.Process == nil)
	return nil
}

func (m *CollectorManager) Stop() error {
	if m.Cmd == nil || m.Cmd.Process == nil {
		return fmt.Errorf("collector process is not running")
	}

	err := m.Cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("failed to stop collector: %w", err)
	}
	_, _ = m.Cmd.Process.Wait()
	fmt.Println("Collector stopped successfully")
	return nil
}

func (m *CollectorManager) Restart() error {
	fmt.Println("Restarting collector...")

	if err := m.Stop(); err != nil {
		return fmt.Errorf("failed to stop collector during restart: %w", err)
	}

	time.Sleep(2 * time.Second)

	if err := m.Start(); err != nil {
		return fmt.Errorf("failed to restart collector: %w", err)
	}

	fmt.Println("Collector restarted successfully")
	return nil
}

func (m *CollectorManager) Status() string {
	if m.Cmd == nil || m.Cmd.Process == nil || m.Cmd.ProcessState != nil {
		return "Collector is not running"
	}
	return "Collector is running"
}
