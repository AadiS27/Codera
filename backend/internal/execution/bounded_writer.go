package execution

import (
	"bytes"
	"context"
	"sync"
)

type BoundedWriter struct {
	mu         sync.Mutex
	buf        bytes.Buffer
	maxSize    int64
	current    int64
	cancelFunc context.CancelCauseFunc
	exceeded   bool
}

func NewBoundedWriter(maxSize int64, cancelFunc context.CancelCauseFunc) *BoundedWriter {
	return &BoundedWriter{
		maxSize:    maxSize,
		cancelFunc: cancelFunc,
	}
}

func (w *BoundedWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.exceeded {
		// Just drop writes after exceeding limit
		return len(p), nil
	}

	writeSize := int64(len(p))
	if w.current+writeSize > w.maxSize {
		// Write exactly up to the limit, then trigger cancellation
		allowed := w.maxSize - w.current
		if allowed > 0 {
			w.buf.Write(p[:allowed])
			w.current += allowed
		}

		w.exceeded = true
		if w.cancelFunc != nil {
			// Trigger process cancellation!
			w.cancelFunc(ErrOutputLimitExceeded)
		}

		// Return full length so caller doesn't error out prematurely,
		// allowing process to be cleanly killed by context cancellation.
		return len(p), nil
	}

	w.buf.Write(p)
	w.current += writeSize
	return len(p), nil
}

func (w *BoundedWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func (w *BoundedWriter) Exceeded() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.exceeded
}
