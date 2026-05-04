package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ScrawnDotDev/scrawn-cli/internal/apperr"
)

func smokeTestServer(target string, packageManager string) error {
	ctx, cancel := context.WithTimeout(context.Background(), ServerTimeout)
	defer cancel()

	var command []string
	switch packageManager {
	case "bun":
		command = []string{"bun", "run", "dev:backend"}
	case "npm":
		command = []string{"npm", "run", "dev:backend"}
	default:
		return &apperr.CommandError{Summary: "unsupported package manager", Detail: packageManager}
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = target
	cmd.Env = os.Environ()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return &apperr.CommandError{Summary: "failed to start server", Detail: err.Error()}
	}

	readyErr := waitForHTTPHealth(DefaultHTTPURL, ServerTimeout, stdout, stderr)
	stopErr := stopProcess(cmd)
	if readyErr != nil {
		if stopErr != nil {
			return &apperr.CommandError{Summary: "server health check failed", Detail: readyErr.Error() + "; additionally failed to stop process: " + stopErr.Error()}
		}
		return &apperr.CommandError{Summary: "server health check failed", Detail: readyErr.Error()}
	}
	if stopErr != nil {
		return &apperr.CommandError{Summary: "server passed health check but failed to stop cleanly", Detail: stopErr.Error()}
	}

	return nil
}

func waitForHTTPHealth(endpoint string, timeout time.Duration, stdout *bytes.Buffer, stderr *bytes.Buffer) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(endpoint)
		if err == nil {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 256))
			resp.Body.Close()
			if readErr == nil && resp.StatusCode == http.StatusOK {
				if strings.TrimSpace(string(body)) == "Hello World!" {
					return nil
				}
			}
		}

		time.Sleep(1 * time.Second)
	}

	logs := collectLogs(stdout, stderr)
	if logs == "" {
		logs = "no startup logs were captured"
	}
	return errors.New("the server never became healthy at " + endpoint + ". Logs:\n" + logs)
}

func stopProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		killCmd := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
		if output, err := killCmd.CombinedOutput(); err != nil {
			if killErr := cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
				message := strings.TrimSpace(string(output))
				if message == "" {
					message = err.Error()
				}
				return errors.New(message + ": " + killErr.Error())
			}
		}
		waitErr := cmd.Wait()
		if waitErr != nil && !strings.Contains(strings.ToLower(waitErr.Error()), "already finished") && !isExpectedShutdownError(waitErr) {
			return waitErr
		}
		return nil
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		if killErr := cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return killErr
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "signal") {
			return nil
		}
		return err
	case <-time.After(5 * time.Second):
		if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
		waitErr := <-done
		if waitErr != nil && !isExpectedShutdownError(waitErr) {
			return waitErr
		}
		return nil
	}
}

func isExpectedShutdownError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "signal") ||
		strings.Contains(lower, "killed") ||
		strings.Contains(lower, "terminated") ||
		strings.Contains(lower, "exit status")
}
