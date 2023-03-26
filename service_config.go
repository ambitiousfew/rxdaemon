package rxd

import (
	"fmt"
	"sync"
)

// ServiceConfig all services will require a config as a *ServiceConfig in their service struct.
// This config contains preconfigured shutdown channel,
type ServiceConfig struct {
	name string

	opts *serviceOpts

	// ShutdownC is provided to each service to give the ability to watch for a shutdown signal.
	ShutdownC chan struct{}

	StateC chan State

	// Logging channel for manage to attach to services to use
	logC chan LogMessage

	// isStopped is a flag to tell is if we have been asked to run the Stop state
	isStopped bool
	// isShutdown is a flag that is true if close() has been called on the ShutdownC for the service in manager shutdown method
	isShutdown bool
	// mu is primarily used for mutations against isStopped and isShutdown between manager and wrapped service logic
	mu sync.Mutex
}

// NotifyStateChange takes a state and iterates over all services added via UsingServiceNotify, if any
func (cfg *ServiceConfig) NotifyStateChange(state State) {
	// If we dont have any services to notify, dont try.
	if cfg.opts.serviceNotify == nil {
		return
	}

	cfg.opts.serviceNotify.notify(state, cfg.logC)
}

func (cfg *ServiceConfig) setIsStopped(value bool) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.isStopped = value
}

func (cfg *ServiceConfig) shutdown() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if !cfg.isShutdown {
		close(cfg.ShutdownC)
		close(cfg.StateC)
		cfg.isShutdown = true
	}
}

// LogInfo takes a string message and sends it down the logC channel as a LogMessage type with log level of Info
func (cfg *ServiceConfig) LogInfo(message string) {
	cfg.logC <- NewLog(serviceLog(cfg, message), Info)
}

// LogDebug takes a string message and sends it down the logC channel as a LogMessage type with log level of Debug
func (cfg *ServiceConfig) LogDebug(message string) {
	cfg.logC <- NewLog(serviceLog(cfg, message), Debug)
}

// LogError takes a string message and sends it down the logC channel as a LogMessage type with log level of Error
func (cfg *ServiceConfig) LogError(message string) {
	cfg.logC <- NewLog(serviceLog(cfg, message), Error)
}

// serviceLog is a helper that prefixes log string messages with the service name
func serviceLog(cfg *ServiceConfig, message string) string {
	return fmt.Sprintf("%s %s", cfg.name, message)
}

// NewServiceConfig will apply all options in the order given prior to creating the ServiceConfig instance created.
func NewServiceConfig(name string, options ...ServiceOption) *ServiceConfig {
	// Default policy to restart immediately (3s) and always try to restart itself.
	opts := &serviceOpts{
		// RestartPolicy:  Always,
		// RestartTimeout: 3 * time.Second,
		runPolicy: RunUntilStoppedPolicy,
	}

	// Apply all functional options to update defaults.
	for _, option := range options {
		option(opts)
	}

	return &ServiceConfig{
		name:       name,
		ShutdownC:  make(chan struct{}),
		StateC:     make(chan State),
		opts:       opts,
		isStopped:  true,
		isShutdown: false,
	}
}
