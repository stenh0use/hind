package cluster

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
	"github.com/stenh0use/hind/pkg/provider/mock"
)

func TestStopWithOptions(t *testing.T) {
	tests := []struct {
		name             string
		statuses         map[string]string
		force            bool
		stopErrFor       string
		wantStopped      int
		wantAlready      int
		wantFailed       int
		wantPreFailed    int
		wantFailListSize int
	}{
		{name: "already stopped idempotent", statuses: map[string]string{"n1": provider.Stopped.String(), "n2": provider.Stopped.String()}, wantAlready: 2},
		{name: "partial failure continues", statuses: map[string]string{"n1": provider.Running.String(), "n2": provider.Running.String()}, stopErrFor: "n2", wantStopped: 1, wantFailed: 1, wantFailListSize: 1},
		{name: "unhealthy counted", statuses: map[string]string{"n1": provider.Error.String(), "n2": provider.Running.String()}, wantStopped: 1, wantAlready: 1, wantPreFailed: 1},
		{name: "force uses kill", statuses: map[string]string{"n1": provider.Running.String()}, force: true, wantStopped: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stops := 0
			kills := 0
			client := &mock.ClientStub{}
			client.InspectContainerFn = func(_ context.Context, name string) (*provider.ContainerInfo, error) {
				status, ok := tt.statuses[name]
				if !ok {
					return nil, nil
				}
				return &provider.ContainerInfo{Name: name, Status: status}, nil
			}
			client.StopContainerFn = func(_ context.Context, name string) error {
				stops++
				if name == tt.stopErrFor {
					return errors.New("stop failed")
				}
				return nil
			}
			client.KillContainerFn = func(_ context.Context, _ string) error {
				kills++
				return nil
			}

			m := &Manager{
				logger:   &log.Logger{Handler: discard.New(), Level: log.ErrorLevel},
				provider: client,
				config:   &config.Cluster{Name: tt.name, Nodes: []config.Node{{Name: "n1"}, {Name: "n2"}}},
			}

			res, err := m.StopWithOptions(context.Background(), StopOptions{Force: tt.force})
			if err != nil {
				t.Fatalf("StopWithOptions() error = %v", err)
			}
			if res.StoppedCount != tt.wantStopped || res.AlreadyStoppedCount != tt.wantAlready || res.FailedCount != tt.wantFailed || res.FailedPreStopCount != tt.wantPreFailed {
				t.Fatalf("unexpected result: %+v", res)
			}
			if len(res.Failures) != tt.wantFailListSize {
				t.Fatalf("failures len = %d, want %d", len(res.Failures), tt.wantFailListSize)
			}
			if tt.force {
				if kills == 0 {
					t.Fatalf("expected kill calls")
				}
				if stops != 0 {
					t.Fatalf("expected no stop calls when force=true")
				}
			}
		})
	}
}
