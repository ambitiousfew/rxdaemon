package rxd

import (
	"context"
	"time"

	"github.com/ambitiousfew/rxd/intracom"
	"github.com/ambitiousfew/rxd/log"
)

type ServiceWatcher interface {
	WatchAllStates(ServiceFilter) (<-chan ServiceStates, context.CancelFunc)
	WatchAnyServices(action ServiceAction, target State, services ...string) (<-chan ServiceStates, context.CancelFunc)
	WatchAllServices(action ServiceAction, target State, services ...string) (<-chan ServiceStates, context.CancelFunc)
}

type ServiceContext interface {
	context.Context
	ServiceWatcher
	Name() string
	Log(level log.Level, message string, fields ...log.Field)
	// With returns a new ServiceContext with the given fields appended to the existing fields.
	With(fields ...log.Field) ServiceContext
}

type serviceContext struct {
	context.Context
	name   string
	fields []log.Field
	logC   chan<- DaemonLog
	// ic       intracom.Intracom[ServiceStates]
	icStates intracom.Topic[ServiceStates]
}

// NewServiceContext returns a new child ServiceContext with the given name and fields.
// The new child context will have the same log channel and internal communication as the parent.
func ServiceContextWith(parent ServiceContext) ServiceContext {
	return serviceContext{
		Context: parent,
		name:    parent.Name(),
		fields:  parent.(serviceContext).fields,
		logC:    parent.(serviceContext).logC,
		// ic:      parent.(serviceContext).ic,
		icStates: parent.(serviceContext).icStates,
	}
}

// NewServiceContext returns a new child ServiceContext sharing the same fields and log channel as the parent.
// The new child context will keep the same name as the parent.
func ServiceContextWithCancel(parent ServiceContext) (ServiceContext, context.CancelFunc) {
	logC := parent.(serviceContext).logC
	icStates := parent.(serviceContext).icStates
	return newServiceContextWithCancel(parent, parent.Name(), logC, icStates)
}

func newServiceContextWithCancel(parent context.Context, name string, logC chan<- DaemonLog, icStates intracom.Topic[ServiceStates]) (ServiceContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	fields := []log.Field{}
	if name != "" {
		fields = append(fields, log.String("service", name))
	}
	return serviceContext{
		Context:  ctx,
		name:     name,
		logC:     logC,
		fields:   fields,
		icStates: icStates,
	}, cancel
}

// With returns a new child ServiceContext with the given fields appended to the existing fields.
// The new child context will have the same name as the parent.
func (sc serviceContext) With(fields ...log.Field) ServiceContext {
	return serviceContext{
		Context:  sc.Context,
		name:     sc.name,
		fields:   append(fields, sc.fields...),
		logC:     sc.logC,
		icStates: sc.icStates,
	}
}

func (sc serviceContext) Name() string {
	return sc.name
}

func (sc serviceContext) Log(level log.Level, message string, fields ...log.Field) {
	sc.logC <- DaemonLog{
		Name:    sc.name,
		Level:   level,
		Message: message,
		Fields:  append(fields, sc.fields...),
	}
}

func (sc serviceContext) Deadline() (deadline time.Time, ok bool) {
	return sc.Context.Deadline()
}

func (sc serviceContext) Done() <-chan struct{} {
	return sc.Context.Done()
}

func (sc serviceContext) Err() error {
	return sc.Context.Err()
}

func (sc serviceContext) Value(key interface{}) interface{} {
	return sc.Context.Value(key)
}

func (sc serviceContext) WatchAllServices(action ServiceAction, target State, services ...string) (<-chan ServiceStates, context.CancelFunc) {
	ch := make(chan ServiceStates, 1)
	watchCtx, cancel := context.WithCancel(sc)

	go func(ctx context.Context) {
		defer cancel()
		// subscribe to the internal states on behalf of the service context given.
		consumer := internalStatesConsumer(action, target, sc.name)
		sub, err := sc.icStates.Subscribe(intracom.SubscriberConfig{
			ConsumerGroup: consumer,
			ErrIfExists:   false,
			BufferSize:    1,
			BufferPolicy:  intracom.DropOldest,
		})
		if err != nil {
			sc.Log(log.LevelError, "failed to subscribe to internal states: "+err.Error())
			return
		}

		for {
			select {
			case <-ctx.Done():
				return

			case states, open := <-sub:
				if !open {
					return
				}

				interestedServices := make(ServiceStates, len(services))
				for _, name := range services {
					switch action {
					case Entering:
						// entering is the same as the target state, so we check for the exact target state.
						if val, ok := states[name]; ok && val == target {
							interestedServices[name] = val
						}

					case Exiting:
						// exiting is the opposite of entering, so we check for the opposite of the target state.
						if val, ok := states[name]; ok && val != target {
							interestedServices[name] = val
						}
					default:
						// ignore
						continue
					}
				}

				// if we found all those we care about.
				if len(interestedServices) == len(services) {
					select {
					case <-ctx.Done():
						return
					case ch <- interestedServices: // send out the states
						// TODO: should we stop here, or reset and keep collecting the interested services?
					}
				}

			}
		}
	}(watchCtx)

	return ch, cancel
}

func (sc serviceContext) WatchAnyServices(action ServiceAction, target State, services ...string) (<-chan ServiceStates, context.CancelFunc) {
	ch := make(chan ServiceStates, 1)
	watchCtx, cancel := context.WithCancel(sc)

	go func(ctx context.Context) {
		defer cancel()

		// subscribe to the internal states on behalf of the service context given.
		consumer := internalStatesConsumer(action, target, sc.name)
		sub, err := sc.icStates.Subscribe(intracom.SubscriberConfig{
			ConsumerGroup: consumer,
			ErrIfExists:   false,
			BufferSize:    1,
			BufferPolicy:  intracom.DropOldest,
		})
		if err != nil {
			sc.Log(log.LevelError, "failed to subscribe to internal states: "+err.Error())
			return
		}
		// sub, unsub := sc.ic.Subscribe(ctx, intracom.SubscriberConfig{
		// 	Topic:         internalServiceStates,
		// 	ConsumerGroup: consumer,
		// 	BufferSize:    1,
		// 	BufferPolicy:  intracom.DropOldest,
		// })

		// defer unsub()

		for {
			select {
			case <-ctx.Done():
				return

			case states, open := <-sub:
				if !open {
					return
				}

				interestedServices := make(ServiceStates, len(services))
				for _, service := range services {
					switch action {
					case Entering:
						if val, ok := states[service]; ok && val == target {
							interestedServices[service] = val
						}

					case Exiting:
						if val, ok := states[service]; ok && val != target {
							interestedServices[service] = val
						}
					}
				}

				// if we found all those we care about.
				if len(interestedServices) > 0 {
					select {
					case <-ctx.Done(): // user cancelled us
						return
					case ch <- interestedServices: // send out the states we cared about
					}
				}
			}

		}

	}(watchCtx)

	return ch, cancel
}

func (sc serviceContext) WatchAllStates(filter ServiceFilter) (<-chan ServiceStates, context.CancelFunc) {
	ch := make(chan ServiceStates, 1)
	watchCtx, cancel := context.WithCancel(sc)

	go func(ctx context.Context) {
		defer cancel()

		consumer := internalAllStatesConsumer(sc.name)
		sub, err := sc.icStates.Subscribe(intracom.SubscriberConfig{
			ConsumerGroup: consumer,
			ErrIfExists:   false,
			BufferSize:    1,
			BufferPolicy:  intracom.DropOldest,
		})
		if err != nil {
			sc.Log(log.LevelError, "failed to subscribe to internal states: "+err.Error())
			return
		}
		// subscribe to the internal states on behalf of the service context given.
		// sub, unsub := sc.ic.Subscribe(ctx, intracom.SubscriberConfig{
		// 	Topic:         internalServiceStates,
		// 	ConsumerGroup: consumer,
		// 	BufferSize:    1,
		// 	BufferPolicy:  intracom.DropOldest,
		// })
		// defer unsub()

		for {
			select {
			case <-ctx.Done():
				return

			case states, open := <-sub:
				if !open {
					return
				}

				// if no filters are given or mode is set to none, then we just send out all the states we have.
				if len(filter.Names) == 0 || filter.Mode == None {
					select {
					case <-ctx.Done():
						return
					case ch <- states:
						// no filtering applied, send out all the states we have.
					}
					continue
				}

				// if we have filters, then we need to filter the states we have.
				filteredInterests := make(ServiceStates, len(filter.Names))
				for name, state := range states {
					switch filter.Mode {
					case Include:
						// if the FilterSet given contains the service name, then we include it.
						if _, ok := filter.Names[name]; ok {
							filteredInterests[name] = state
						}

					case Exclude:
						// if the FilterSet given does not contain the service name, then we include it.
						if _, ok := filter.Names[name]; !ok {
							filteredInterests[name] = state
						}
					}
				}

				select {
				case <-ctx.Done():
					return
				case ch <- filteredInterests: // send out the states
				}
			}
		}
	}(watchCtx)

	return ch, cancel
}
