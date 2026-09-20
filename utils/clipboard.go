// Package utils provides platform helpers.
package utils

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Copy sends the value through stdin, never through shell interpolation.
func Copy(value string) error {
	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"pbcopy"}}
	case "windows":
		candidates = [][]string{{"clip.exe"}}
	case "linux":
		candidates = [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}}
	default:
		return fmt.Errorf("clipboard is unsupported on %s", runtime.GOOS)
	}
	for _, candidate := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		cmd := exec.CommandContext(ctx, candidate[0], candidate[1:]...)
		cmd.Stdin = strings.NewReader(value)
		err := cmd.Run()
		cancel()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("clipboard unavailable; check your clipboard utility and desktop session")
}
