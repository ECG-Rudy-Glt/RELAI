package main

import (
	"strings"
	"testing"
)

func TestIndexNextStepNamesIndexBuild(t *testing.T) {
	got := indexNextStep("widgets")
	if !strings.Contains(got, "relai index build") {
		t.Errorf("indexNextStep should name relai index build, got %q", got)
	}
	if !strings.Contains(got, "widgets") {
		t.Errorf("indexNextStep should name the scaffolded dir, got %q", got)
	}
}
