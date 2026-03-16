package errors

import (
	"database/sql"
	"net"
	"os"
	"testing"
)

func TestIsTransient(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if IsTransient(nil) {
			t.Fatalf("expected nil error to be non-transient")
		}
	})

	t.Run("sql err conn done", func(t *testing.T) {
		if !IsTransient(sql.ErrConnDone) {
			t.Fatalf("expected sql.ErrConnDone to be transient")
		}
	})

	t.Run("timeout network error", func(t *testing.T) {
		err := &net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}
		if !IsTransient(err) {
			t.Fatalf("expected timeout network error to be transient")
		}
	})

	t.Run("dns not found", func(t *testing.T) {
		err := &net.DNSError{Err: "no such host", Name: "db.example.invalid", IsTimeout: false, IsTemporary: false}
		if IsTransient(err) {
			t.Fatalf("expected permanent DNS error to be non-transient")
		}
	})

	t.Run("connection refused message", func(t *testing.T) {
		err := &net.OpError{Op: "dial", Net: "tcp", Err: os.ErrPermission}
		if IsTransient(err) {
			t.Fatalf("expected unrelated network error to be non-transient before message checks")
		}

		if !IsTransient(&messageError{msg: "dial tcp 127.0.0.1:5432: connection refused"}) {
			t.Fatalf("expected connection refused to be treated as transient")
		}
	})
}

type messageError struct {
	msg string
}

func (e *messageError) Error() string {
	return e.msg
}
