//go:build integration

package tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testImage = "disgo-test"

// TestInstallIntegration builds the test Docker image, spins up a container,
// copies the disgo binary and testdata into it, runs disgo install, and verifies
// that the installers ran correctly (config generator injection, cross-package
// env var export, dependency ordering, and apt package installation).
func TestInstallIntegration(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH; skipping integration test")
	}

	// --- Locate the pre-built binary (set by the Makefile via DISGO_BIN) ---
	binPath := os.Getenv("DISGO_BIN")
	if binPath == "" {
		t.Fatal("DISGO_BIN env var not set; run tests via 'make test-integration'")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("DISGO_BIN %q not found: %v", binPath, err)
	}

	// --- Build the test Docker image ---
	testdataAbs, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("resolving testdata path: %v", err)
	}
	dockerfilePath := filepath.Join(testdataAbs, "Dockerfile")
	mustRun(t, "docker", "build", "-t", testImage, "-f", dockerfilePath, testdataAbs)

	// --- Start a detached container ---
	containerID := mustRun(t, "docker", "run", "-d", "--rm", testImage, "sleep", "300")
	containerID = strings.TrimSpace(containerID)
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", containerID).Run() //nolint:errcheck
	})

	// --- Copy binary into container (accessible to dev user) ---
	mustRun(t, "docker", "cp", binPath, containerID+":/usr/local/bin/disgo")
	mustDockerExec(t, containerID, "sudo", "chmod", "+x", "/usr/local/bin/disgo")

	// --- Copy testdata into container ---
	mustRun(t, "docker", "cp", testdataAbs, containerID+":/testdata")
	mustDockerExec(t, containerID, "sudo", "chown", "-R", "dev:dev", "/testdata")
	mustDockerExec(t, containerID, "chmod", "-R", "+x", "/testdata")

	// --- Run disgo install ---
	mustDockerExec(t, containerID,
		"/usr/local/bin/disgo", "install",
		"--distro", "/testdata/distro.yml",
	)

	// --- Verify outcomes ---
	checks := []struct {
		desc     string
		cmd      []string
		contains string
	}{
		{
			desc: "producer ran",
			cmd:  []string{"test", "-f", "/tmp/producer-ran"},
		},
		{
			desc: "consumer ran",
			cmd:  []string{"test", "-f", "/tmp/consumer-ran"},
		},
		{
			desc:     "config generator injected GREETING",
			cmd:      []string{"cat", "/tmp/greeting"},
			contains: "hello",
		},
		{
			desc:     "cross-package TOKEN export",
			cmd:      []string{"cat", "/tmp/token"},
			contains: "abc123",
		},
		{
			desc:     "jq is installed",
			cmd:      []string{"which", "jq"},
			contains: "jq",
		},
		{
			desc:     "test/producer recorded in state",
			cmd:      []string{"/usr/local/bin/disgo", "list"},
			contains: "test/producer",
		},
		{
			desc:     "test/consumer recorded in state",
			cmd:      []string{"/usr/local/bin/disgo", "list"},
			contains: "test/consumer",
		},
		{
			desc:     "test/jq recorded in state",
			cmd:      []string{"/usr/local/bin/disgo", "list"},
			contains: "test/jq",
		},
	}

	for _, c := range checks {
		t.Run(c.desc, func(t *testing.T) {
			args := append([]string{"exec", containerID}, c.cmd...)
			out, err := exec.Command("docker", args...).CombinedOutput()
			if err != nil {
				t.Fatalf("check %q failed: %v\noutput: %s", c.desc, err, out)
			}
			if c.contains != "" && !strings.Contains(string(out), c.contains) {
				t.Fatalf("expected output to contain %q, got: %q", c.contains, string(out))
			}
		})
	}
}

// mustRun runs a command and returns its stdout, failing the test on error.
func mustRun(t *testing.T, name string, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("command %q failed: %v\nstdout: %s\nstderr: %s",
			fmt.Sprintf("%s %s", name, strings.Join(args, " ")), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// mustDockerExec runs a command inside the container as the default user, failing the test on error.
func mustDockerExec(t *testing.T, containerID string, cmd ...string) {
	t.Helper()
	args := append([]string{"exec", containerID}, cmd...)
	mustRun(t, "docker", args...)
}
