package tools

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func PingHost(host string, timeout time.Duration) (string, error) {
	// Use -c 1 for Linux/macOS to send only one packet
	// Use -W 1 for a 1-second wait time for the reply (adjust if needed)
	// Consider using platform-specific flags if necessary or a go ping library
	cmd := exec.Command("ping", "-c", "1", "-W", "1", host)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start ping command: %w", err)
	}

	// Wait for the command to finish or timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		// Timeout occurred
		if err := cmd.Process.Kill(); err != nil {
			return "", fmt.Errorf("failed to kill ping process after timeout: %w", err)
		}
		return "", fmt.Errorf("ping command timed out after %v", timeout)
	case err := <-done:
		// Command finished
		output := out.String() + stderr.String()
		if err != nil {
			// Ping might return non-zero exit code even if it gets output (e.g., packet loss)
			// We return the output along with the error in this case.
			return strings.TrimSpace(output), fmt.Errorf("ping command failed with exit code: %w. Output: %s", err, output)
		}
		return strings.TrimSpace(output), nil
	}
}
