package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ambitiousfew/rxd"
)

// APIPollingService create a struct for your service which requires a config field along with any other state
// your service might need to maintain throughout the life of the service.
type APIPollingService struct {
	// fields this specific server uses
	client        *http.Client
	apiBase       string
	retryDuration time.Duration
	maxPollCount  int
}

// NewAPIPollingService just a factory helper function to help create and return a new instance of the service.
func NewAPIPollingService() *APIPollingService {
	return &APIPollingService{
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		// We will check every 10s to see if we can establish a connection to the API when Idle retrying.
		retryDuration: 10 * time.Second,
		apiBase:       "http://localhost:8000",
		maxPollCount:  5,
	}
}

// Idle can be used for some pre-run checks or used to have run fallback to an idle retry state.
func (s *APIPollingService) Idle(c *rxd.ServiceContext) rxd.ServiceResponse {
	for {
		select {
		case <-c.ShutdownSignal():
			return rxd.NewResponse(nil, rxd.StopState)
		case state := <-c.ChangeState():
			// Polling service can wait to be Notified of a specific state change, or even a state to be put into.
			if state == rxd.RunState {
				return rxd.NewResponse(nil, rxd.RunState)
			}
		}
	}
}

// Run is where you want the main logic of your service to run
// when things have been initialized and are ready, this runs the heart of your service.
func (s *APIPollingService) Run(c *rxd.ServiceContext) rxd.ServiceResponse {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	c.Log.Info(fmt.Sprintf("has started to poll"))
	var pollCount int
	for {
		select {
		case <-c.ShutdownSignal():
			return rxd.NewResponse(nil, rxd.StopState)
		case state := <-c.ChangeState():
			// Polling service can wait to be Notified of a specific state change, or even a state to be put into.
			if state == rxd.StopState {
				// if the parent we depend on says they are stopping we will move to exit which also calls Stop() first.
				return rxd.NewResponse(nil, rxd.ExitState)
			}
		case <-timer.C:
			if pollCount > s.maxPollCount {
				c.Log.Info(fmt.Sprintf("has reached its maximum poll count, stopping service"))
				return rxd.NewResponse(nil, rxd.StopState)
			}

			resp, err := s.client.Get(s.apiBase + "/api")
			if err != nil {
				c.Log.Error(err.Error())
				// if we error, reset timer and try again...
				timer.Reset(s.retryDuration)
				continue
			}

			respBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil {
				c.Log.Error(err.Error())
				// we could return to new state: idle or stop or just continue
			}

			var respBody map[string]any
			err = json.Unmarshal(respBytes, &respBody)
			if err != nil {
				c.Log.Error(err.Error())
				// we could return to new state: idle or stop or just continue to keep trying.
			}

			c.Log.Info(fmt.Sprintf("received response from the API: %v", respBody))
			// Increment polling counter
			pollCount++

			// Retry every 10s after the first time.
			timer.Reset(10 * time.Second)
		}
	}
}

// Stop handles anything you might need to do to clean up before ending your service.
func (s *APIPollingService) Stop(c *rxd.ServiceContext) rxd.ServiceResponse {
	// We must return a NewResponse, we use NoopState because it exits with no operation.
	// using StopState would try to recall Stop again.
	return rxd.NewResponse(nil, rxd.ExitState)
}

func (s *APIPollingService) Init(c *rxd.ServiceContext) rxd.ServiceResponse {
	return rxd.NewResponse(nil, rxd.IdleState)
}

// Ensure we meet the interface or error.
var _ rxd.Service = &APIPollingService{}
