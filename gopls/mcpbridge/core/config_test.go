package core

import (
	"testing"

	"golang.org/x/tools/gopls/internal/settings"
)

func TestLoadConfig(t *testing.T) {
	t.Run("EmptyConfig", func(t *testing.T) {
		config, err := LoadConfig([]byte{})
		if err != nil {
			t.Fatalf("Failed to load empty config: %v", err)
		}
		if config == nil {
			t.Fatal("Expected non-nil config")
		}
		if config.Gopls == nil {
			t.Error("Expected Gopls map to be initialized")
		}
	})

	t.Run("NilConfig", func(t *testing.T) {
		config, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("Failed to load nil config: %v", err)
		}
		if config == nil {
			t.Fatal("Expected non-nil config")
		}
	})

	t.Run("ValidConfig", func(t *testing.T) {
		json := `{
			"gopls": {
				"staticcheck": true,
				"buildFlags": ["-tags=integration"]
			},
			"logging": {
				"level": "debug"
			},
			"workdir": "/test/path"
		}`

		config, err := LoadConfig([]byte(json))
		if err != nil {
			t.Fatalf("Failed to load valid config: %v", err)
		}

		if config.Workdir != "/test/path" {
			t.Errorf("Expected workdir /test/path, got %s", config.Workdir)
		}

		if config.Logging == nil || config.Logging.Level != "debug" {
			t.Error("Expected logging level debug")
		}

		if len(config.Gopls) != 2 {
			t.Errorf("Expected 2 gopls options, got %d", len(config.Gopls))
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		_, err := LoadConfig([]byte("{invalid json"))
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestLoadConfigFromMap(t *testing.T) {
	t.Run("ValidMap", func(t *testing.T) {
		m := map[string]any{
			"gopls": map[string]any{
				"staticcheck": true,
			},
			"workdir": "/test",
		}

		config, err := LoadConfigFromMap(m)
		if err != nil {
			t.Fatalf("Failed to load from map: %v", err)
		}

		if config.Workdir != "/test" {
			t.Errorf("Expected workdir /test, got %s", config.Workdir)
		}

		if len(config.Gopls) != 1 {
			t.Errorf("Expected 1 gopls option, got %d", len(config.Gopls))
		}
	})

	t.Run("NilMap", func(t *testing.T) {
		config, err := LoadConfigFromMap(nil)
		if err != nil {
			t.Fatalf("Failed to load nil map: %v", err)
		}
		if config == nil {
			t.Fatal("Expected non-nil config")
		}
	})

	t.Run("EmptyMap", func(t *testing.T) {
		config, err := LoadConfigFromMap(map[string]any{})
		if err != nil {
			t.Fatalf("Failed to load empty map: %v", err)
		}
		if config == nil {
			t.Fatal("Expected non-nil config")
		}
	})
}

func TestApplyGoplsOptions(t *testing.T) {
	t.Run("ApplyStaticcheck", func(t *testing.T) {
		config := &MCPConfig{
			Gopls: map[string]any{
				"staticcheck": true,
			},
		}

		opts := &settings.Options{}
		err := config.ApplyGoplsOptions(opts)
		if err != nil {
			t.Fatalf("Failed to apply options: %v", err)
		}

		if !opts.Staticcheck {
			t.Error("Expected Staticcheck to be enabled")
		}
	})

	t.Run("ApplyBuildFlags", func(t *testing.T) {
		config := &MCPConfig{
			Gopls: map[string]any{
				"buildFlags": []any{"-tags=integration", "--verbose"},
			},
		}

		opts := &settings.Options{}
		err := config.ApplyGoplsOptions(opts)
		if err != nil {
			t.Fatalf("Failed to apply options: %v", err)
		}

		if len(opts.BuildFlags) != 2 {
			t.Errorf("Expected 2 build flags, got %d", len(opts.BuildFlags))
		}
	})

	t.Run("ApplyMultipleOptions", func(t *testing.T) {
		config := &MCPConfig{
			Gopls: map[string]any{
				"staticcheck":   true,
				"verboseOutput": true,
			},
		}

		opts := &settings.Options{}
		err := config.ApplyGoplsOptions(opts)
		if err != nil {
			t.Fatalf("Failed to apply options: %v", err)
		}

		if !opts.Staticcheck {
			t.Error("Expected Staticcheck to be enabled")
		}
		if !opts.VerboseOutput {
			t.Error("Expected VerboseOutput to be enabled")
		}
	})

	t.Run("InvalidOption", func(t *testing.T) {
		config := &MCPConfig{
			Gopls: map[string]any{
				"invalidOptionThatDoesNotExist": true,
			},
		}

		opts := &settings.Options{}
		err := config.ApplyGoplsOptions(opts)
		if err == nil {
			t.Error("Expected error for invalid option")
		}
		t.Logf("Got expected error: %v", err)
	})

	t.Run("NilConfig", func(t *testing.T) {
		opts := &settings.Options{}
		err := (*MCPConfig)(nil).ApplyGoplsOptions(opts)
		if err != nil {
			t.Fatalf("Failed to apply nil config: %v", err)
		}
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		config := &MCPConfig{
			Gopls: map[string]any{},
		}

		opts := &settings.Options{}
		err := config.ApplyGoplsOptions(opts)
		if err != nil {
			t.Fatalf("Failed to apply empty config: %v", err)
		}
	})
}

func TestGoplsOptions(t *testing.T) {
	t.Run("CreateOptions", func(t *testing.T) {
		config := &MCPConfig{
			Gopls: map[string]any{
				"staticcheck": true,
			},
		}

		opts, err := config.GoplsOptions()
		if err != nil {
			t.Fatalf("Failed to create options: %v", err)
		}

		if opts == nil {
			t.Fatal("Expected non-nil options")
		}

		if !opts.Staticcheck {
			t.Error("Expected Staticcheck to be enabled")
		}
	})

	t.Run("NilConfig", func(t *testing.T) {
		opts, err := (*MCPConfig)(nil).GoplsOptions()
		if err != nil {
			t.Fatalf("Failed to create options from nil config: %v", err)
		}
		if opts == nil {
			t.Fatal("Expected non-nil options")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("Expected non-nil default config")
	}

	if config.Gopls == nil {
		t.Error("Expected Gopls map to be initialized")
	}

	if config.Logging == nil {
		t.Error("Expected Logging to be initialized")
	}

	if config.Logging.Level != "info" {
		t.Errorf("Expected default logging level info, got %s", config.Logging.Level)
	}
}
