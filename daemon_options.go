package rxd

import "os"

type DaemonOption func(*daemon)

// UsingAliveTimeout sets the interval in seconds for when the daemon should report that it is still alive
// to the service manager. If the value is set to 0, the daemon will not interact with the service manager.
func UsingReportAlive(timeoutSecs uint64) DaemonOption {
	return func(d *daemon) {
		d.reportAliveSecs = timeoutSecs
	}
}

// UsingOSSignals sets the OS signals that the daemon should listen for. If no signals are provided, the daemon
// will listen for SIGINT and SIGTERM by default.
func UsingOSSignals(signals ...os.Signal) DaemonOption {
	return func(d *daemon) {
		d.signals = signals
	}
}