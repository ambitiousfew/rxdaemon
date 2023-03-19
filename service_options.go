package rxd

// RunPolicy service option type representing the run policy of a given service
// basically controlling different ways of stopping a service like running only once when it succeeds
// without an error on Run
type RunPolicy string

const (
	// RunUntilStoppedPolicy will continue to run the service until a StopState is returned at some point
	RunUntilStoppedPolicy RunPolicy = "until_stopped"
	// RunOnceIfSuccessPolicy will continue to run the service as long fails happen, use for running a service once successfully
	RunOnceIfSuccessPolicy RunPolicy = "run_once_success"
	// RunOncePolicy will only allow the a single Run to take place regardless of success/failure
	RunOncePolicy RunPolicy = "run_once_unbiased"
)

// ServiceOption are simply using an Option pattern to customize options
// such as restart policies, timeouts for a given service and how it should run.
type ServiceOption func(*ServiceOpts)

// UsingRunPolicy applies a given policy to the ServiceOption instance
func UsingRunPolicy(policy RunPolicy) ServiceOption {
	return func(so *ServiceOpts) {
		so.RunPolicy = policy
	}
}

// ServiceOpts will allow for customizations of how a service runs and should always have
// a reasonable default to fallback if the case one isnt provided.
// This would be set by the ServiceConfig upon creation.
type ServiceOpts struct {
	RunPolicy RunPolicy
}
