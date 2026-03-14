package tunnel

import (
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestNewFromConfig_Nil(t *testing.T) {
	reg, mgr := NewFromConfig(nil)
	if reg == nil || mgr == nil {
		t.Fatal("NewFromConfig(nil) should return non-nil registry and manager")
	}
	list := reg.List()
	if len(list) == 0 {
		t.Error("expected at least bore provider by default")
	}
	found := false
	for _, n := range list {
		if n == "bore" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected bore in provider list")
	}
}

func TestNewFromConfig_WithLocaltunnel(t *testing.T) {
	cfg := &types.TunnelConfig{DefaultProvider: "bore", Providers: map[string]types.TunnelProviderConfig{
		"bore":         {Enabled: true},
		"localtunnel":  {Enabled: true},
	}}
	reg, mgr := NewFromConfig(cfg)
	if reg == nil || mgr == nil {
		t.Fatal("NewFromConfig should return non-nil registry and manager")
	}
	list := reg.List()
	hasBore, hasLocaltunnel := false, false
	for _, n := range list {
		if n == "bore" {
			hasBore = true
		}
		if n == "localtunnel" {
			hasLocaltunnel = true
		}
	}
	if !hasBore {
		t.Error("expected bore in provider list")
	}
	if !hasLocaltunnel {
		t.Error("expected localtunnel in provider list")
	}
}
