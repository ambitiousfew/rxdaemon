package rxd

import (
	"context"

	"github.com/ambitiousfew/rxd/log"
)

// TODO: This is a basic implementation for interacting with a service manager.
// Actual interactions with a service manager are a little more involved.
// This implementation really only provides a way to notify watchdog if its enabled.
// This is not cross-platform and is only for linux systems that use systemd.
//
// Ideally there should be a subpackage that provides an interface for running
// as a system daemon on different platforms. This subpackage would need to
// provide a factory that can hand back a given struct that meets the interface
// based on the runtime value(s).
//
// Basically if we build with linux tags, we get the systemd implementation.
// if we build with windows tags, we get the windows service implementation.
// Because I want rxd to be cross-platform, this is already a consideration
// for the future. Currently its a big lift and current needs are only for linux.

type SystemNotifier interface {
	Start(ctx context.Context, logger log.Logger) error
	Notify(state NotifyState) error
}

const (
	NotifyStateStopped NotifyState = iota
	NotifyStateStopping
	NotifyStateRestarting
	NotifyStateReloading
	NotifyStateReady
	NotifyStateAlive
)

type NotifyState uint8

func (s NotifyState) String() string {
	switch s {
	case NotifyStateStopped:
		return "STOPPED"
	case NotifyStateStopping:
		return "STOPPING"
	case NotifyStateRestarting:
		return "RESTARTING"
	case NotifyStateReloading:
		return "RELOADING"
	case NotifyStateReady:
		return "READY"
	case NotifyStateAlive:
		return "ALIVE"
	default:
		return ""
	}
}
