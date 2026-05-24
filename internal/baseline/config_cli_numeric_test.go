package baseline

import (
	"strings"
	"testing"
)

func TestParseConfigValueParsesIntegers(t *testing.T) {
	value, err := parseConfigValue("900", false)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := value.(int); !ok || got != 900 {
		t.Fatalf("expected int 900, got %[1]T %[1]v", value)
	}
}

func TestDynamicMemorySeedDoesNotBecomeExpectedFact(t *testing.T) {
	if got := expectedFact("dynamic"); got != nil {
		t.Fatalf("dynamic memory seed should not produce a brittle expected fact, got %+v", got)
	}
}

func TestValidateConfigFlagsHermesStaticDefaultProjectSeed(t *testing.T) {
	cfg := defaultConfig()
	cfg.Target.Runtime = "hermes"
	cfg.Target.Entity = "agent:hermes"
	cfg.MemorySeeds = defaultMemorySeeds()
	issues := validateConfig(cfg)
	var found bool
	for _, issue := range issues {
		if strings.Contains(issue, "memory_seeds.project") && strings.Contains(issue, "dynamic") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Hermes config validation to flag static default project seed, got %+v", issues)
	}
}
