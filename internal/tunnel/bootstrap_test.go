package tunnel

import (
	"testing"
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
