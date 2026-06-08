package task

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns-worker/pkg/config"
)

// restartService stops and starts the named OS service.
// On Windows it uses the Service Control Manager (sc.exe).
// On all other platforms it delegates to systemctl.
func restartService(ctx context.Context, action config.ActionConfig) error {
	name := strings.TrimSpace(action.ServiceName)

	if runtime.GOOS == "windows" {
		return restartServiceWindows(ctx, name)
	}

	return restartServiceSystemd(ctx, name)
}

func restartServiceWindows(ctx context.Context, name string) error {
	// Stop the service; ignore "service is not running" (exit code 1062).
	stop := exec.CommandContext(ctx, "sc", "stop", name)
	if out, err := stop.CombinedOutput(); err != nil {
		trimmed := strings.TrimSpace(string(out))
		// 1062 = ERROR_SERVICE_NOT_ACTIVE – acceptable, continue to start.
		if !strings.Contains(trimmed, "1062") {
			return fmt.Errorf("service_restart: sc stop %q: %w\n%s", name, err, trimmed)
		}
		log.Warn().Msgf("[WORKER] service_restart: %q was not running, proceeding with start", name)
	}

	// Wait for the service to stop before starting it again; otherwise, sc start may fail with "service is already running" (1060).
	// Wait timeout is 30s, which should be more than enough for any service to stop.
	for i := 0; i < 30; i++ {
		out, err := exec.CommandContext(ctx, "sc", "query", name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("service_restart: sc query %q state=inactive: %w\n%s", name, err, strings.TrimSpace(string(out)))
		}
		if bytes.Contains(out, []byte("STATE              : 1  STOPPED")) {
			break
		}
		log.Info().Msgf("[WORKER] waiting for service %q to stop...", name)
		time.Sleep(1 * time.Second)
	}

	start := exec.CommandContext(ctx, "sc", "start", name)
	if out, err := start.CombinedOutput(); err != nil {
		return fmt.Errorf("service_restart: sc start %q: %w\n%s", name, err, strings.TrimSpace(string(out)))
	}

	return nil
}

func restartServiceSystemd(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "restart", name)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("service_restart: systemctl restart %q: %w\n%s", name, err, strings.TrimSpace(buf.String()))
	}

	return nil
}
