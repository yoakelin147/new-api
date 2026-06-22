package common

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type StreamEndReason string

const (
	StreamEndReasonNone                StreamEndReason = ""
	StreamEndReasonDone                StreamEndReason = "done"
	StreamEndReasonTimeout             StreamEndReason = "timeout"
	StreamEndReasonClientGone          StreamEndReason = "client_gone"
	StreamEndReasonClientGoneAfterDone StreamEndReason = "client_gone_after_done"
	StreamEndReasonScannerErr          StreamEndReason = "scanner_error"
	StreamEndReasonHandlerStop         StreamEndReason = "handler_stop"
	StreamEndReasonEOF                 StreamEndReason = "eof"
	StreamEndReasonPanic               StreamEndReason = "panic"
	StreamEndReasonPingFail            StreamEndReason = "ping_fail"
)

const maxStreamErrorEntries = 20

type StreamErrorEntry struct {
	Message   string
	Timestamp time.Time
}

type StreamStatus struct {
	EndReason StreamEndReason
	EndError  error
	endOnce   sync.Once

	mu                   sync.Mutex
	Errors               []StreamErrorEntry
	ErrorCount           int
	completedBeforeClose bool
}

func NewStreamStatus() *StreamStatus {
	return &StreamStatus{}
}

func (s *StreamStatus) SetEndReason(reason StreamEndReason, err error) {
	if s == nil {
		return
	}
	s.endOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if reason == StreamEndReasonClientGone && s.completedBeforeClose {
			reason = StreamEndReasonClientGoneAfterDone
			err = nil
		}
		s.EndReason = reason
		s.EndError = err
	})
}

func (s *StreamStatus) MarkCompletedBeforeClose() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.completedBeforeClose = true
	if s.EndReason == StreamEndReasonClientGone {
		s.EndReason = StreamEndReasonClientGoneAfterDone
		s.EndError = nil
	}
	s.mu.Unlock()
}

func (s *StreamStatus) CompletedBeforeClose() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.completedBeforeClose
}

func (s *StreamStatus) RecordError(msg string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount++
	if len(s.Errors) < maxStreamErrorEntries {
		s.Errors = append(s.Errors, StreamErrorEntry{
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

func (s *StreamStatus) HasErrors() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ErrorCount > 0
}

func (s *StreamStatus) TotalErrorCount() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ErrorCount
}

func (s *StreamStatus) IsNormalEnd() bool {
	if s == nil {
		return true
	}
	return s.EndReason == StreamEndReasonDone ||
		s.EndReason == StreamEndReasonClientGoneAfterDone ||
		s.EndReason == StreamEndReasonEOF ||
		s.EndReason == StreamEndReasonHandlerStop
}

func (s *StreamStatus) Summary() string {
	if s == nil {
		return "StreamStatus<nil>"
	}
	b := &strings.Builder{}
	fmt.Fprintf(b, "reason=%s", s.EndReason)
	if s.EndError != nil {
		fmt.Fprintf(b, " end_error=%q", s.EndError.Error())
	}
	s.mu.Lock()
	if s.ErrorCount > 0 {
		fmt.Fprintf(b, " soft_errors=%d", s.ErrorCount)
	}
	s.mu.Unlock()
	return b.String()
}
