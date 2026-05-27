package dis

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed wrapper.sh
var wrapperScript []byte

// writeWrapper writes the embedded wrapper.sh to a temporary file and returns
// its path. The caller must delete the file when done (e.g. defer os.Remove(p)).
func writeWrapper() (string, error) {
	f, err := os.CreateTemp("", "dis-wrapper-*.sh")
	if err != nil {
		return "", fmt.Errorf("creating temp wrapper file: %w", err)
	}
	if _, err := f.Write(wrapperScript); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("writing wrapper file: %w", err)
	}
	if err := f.Chmod(0755); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("chmod wrapper file: %w", err)
	}
	f.Close()
	return f.Name(), nil
}
