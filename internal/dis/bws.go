package dis

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// bwsVersion is the Bitwarden Secrets Manager CLI (bws) release dis installs.
const bwsVersion = "2.1.0"

// bwsPin is a pinned bws release asset and the sha256 of its zip, from the
// release's bws-sha256-checksums file.
type bwsPin struct {
	asset  string
	sha256 string
}

// bwsPins are the statically linked (musl) builds, which run on any Linux
// libc, keyed by GOOS/GOARCH.
var bwsPins = map[string]bwsPin{
	"linux/amd64": {"bws-x86_64-unknown-linux-musl-" + bwsVersion + ".zip", "f59ee150e42b82128d437087e9bac920053c6bfddcb960d20ce9386e5ac9bba6"},
	"linux/arm64": {"bws-aarch64-unknown-linux-musl-" + bwsVersion + ".zip", "eb0f1ae61d1c3b74244d2841233276e05c77e8be4da197ed90fc6248387005e1"},
}

// bwsReleaseURL is where the pinned assets are downloaded from; tests point it
// at a local server.
var bwsReleaseURL = "https://github.com/bitwarden/sdk-sm/releases/download/bws-v" + bwsVersion + "/"

// bwsExecutable returns the bws to run. DIS_BWS, when set, names it and skips
// the pin, the CA check and the download (for tests, and hosts that bring
// their own bws). Otherwise it is the pinned build in
// ~/.local/share/dis/bin/bws-<version>, downloaded on first use.
func bwsExecutable(home string) (string, error) {
	if p := os.Getenv("DIS_BWS"); p != "" {
		return p, nil
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	pin, ok := bwsPins[platform]
	if !ok {
		return "", fmt.Errorf("bws: no pinned build for %s", platform)
	}
	// bws panics without the system CA store, so check it even when bws is
	// already installed.
	if err := checkSystemCAs(); err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "dis", "bin")
	path := filepath.Join(dir, "bws-"+bwsVersion)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	fmt.Printf("==> Installing bws %s to %s\n", bwsVersion, path)
	if err := installBws(dir, path, bwsReleaseURL+pin.asset, pin.sha256); err != nil {
		return "", err
	}
	return path, nil
}

// checkSystemCAs fails when the system CA certificates can't be loaded.
func checkSystemCAs() error {
	pool, err := x509.SystemCertPool()
	// Subjects is deprecated only because it omits system roots on some
	// platforms; on Linux the system pool is read from files and listed.
	if err != nil || len(pool.Subjects()) == 0 { //nolint:staticcheck
		return errors.New("bws needs the system CA certificates: install the ca-certificates package")
	}
	return nil
}

// installBws downloads the zip at url, checks its sha256, and writes its only
// entry, bws, to path with mode 0755, atomically. Other bws-* files in dir,
// left by earlier pins, are removed.
func installBws(dir, path, url, wantSHA256 string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("bws download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bws download: %s: %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("bws download: %w", err)
	}
	// The zip is only held in memory, so a mismatch leaves nothing behind.
	if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != wantSHA256 {
		return errors.New("bws download: checksum mismatch")
	}
	bin, err := bwsFromZip(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".bws-download-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	old, _ := filepath.Glob(filepath.Join(dir, "bws-*"))
	for _, p := range old {
		if p != path {
			os.Remove(p)
		}
	}
	return nil
}

// bwsFromZip returns the content of the zip's bws entry. Any other entry is
// rejected, so nothing but bws is ever written.
func bwsFromZip(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("bws download: %w", err)
	}
	var bin []byte
	for _, f := range zr.File {
		if f.Name != "bws" || bin != nil {
			return nil, fmt.Errorf("bws download: unexpected entry %q in the zip", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("bws download: %w", err)
		}
		bin, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("bws download: %w", err)
		}
	}
	if bin == nil {
		return nil, errors.New("bws download: no bws in the zip")
	}
	return bin, nil
}

// ensureBwsStateDir makes ~/.config/bws, where bws keeps its state, private:
// bws itself creates it with mode 755.
func ensureBwsStateDir(home string) error {
	dir := filepath.Join(home, ".config", "bws")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}
