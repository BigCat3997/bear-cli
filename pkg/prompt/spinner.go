package prompt

import (
	"fmt"
	"os"
	"time"
)

// StartSpinner renders an ASCII spinner until the returned function is called.
func StartSpinner(message string) func(err error) {
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		frames := []rune{'-', '\\', '|', '/'}
		i := 0
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				close(done)
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stderr, "\r[%c] %s", frames[i%len(frames)], message)
				i++
			}
		}
	}()

	return func(err error) {
		close(stop)
		<-done

		status := "OK"
		if err != nil {
			status = "ERR"
		}

		fmt.Fprintf(os.Stderr, "\r[%s] %s\n", status, message)
	}
}
