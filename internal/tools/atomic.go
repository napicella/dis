package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to a temp file next to path and renames it over
// path, creating the parent dirs. An existing file keeps its mode; a new one
// gets perm.
// When path is a symlink, the link's final target is written instead, even
// when it does not exist yet.
//
// The rename is atomic within a filesystem, so a program reading the config
// sees the old content or the new, never half a file, and a failed write leaves
// the old file in place. The temp file is in the same directory as the target
// so the rename never crosses filesystems. Renaming over a symlink would
// replace the link with a regular file, which is why the link is resolved
// first (dotfile managers often symlink configs).
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	path, err := resolveLink(path)
	if err != nil {
		return err
	}
	if fi, err := os.Stat(path); err == nil {
		// A directory, device or socket can't be replaced with a rename.
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		perm = fi.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// maxLinks bounds the symlinks resolveLink follows, as the kernel does.
const maxLinks = 40

// resolveLink follows path while it is a symlink and returns the first path
// that is not one, which may not exist (a dangling link). A relative target
// is relative to the directory holding the link.
//
// filepath.EvalSymlinks would be simpler, but it fails on a dangling link,
// and a link to a config that doesn't exist yet is a normal first install.
func resolveLink(path string) (string, error) {
	orig := path
	for range maxLinks + 1 {
		fi, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) || (err == nil && fi.Mode()&fs.ModeSymlink == 0) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			// Resolve the link's dir first, so ".." in target goes up from
			// where the link really is, as the kernel does.
			dir := filepath.Dir(path)
			if real, err := filepath.EvalSymlinks(dir); err == nil {
				dir = real
			}
			target = filepath.Join(dir, target)
		}
		path = target
	}
	return "", fmt.Errorf("%s: too many levels of symbolic links", orig)
}
