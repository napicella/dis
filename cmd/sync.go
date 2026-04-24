package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	githubRepo    = "napicella/dis"
	packagesAsset = "dis_packages.tar.gz"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Download the latest common packages from the dis release",
	Long: `Fetches the latest dis release from GitHub, downloads dis_packages.tar.gz,
and extracts it to ~/.local/share/dis/packages.

Run this after installing dis or when a new release is available to keep
your common package library up to date.

The token can also be provided via the GITHUB_TOKEN environment variable.`,
	RunE: runSync,
}

var syncToken string

func init() {
	syncCmd.Flags().StringVar(&syncToken, "token", "", "GitHub personal access token (or set GITHUB_TOKEN env var)")
	rootCmd.AddCommand(syncCmd)
}

func runSync(_ *cobra.Command, _ []string) error {
	token := syncToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("a GitHub token is required: use --token or set GITHUB_TOKEN")
	}

	// Resolve latest release tag.
	tag, err := latestReleaseTag(token)
	if err != nil {
		return fmt.Errorf("fetching latest release: %w", err)
	}
	fmt.Printf("==> Syncing packages from %s\n", tag)

	// Resolve asset ID.
	assetID, err := releaseAssetID(token, tag, packagesAsset)
	if err != nil {
		return fmt.Errorf("finding asset %q in release %s: %w", packagesAsset, tag, err)
	}

	// Download tarball.
	destDir, err := packagesDir()
	if err != nil {
		return err
	}
	if err := downloadAndExtract(token, assetID, destDir); err != nil {
		return fmt.Errorf("extracting packages: %w", err)
	}

	fmt.Printf("==> Packages installed to %s\n", destDir)
	return nil
}

// packagesDir returns ~/.local/share/dis/packages, creating it if needed.
func packagesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "dis", "packages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func ghGet(token, url string, accept string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", accept)
	return http.DefaultClient.Do(req)
}

func latestReleaseTag(token string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	resp, err := ghGet(token, url, "application/vnd.github+json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("no tag_name in response")
	}
	return release.TagName, nil
}

func releaseAssetID(token, tag, assetName string) (int64, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", githubRepo, tag)
	resp, err := ghGet(token, url, "application/vnd.github+json")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var release struct {
		Assets []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return 0, err
	}
	for _, a := range release.Assets {
		if a.Name == assetName {
			return a.ID, nil
		}
	}
	return 0, fmt.Errorf("asset %q not found in release %s", assetName, tag)
}

func downloadAndExtract(token string, assetID int64, destDir string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/assets/%d", githubRepo, assetID)
	resp, err := ghGet(token, url, "application/octet-stream")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("reading gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Strip leading "packages/" prefix.
		rel := strings.TrimPrefix(hdr.Name, "packages/")
		if rel == "" || rel == "." {
			continue
		}

		target := filepath.Join(destDir, rel)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}
