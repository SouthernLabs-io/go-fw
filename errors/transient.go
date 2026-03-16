package errors

import (
	"database/sql"
	stderrors "errors"
	"net"
	"strings"
)

// IsTransient reports whether err looks like a transient error that may succeed if retried after a delay.
// It covers errors from sql and net packages. More can be added as needed.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	if stderrors.Is(err, sql.ErrConnDone) {
		return true
	}

	var netErr net.Error
	if stderrors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "unexpected eof") ||
		strings.Contains(msg, "server closed the connection unexpectedly")
}
