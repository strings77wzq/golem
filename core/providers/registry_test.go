package providers

import (
	"testing"
)

func TestRegistry_RegisterAndCreate(t *testing.T) {
	reg := NewRegistry()

	reg.Register("test", func(cfg ProviderConfig) LLMProvider {
		return NewMockProvider(cfg.Vendor)
	})

	provider, err := reg.Create(ProviderConfig{Vendor: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "test" {
		t.Errorf("expected name 'test', got %q", provider.Name())
	}
}

func TestRegistry_UnknownVendor(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Create(ProviderConfig{Vendor: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown vendor")
	}
}

func TestRegistry_Overwrite(t *testing.T) {
	reg := NewRegistry()

	reg.Register("test", func(cfg ProviderConfig) LLMProvider {
		return NewMockProvider("first")
	})

	reg.Register("test", func(cfg ProviderConfig) LLMProvider {
		return NewMockProvider("second")
	})

	provider, _ := reg.Create(ProviderConfig{Vendor: "test"})
	if provider.Name() != "second" {
		t.Errorf("expected overwritten provider 'second', got %q", provider.Name())
	}
}

func TestRegistry_HasVendor(t *testing.T) {
	reg := NewRegistry()

	if reg.HasVendor("test") {
		t.Error("should not have test vendor before registration")
	}

	reg.Register("test", func(cfg ProviderConfig) LLMProvider {
		return NewMockProvider("test")
	})

	if !reg.HasVendor("test") {
		t.Error("should have test vendor after registration")
	}
}

func TestRegistry_Vendors(t *testing.T) {
	reg := NewRegistry()

	reg.Register("a", func(cfg ProviderConfig) LLMProvider { return NewMockProvider("a") })
	reg.Register("b", func(cfg ProviderConfig) LLMProvider { return NewMockProvider("b") })

	vendors := reg.Vendors()
	if len(vendors) != 2 {
		t.Errorf("expected 2 vendors, got %d", len(vendors))
	}
}
