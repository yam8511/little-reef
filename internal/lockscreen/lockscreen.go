// Package lockscreen shows the running round where a phone shows it while
// locked: an ongoing notification whose clock counts down by itself, with a
// button to pause or resume, and a foreground service that keeps the app's
// clock running in the background. It is a native module implemented in
// Kotlin for Android (see AGENTS.md); elsewhere NewTimer returns
// errors.ErrUnsupported and the app goes on without it.
package lockscreen

import (
	"io"
	"time"
)

// Timer is the round shown on the lock screen and in the notification shade.
//
//mygo:native
type Timer interface {
	// Show posts the ongoing notification, or updates it, and keeps the app
	// running in the background while it shows. The first time, it asks for
	// permission to post notifications where the system needs it.
	Show(r Round) error
	// Hide takes the notification away and lets the app sleep again.
	Hide() error
	// OnAction calls f with the button pressed on the notification, until
	// the subscription is closed.
	OnAction(f func(a Action)) io.Closer
	// BackgroundRestricted reports whether the system's battery optimization
	// may stop the app while it runs in the background.
	BackgroundRestricted() bool
	// AllowBackground asks the system to let the app run in the background:
	// its dialog where it has one, else the battery optimization settings.
	AllowBackground() error
	// NotificationsBlocked reports whether the system drops the countdown:
	// the app's notifications, or its countdown channel, are turned off, as
	// some systems do for apps installed outside a store.
	NotificationsBlocked() bool
	// OpenNotificationSettings opens the system's notification settings of
	// the app, to turn them on.
	OpenNotificationSettings() error
}

// Round is what the notification says about the current round.
type Round struct {
	// Title is the notification's title, such as 「專注中」.
	Title string
	// Text is its line under the title, such as the task, or what pausing
	// says.
	Text string
	// Running counts down to EndsAt; paused shows Remaining, still.
	Running bool
	// EndsAt is when a running round ends.
	EndsAt time.Time
	// Remaining is what is left of a paused round.
	Remaining time.Duration
	// Total is the round's whole length: the media card's progress bar
	// runs from 0 to it.
	Total time.Duration
	// Pause, Resume and End name the buttons: their labels where they show
	// text, and what assistive technology reads where they show icons.
	Pause, Resume, End string
	// ConfirmEnd is the end button's label once pressed where the button
	// itself asks to confirm, as on the media card: pressed again, it ends
	// the round.
	ConfirmEnd string
	// Style is how the notification looks.
	Style Style
}

// Style is how the round's notification looks. Android promotes only a
// notification of its own templates to a Live Update, so the two differ in
// where they show.
type Style string

const (
	// StyleLive is the system's template, a Live Update on Android 16: the
	// Super Island and the status bar's chip as well as the lock screen,
	// with buttons of text.
	StyleLive Style = "live"
	// StyleLarge is a layout of the app's own: the time left in large
	// digits and buttons of icons, on the lock screen and in the shade.
	StyleLarge Style = "large"
	// StyleMedia is the system's media card, as a player's: buttons of
	// icons and a progress bar of the time spent out of the round's
	// length, which the system moves on by itself. Experimental.
	StyleMedia Style = "media"
)

// Action is a button of the notification.
type Action string

const (
	// ActionPause pauses the running round.
	ActionPause Action = "pause"
	// ActionResume resumes the paused round.
	ActionResume Action = "resume"
	// ActionOpen opens the app, as a tap on the notification does.
	ActionOpen Action = "open"
	// ActionEnd opens the app to end the round, which asks first.
	ActionEnd Action = "end"
	// ActionEndNow ends the round at once: the button asked to confirm
	// itself, pressed twice, as the media card's does, which may not bring
	// the app to the front.
	ActionEndNow Action = "end-now"
)
