package test

import (
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

type neverClosingBody struct {
	readStarted chan struct{}
	unblock     chan struct{}
	readOnce    sync.Once
	closeOnce   sync.Once
}

func newNeverClosingBody() *neverClosingBody {
	return &neverClosingBody{
		readStarted: make(chan struct{}),
		unblock:     make(chan struct{}),
	}
}

func (b *neverClosingBody) Read([]byte) (int, error) {
	b.readOnce.Do(func() { close(b.readStarted) })
	<-b.unblock
	return 0, io.EOF
}

func (b *neverClosingBody) Close() error {
	b.closeOnce.Do(func() { close(b.unblock) })
	return nil
}

func TestResponseRequireStatusDoesNotReadBodyOnExpectedStatus(t *testing.T) {
	body := newNeverClosingBody()
	response := NewResponse(t, &http.Response{
		StatusCode: http.StatusOK,
		Body:       body,
	})
	done := make(chan struct{})

	go func() {
		response.RequireStatus(http.StatusOK)
		close(done)
	}()

	select {
	case <-done:
	case <-body.readStarted:
		_ = body.Close()
		<-done
		t.Fatal("RequireStatus read an open response body for an expected status")
	case <-time.After(time.Second):
		_ = body.Close()
		<-done
		t.Fatal("RequireStatus did not return for an expected status")
	}
}
