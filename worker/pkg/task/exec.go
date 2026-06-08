package task

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns-worker/pkg/config"
)

// runs the command string specified in action.Command.
// On Windows the command is passed to cmd.exe /C; on other platforms to sh -c.
// stdout and stderr are captured and logged.
func run(ctx context.Context, action config.ActionConfig) error {
	command := strings.TrimSpace(action.Command)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(buf.String())
		return fmt.Errorf("exec: %q: %w\n%s", command, err, output)
	}

	if out := strings.TrimSpace(buf.String()); out != "" {
		log.Info().Msgf("[WORKER] exec output:\n%s", out)
	}

	return nil
}
