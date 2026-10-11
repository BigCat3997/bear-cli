package browser

import (
	"strings"
	"testing"
	"time"
)

func TestWaitForBrowserNonInteractiveWaitsForBrowserClose(t *testing.T) {
	done := make(chan struct{})
	returned := make(chan struct{})
	go func() {
		waitForBrowser(done, false, strings.NewReader(""))
		close(returned)
	}()

	select {
	case <-returned:
		t.Fatal("waitForBrowser returned before the browser closed")
	case <-time.After(20 * time.Millisecond):
	}

	close(done)
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("waitForBrowser did not return after the browser closed")
	}
}

func TestWaitForBrowserInteractiveAcceptsEnter(t *testing.T) {
	done := make(chan struct{})
	returned := make(chan struct{})
	go func() {
		waitForBrowser(done, true, strings.NewReader("\n"))
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("waitForBrowser did not return after Enter")
	}
}
