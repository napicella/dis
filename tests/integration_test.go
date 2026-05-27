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

const testImage = "dis-test"

// TestInstallIntegration builds the test Docker image, spins up a container,
// copies the dis binary and testdata into it, runs dis install, and verifies
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

	// --- Copy binary into container as "dis" ---
	mustRun(t, "docker", "cp", binPath, containerID+":/usr/local/bin/dis")
	mustDockerExec(t, containerID, "sudo", "chmod", "+x", "/usr/local/bin/dis")

	// --- Copy testdata into container ---
	mustRun(t, "docker", "cp", testdataAbs, containerID+":/testdata")
	mustDockerExec(t, containerID, "sudo", "chown", "-R", "dev:dev", "/testdata")
	mustDockerExec(t, containerID, "chmod", "-R", "+x", "/testdata")

	// --- Run dis install ---
	mustDockerExec(t, containerID,
		"/usr/local/bin/dis", "install",
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
			cmd:      []string{"/usr/local/bin/dis", "list"},
			contains: "test/producer",
		},
		{
			desc:     "test/consumer recorded in state",
			cmd:      []string{"/usr/local/bin/dis", "list"},
			contains: "test/consumer",
		},
		{
			desc:     "test/jq recorded in state",
			cmd:      []string{"/usr/local/bin/dis", "list"},
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

// TestInstallIntegrationExportsCache verifies that exported values from a
// previously-installed package are available from the cache when that package
// is skipped on a subsequent run.
//
// Flow:
//  1. dis install  → producer runs (exports TOKEN to cache), consumer runs.
//  2. dis run --reinstall test/consumer → producer is already installed (skipped),
//     but TOKEN comes from the exports cache; consumer re-runs successfully.
func TestInstallIntegrationExportsCache(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH; skipping integration test")
	}

	binPath := os.Getenv("DISGO_BIN")
	if binPath == "" {
		t.Fatal("DISGO_BIN env var not set; run tests via 'make test-integration'")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("DISGO_BIN %q not found: %v", binPath, err)
	}

	testdataAbs, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("resolving testdata path: %v", err)
	}
	dockerfilePath := filepath.Join(testdataAbs, "Dockerfile")
	mustRun(t, "docker", "build", "-t", testImage, "-f", dockerfilePath, testdataAbs)

	containerID := mustRun(t, "docker", "run", "-d", "--rm", testImage, "sleep", "300")
	containerID = strings.TrimSpace(containerID)
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", containerID).Run() //nolint:errcheck
	})

	mustRun(t, "docker", "cp", binPath, containerID+":/usr/local/bin/dis")
	mustDockerExec(t, containerID, "sudo", "chmod", "+x", "/usr/local/bin/dis")
	mustRun(t, "docker", "cp", testdataAbs, containerID+":/testdata")
	mustDockerExec(t, containerID, "sudo", "chown", "-R", "dev:dev", "/testdata")
	mustDockerExec(t, containerID, "chmod", "-R", "+x", "/testdata")

	// Step 1: full install — producer runs and exports TOKEN to the cache.
	mustDockerExec(t, containerID,
		"/usr/local/bin/dis", "install",
		"--distro", "/testdata/distro.yml",
	)

	// Remove consumer-ran so we can confirm the consumer executes again.
	mustDockerExec(t, containerID, "rm", "-f", "/tmp/consumer-ran")

	// Step 2: re-run consumer only; producer is already installed (skipped)
	// but TOKEN must come from the exports cache.
	mustDockerExec(t, containerID,
		"/usr/local/bin/dis", "run",
		"--distro", "/testdata/distro.yml",
		"--reinstall",
		"test/consumer",
	)

	checks := []struct {
		desc     string
		cmd      []string
		contains string
	}{
		{
			desc: "consumer ran again after reinstall",
			cmd:  []string{"test", "-f", "/tmp/consumer-ran"},
		},
		{
			desc:     "TOKEN injected from exports cache",
			cmd:      []string{"cat", "/tmp/token"},
			contains: "abc123",
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

// TestInstallIntegrationWorkspace verifies the dis.workspace code path: a source
// directory with a dis.workspace file is walked only for the declared package
// roots, and the installer receives correct DIS_PKG_ROOT and DIS_CONFIG_FOLDER.
func TestInstallIntegrationWorkspace(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH; skipping integration test")
	}

	binPath := os.Getenv("DISGO_BIN")
	if binPath == "" {
		t.Fatal("DISGO_BIN env var not set; run tests via 'make test-integration'")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("DISGO_BIN %q not found: %v", binPath, err)
	}

	// Reuse the same Docker image built in the other test.
	testdataAbs, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("resolving testdata path: %v", err)
	}
	dockerfilePath := filepath.Join(testdataAbs, "Dockerfile")
	mustRun(t, "docker", "build", "-t", testImage, "-f", dockerfilePath, testdataAbs)

	containerID := mustRun(t, "docker", "run", "-d", "--rm", testImage, "sleep", "300")
	containerID = strings.TrimSpace(containerID)
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", containerID).Run() //nolint:errcheck
	})

	mustRun(t, "docker", "cp", binPath, containerID+":/usr/local/bin/dis")
	mustDockerExec(t, containerID, "sudo", "chmod", "+x", "/usr/local/bin/dis")

	testdataWsAbs, err := filepath.Abs("testdata-workspace")
	if err != nil {
		t.Fatalf("resolving testdata-workspace path: %v", err)
	}
	mustRun(t, "docker", "cp", testdataWsAbs, containerID+":/testdata-workspace")
	mustDockerExec(t, containerID, "sudo", "chown", "-R", "dev:dev", "/testdata-workspace")
	mustDockerExec(t, containerID, "chmod", "-R", "+x", "/testdata-workspace")

	mustDockerExec(t, containerID,
		"/usr/local/bin/dis", "install",
		"--distro", "/testdata-workspace/distro.yml",
	)

	checks := []struct {
		desc     string
		cmd      []string
		contains string
	}{
		{
			desc: "ws/hello installer ran",
			cmd:  []string{"test", "-f", "/tmp/ws-hello-ran"},
		},
		{
			desc:     "DIS_CONFIG_FOLDER pointed to configs dir",
			cmd:      []string{"cat", "/tmp/ws-greeting"},
			contains: "hello-from-configs",
		},
		{
			desc:     "DIS_PKG_ROOT points to package root (hello/), not source root (src/)",
			cmd:      []string{"cat", "/tmp/ws-pkg-root"},
			contains: "hello",
		},
		{
			desc:     "ws/hello recorded in state",
			cmd:      []string{"/usr/local/bin/dis", "list"},
			contains: "ws/hello",
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

// TestInstallIntegrationScopedParameters verifies that scoped parameters
// declared on packages entries are injected only into the packages they target.
//
// distro layout:
//   - GLOBAL_PARAM is a top-level global available to every package.
//   - SHARED_PARAM is scoped to both test/pkg-a and test/pkg-b via names:[...].
//   - EXCLUSIVE_PARAM is scoped only to test/pkg-b via name: test/pkg-b.
//
// Expected outcomes:
//   - pkg-a receives GLOBAL_PARAM and SHARED_PARAM but NOT EXCLUSIVE_PARAM.
//   - pkg-b receives all three.
func TestInstallIntegrationScopedParameters(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH; skipping integration test")
	}

	binPath := os.Getenv("DISGO_BIN")
	if binPath == "" {
		t.Fatal("DISGO_BIN env var not set; run tests via 'make test-integration'")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("DISGO_BIN %q not found: %v", binPath, err)
	}

	testdataAbs, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("resolving testdata path: %v", err)
	}
	dockerfilePath := filepath.Join(testdataAbs, "Dockerfile")
	mustRun(t, "docker", "build", "-t", testImage, "-f", dockerfilePath, testdataAbs)

	containerID := mustRun(t, "docker", "run", "-d", "--rm", testImage, "sleep", "300")
	containerID = strings.TrimSpace(containerID)
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", containerID).Run() //nolint:errcheck
	})

	mustRun(t, "docker", "cp", binPath, containerID+":/usr/local/bin/dis")
	mustDockerExec(t, containerID, "sudo", "chmod", "+x", "/usr/local/bin/dis")

	testdataScopedAbs, err := filepath.Abs("testdata-scoped")
	if err != nil {
		t.Fatalf("resolving testdata-scoped path: %v", err)
	}
	mustRun(t, "docker", "cp", testdataScopedAbs, containerID+":/testdata-scoped")
	mustDockerExec(t, containerID, "sudo", "chown", "-R", "dev:dev", "/testdata-scoped")
	mustDockerExec(t, containerID, "chmod", "-R", "+x", "/testdata-scoped")

	mustDockerExec(t, containerID,
		"/usr/local/bin/dis", "install",
		"--distro", "/testdata-scoped/distro.yml",
	)

	checks := []struct {
		desc        string
		cmd         []string
		contains    string
		notContains string
	}{
		{
			desc: "pkg-a ran",
			cmd:  []string{"test", "-f", "/tmp/pkg-a-ran"},
		},
		{
			desc: "pkg-b ran",
			cmd:  []string{"test", "-f", "/tmp/pkg-b-ran"},
		},
		{
			desc:     "pkg-a received global param",
			cmd:      []string{"cat", "/tmp/pkg-a-env"},
			contains: "GLOBAL_PARAM=global-value",
		},
		{
			desc:     "pkg-a received shared scoped param",
			cmd:      []string{"cat", "/tmp/pkg-a-env"},
			contains: "SHARED_PARAM=shared-value",
		},
		{
			desc:        "pkg-a did NOT receive exclusive param (isolation)",
			cmd:         []string{"cat", "/tmp/pkg-a-env"},
			notContains: "EXCLUSIVE_PARAM",
		},
		{
			desc:     "pkg-b received global param",
			cmd:      []string{"cat", "/tmp/pkg-b-env"},
			contains: "GLOBAL_PARAM=global-value",
		},
		{
			desc:     "pkg-b received shared scoped param",
			cmd:      []string{"cat", "/tmp/pkg-b-env"},
			contains: "SHARED_PARAM=shared-value",
		},
		{
			desc:     "pkg-b received exclusive scoped param",
			cmd:      []string{"cat", "/tmp/pkg-b-env"},
			contains: "EXCLUSIVE_PARAM=exclusive-value",
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
			if c.notContains != "" && strings.Contains(string(out), c.notContains) {
				t.Fatalf("expected output NOT to contain %q, got: %q", c.notContains, string(out))
			}
		})
	}
}

// TestInstallIntegrationRCTools verifies that the `dis tools add-rc-*` commands
// correctly upsert named sections in the generated RC files.
//
// The test installer (testdata/packages/rc_tools.sh) calls:
//   - dis tools add-rc-init    --name test-init    --content 'export TEST_INIT=1'
//   - dis tools add-rc-path    --name test-path    --content 'export PATH="/test/bin:$PATH"'
//   - dis tools add-rc-aliases --name test-aliases --content 'alias ll="ls -la"'
//
// Verifications:
//  1. Each RC file contains the expected BEGIN/END section with the correct content.
//  2. Running the installer a second time (idempotency) does NOT duplicate the section.
func TestInstallIntegrationRCTools(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH; skipping integration test")
	}

	binPath := os.Getenv("DISGO_BIN")
	if binPath == "" {
		t.Fatal("DISGO_BIN env var not set; run tests via 'make test-integration'")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("DISGO_BIN %q not found: %v", binPath, err)
	}

	testdataAbs, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("resolving testdata path: %v", err)
	}
	dockerfilePath := filepath.Join(testdataAbs, "Dockerfile")
	mustRun(t, "docker", "build", "-t", testImage, "-f", dockerfilePath, testdataAbs)

	containerID := mustRun(t, "docker", "run", "-d", "--rm", testImage, "sleep", "300")
	containerID = strings.TrimSpace(containerID)
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", containerID).Run() //nolint:errcheck
	})

	mustRun(t, "docker", "cp", binPath, containerID+":/usr/local/bin/dis")
	mustDockerExec(t, containerID, "sudo", "chmod", "+x", "/usr/local/bin/dis")
	mustRun(t, "docker", "cp", testdataAbs, containerID+":/testdata")
	mustDockerExec(t, containerID, "sudo", "chown", "-R", "dev:dev", "/testdata")
	mustDockerExec(t, containerID, "chmod", "-R", "+x", "/testdata")

	// --- First run ---
	mustDockerExec(t, containerID,
		"/usr/local/bin/dis", "install",
		"--distro", "/testdata/distro.yml",
	)

	rcBase := "/home/dev/rc/configs-generated"

	checks := []struct {
		desc     string
		cmd      []string
		contains string
	}{
		{
			desc: "rc-tools installer ran",
			cmd:  []string{"test", "-f", "/tmp/rc-tools-ran"},
		},
		{
			desc:     "bash_init contains BEGIN marker for test-init",
			cmd:      []string{"grep", "-F", "# BEGIN test-init import generated by dis config", rcBase + "/bash_init"},
			contains: "BEGIN test-init",
		},
		{
			desc:     "bash_init contains END marker for test-init",
			cmd:      []string{"grep", "-F", "# END test-init import generated by dis config", rcBase + "/bash_init"},
			contains: "END test-init",
		},
		{
			desc:     "bash_init contains init content",
			cmd:      []string{"grep", "-F", "export TEST_INIT=1", rcBase + "/bash_init"},
			contains: "TEST_INIT",
		},
		{
			desc:     "bash_paths contains BEGIN marker for test-path",
			cmd:      []string{"grep", "-F", "# BEGIN test-path import generated by dis config", rcBase + "/bash_paths"},
			contains: "BEGIN test-path",
		},
		{
			desc:     "bash_paths contains path content",
			cmd:      []string{"grep", "-F", "/test/bin", rcBase + "/bash_paths"},
			contains: "/test/bin",
		},
		{
			desc:     "bash_aliases contains BEGIN marker for test-aliases",
			cmd:      []string{"grep", "-F", "# BEGIN test-aliases import generated by dis config", rcBase + "/bash_aliases"},
			contains: "BEGIN test-aliases",
		},
		{
			desc:     "bash_aliases contains aliases content",
			cmd:      []string{"grep", "-F", `alias ll="ls -la"`, rcBase + "/bash_aliases"},
			contains: "ll",
		},
		{
			desc:     "test/rc-tools recorded in state",
			cmd:      []string{"/usr/local/bin/dis", "list"},
			contains: "test/rc-tools",
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

	// --- Idempotency: reinstall rc-tools and verify sections appear only once ---
	mustDockerExec(t, containerID,
		"/usr/local/bin/dis", "run",
		"--distro", "/testdata/distro.yml",
		"--reinstall",
		"test/rc-tools",
	)

	idempotencyChecks := []struct {
		desc      string
		file      string
		marker    string
		wantExact int
	}{
		{
			desc:      "test-init section appears exactly once after reinstall",
			file:      rcBase + "/bash_init",
			marker:    "# BEGIN test-init import generated by dis config",
			wantExact: 1,
		},
		{
			desc:      "test-path section appears exactly once after reinstall",
			file:      rcBase + "/bash_paths",
			marker:    "# BEGIN test-path import generated by dis config",
			wantExact: 1,
		},
		{
			desc:      "test-aliases section appears exactly once after reinstall",
			file:      rcBase + "/bash_aliases",
			marker:    "# BEGIN test-aliases import generated by dis config",
			wantExact: 1,
		},
	}

	for _, c := range idempotencyChecks {
		t.Run(c.desc, func(t *testing.T) {
			// Use grep -c to count occurrences of the marker.
			args := []string{"exec", containerID, "bash", "-c",
				fmt.Sprintf("grep -cF %q %s", c.marker, c.file)}
			out, err := exec.Command("docker", args...).CombinedOutput()
			if err != nil {
				t.Fatalf("grep -c failed: %v\noutput: %s", err, out)
			}
			count := strings.TrimSpace(string(out))
			if count != fmt.Sprintf("%d", c.wantExact) {
				t.Fatalf("expected marker %q to appear %d time(s) in %s, got count=%s\nfile contents:\n%s",
					c.marker, c.wantExact, c.file, count,
					mustDockerCat(t, containerID, c.file))
			}
		})
	}
}

// mustDockerCat reads the contents of a file inside the container.
func mustDockerCat(t *testing.T, containerID, path string) string {
	t.Helper()
	out, _ := exec.Command("docker", "exec", containerID, "cat", path).CombinedOutput()
	return string(out)
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
