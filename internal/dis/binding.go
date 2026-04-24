package dis

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed binding.sh
var bindingScript []byte

// writeBinding writes the embedded binding.sh to a temporary file and returns
// its path. The caller must delete the file when done (e.g. defer os.Remove(p)).
func writeBinding() (string, error) {
	f, err := os.CreateTemp("", "dis-binding-*.sh")
	if err != nil {
		return "", fmt.Errorf("creating temp binding file: %w", err)
	}
	if _, err := f.Write(bindingScript); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("writing binding file: %w", err)
	}
	if err := f.Chmod(0755); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("chmod binding file: %w", err)
	}
	f.Close()
	return f.Name(), nil
}
