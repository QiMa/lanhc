package main

import (
	"strings"
	"testing"
)

func TestRunTemplateRejectsFreeform(t *testing.T) {
	_, err := runTemplate("smartctl-long", "device", "/dev/sda; reboot")
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected forbidden-characters error, got %v", err)
	}
	_, err = runTemplate("not-a-template", "device", "/dev/sda")
	if err == nil || !strings.Contains(err.Error(), "unknown command template") {
		t.Fatalf("expected unknown-template error, got %v", err)
	}
	_, err = runTemplate("smartctl-long", "wrong", "/dev/sda")
	if err == nil || !strings.Contains(err.Error(), "requires parameter") {
		t.Fatalf("expected wrong-parameter error, got %v", err)
	}
}
