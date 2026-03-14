package errutil

import (
	"errors"
	"fmt"
	"io"
)

// WrapIOErr wraps I/O errors but let EOF errors intact
func WrapIOErr(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) {
		return io.EOF
	}
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", msg, err)
}
