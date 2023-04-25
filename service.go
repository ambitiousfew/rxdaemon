package rxd

import "context"

// State is used to determine the "next state" the service should enter
// when the current state has completed/errored returned. State should
// reflect different states that the interface can enter.
type State string

const (
	// InitState is in the ServiceResponse to inform manager to move us to the Init state (Initial Default).
	InitState State = "init"
	// IdleState is in the ServiceResponse to inform manager to move us to the Idle state
	IdleState State = "idle"
	// RunState is in the ServiceResponse to inform manager to move us to the Run state
	RunState State = "run"
	// StopState is in the ServiceResponse to inform manager to move us to the Stop state
	StopState State = "stop"
	// ExitState is in the ServiceResponse to inform manager to act as the final response type for Stop.
	ExitState State = "exit"
)

type stageFunc func(*ServiceContext) ServiceResponse

type Service interface {
	// Name() string
	Init(*ServiceContext) ServiceResponse
	Idle(*ServiceContext) ServiceResponse
	Run(*ServiceContext) ServiceResponse
	Stop(*ServiceContext) ServiceResponse
	// Reload(*ServiceContext) ServiceResponse
}

// type Service struct {
// 	serviceCtx *ServiceContext

// 	initFunc   stageFunc
// 	idleFunc   stageFunc
// 	runFunc    stageFunc
// 	stopFunc   stageFunc
// 	reloadFunc stageFunc
// }

// NewService creates a new service instance given a name and options.
func NewService(name string, service Service, opts *serviceOpts) *ServiceContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &ServiceContext{
		Ctx:        ctx,
		cancelCtx:  cancel,
		name:       name,
		shutdownC:  make(chan struct{}),
		stateC:     make(chan State),
		opts:       opts,
		isStopped:  true,
		isShutdown: false,
		service:    service,
		dependents: make(map[State][]*ServiceContext),
	}
}

// func (s *Service) Name() string {
// 	return s.serviceCtx.name
// }

// func (s *Service) UsingInitStage(f stageFunc) {
// 	s.initFunc = f
// }

// func (s *Service) UsingIdleStage(f stageFunc) {
// 	s.idleFunc = f
// }

// func (s *Service) UsingRunStage(f stageFunc) {
// 	s.runFunc = f
// }

// func (s *Service) UsingStopStage(f stageFunc) {
// 	s.stopFunc = f
// }

// UsingReloadStage is not implemented yet.
// func (s *Service) UsingReloadStage(f stageFunc) {
// 	s.reloadFunc = f
// }

// func (s *Service) init() ServiceResponse {
// 	return s.initFunc(s.serviceCtx)
// }

// func (s *Service) idle() ServiceResponse {
// 	return s.idleFunc(s.serviceCtx)
// }

// func (s *Service) run() ServiceResponse {
// 	return s.runFunc(s.serviceCtx)
// }

// func (s *Service) stop() ServiceResponse {
// 	return s.stopFunc(s.serviceCtx)
// }

// func (s *Service) reload() ServiceResponse {
// 	return s.reloadFunc(s.serviceCtx)
// }

// Fallback lifecycle stage funcs
func initialize(ctx *ServiceContext) ServiceResponse {
	for {
		select {
		case <-ctx.shutdownC:
			return NewResponse(nil, ExitState)
		default:
			return NewResponse(nil, IdleState)
		}
	}
}

func idle(ctx *ServiceContext) ServiceResponse {
	for {
		select {
		case <-ctx.shutdownC:
			return NewResponse(nil, ExitState)
		default:
			return NewResponse(nil, RunState)
		}
	}
}

func run(ctx *ServiceContext) ServiceResponse {
	for {
		select {
		case <-ctx.shutdownC:
			return NewResponse(nil, ExitState)
		default:
			return NewResponse(nil, StopState)
		}
	}
}

func stop(ctx *ServiceContext) ServiceResponse {
	for {
		select {
		case <-ctx.shutdownC:
			return NewResponse(nil, ExitState)
		default:
			return NewResponse(nil, ExitState)
		}
	}
}

func reload(ctx *ServiceContext) ServiceResponse {
	for {
		select {
		case <-ctx.shutdownC:
			return NewResponse(nil, ExitState)
		default:
			return NewResponse(nil, InitState)
		}
	}
}
