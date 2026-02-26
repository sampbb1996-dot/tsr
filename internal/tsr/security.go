package tsr

import (
	"fmt"
	"strings"
)

func validatePath(path string) error {
	if strings.Contains(path, "..") {
		return fmt.Errorf("security: path %q contains '..' (traversal not allowed)", path)
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("security: path %q must be relative (no leading '/')", path)
	}
	if strings.HasPrefix(path, "~") {
		return fmt.Errorf("security: path %q must be relative (no leading '~')", path)
	}
	// Reject Windows drive paths like C:\
	if len(path) >= 2 && path[1] == ':' {
		return fmt.Errorf("security: path %q looks like a Windows absolute path", path)
	}
	return nil
}
