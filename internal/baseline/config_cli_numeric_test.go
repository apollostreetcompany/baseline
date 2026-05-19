package baseline

import "testing"

func TestParseConfigValueParsesIntegers(t *testing.T) {
	value, err := parseConfigValue("900", false)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := value.(int); !ok || got != 900 {
		t.Fatalf("expected int 900, got %[1]T %[1]v", value)
	}
}
