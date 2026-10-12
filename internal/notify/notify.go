// Package notify delivers desktop notifications when a job finishes. The
// default sender uses beeep, which routes to the native notifier of each OS
// (toast on Windows, Notification Center on macOS, notify-send on Linux).
// Tests and headless environments can swap the sender with SetSender.
package notify

import (
	"github.com/fabianoflorentino/vidctl/internal/dlog"
	"github.com/gen2brain/beeep"
)

// Sender delivers one desktop notification. Errors mean the platform has no
// notification daemon available, which is never fatal for a job.
type Sender interface {
	Notify(title, message string) error
}

// beeepSender is the production sender, backed by the beeep library.
type beeepSender struct{}

func (beeepSender) Notify(title, message string) error {
	return beeep.Notify(title, message, "")
}

var sender Sender = beeepSender{}

// SetSender swaps the active sender. Passing nil disables notifications.
func SetSender(s Sender) {
	sender = s
}

// Send delivers a notification when notifications are enabled and a sender is
// installed. Delivery failures are logged, not returned: a missing desktop
// notification daemon must never affect the compression result.
func Send(title, message string) {
	if sender == nil {
		return
	}
	if err := sender.Notify(title, message); err != nil {
		dlog.Printf("[notify] could not deliver %q: %v", title, err)
	}
}
