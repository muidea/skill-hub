package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestGetConfigConcurrent(t *testing.T) {
	ResetForTest()
	defer ResetForTest()

	rootDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", rootDir)

	configContent := []byte(`
default_tool: open_code
multi_repo:
  enabled: true
  default_repo: main
  repositories:
    main:
      name: main
      enabled: true
      type: official
`)

	configPath := filepath.Join(rootDir, "config.yaml")
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cfg, err := GetConfig()
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}
			if cfg == nil {
				t.Error("GetConfig() returned nil config")
				return
			}
			if cfg.MultiRepo == nil {
				t.Error("GetConfig() returned nil multi repo config")
				return
			}
			if cfg.MultiRepo.DefaultRepo != "main" {
				t.Errorf("GetConfig() default repo = %q, want %q", cfg.MultiRepo.DefaultRepo, "main")
			}
		}()
	}

	wg.Wait()
}
