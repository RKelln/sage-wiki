package config

import "testing"

func TestResultMaxCharsOrDefault(t *testing.T) {
	if got := (SearchConfig{}).ResultMaxCharsOrDefault(); got != 2000 {
		t.Errorf("zero value should default to 2000, got %d", got)
	}
	if got := (SearchConfig{ResultMaxChars: 500}).ResultMaxCharsOrDefault(); got != 500 {
		t.Errorf("explicit value should be returned, got %d", got)
	}
	if got := (SearchConfig{ResultMaxChars: -1}).ResultMaxCharsOrDefault(); got != 2000 {
		t.Errorf("negative value should default to 2000, got %d", got)
	}
}
