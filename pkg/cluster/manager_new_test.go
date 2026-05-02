package cluster

import (
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/provider/mock"
)

func TestNewUsesInjectedProvider(t *testing.T) {
	logger := &log.Logger{Handler: discard.New()}
	injectedProvider := &mock.ClientStub{}

	manager, err := New(logger, "di-seam", injectedProvider)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if manager.Provider() != injectedProvider {
		t.Fatalf("Provider() did not return injected provider")
	}
}

func TestNewReturnsErrorWhenProviderIsNil(t *testing.T) {
	logger := &log.Logger{Handler: discard.New()}

	manager, err := New(logger, "di-seam", nil)
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}

	if manager != nil {
		t.Fatal("New() manager = non-nil, want nil")
	}
}
