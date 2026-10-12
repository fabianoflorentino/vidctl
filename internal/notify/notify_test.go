package notify

import (
	"errors"
	"sync"
	"testing"
)

type fakeSender struct {
	mu     sync.Mutex
	titles []string
	errs   []error
}

func (f *fakeSender) Notify(title, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.titles = append(f.titles, title)
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return err
	}
	return nil
}

func TestSendDelivers(t *testing.T) {
	defer SetSender(nil)
	f := &fakeSender{}
	SetSender(f)
	Send("vidctl", "a.mp4 finished")
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.titles) != 1 || f.titles[0] != "vidctl" {
		t.Fatalf("sender received %v; want one vidctl notification", f.titles)
	}
}

func TestSendNilSenderDoesNotPanic(t *testing.T) {
	SetSender(nil)
	Send("vidctl", "nothing")
}

func TestSendIgnoresDeliveryError(t *testing.T) {
	defer SetSender(nil)
	f := &fakeSender{errs: []error{errors.New("no notification daemon")}}
	SetSender(f)
	Send("vidctl", "quiet failure")
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.titles) != 1 {
		t.Fatalf("sender received %v; want one attempt", f.titles)
	}
}
