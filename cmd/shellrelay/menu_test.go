package main

import (
	"os"
	"strings"
	"testing"
)

func TestMenuRow_NoHighlight(t *testing.T) {
	got := menuRow("hello", false)
	if !strings.Contains(got, "hello") {
		t.Errorf("menuRow output missing text: %q", got)
	}
	if strings.Contains(got, colorSelectedBg) {
		t.Error("non-highlighted row should not contain selection color")
	}
}

func TestMenuRow_Highlight(t *testing.T) {
	got := menuRow("hello", true)
	if !strings.Contains(got, "hello") {
		t.Errorf("menuRow highlight output missing text: %q", got)
	}
	if !strings.Contains(got, colorSelectedBg) {
		t.Error("highlighted row should contain selection color")
	}
}

func TestMenuDivider(t *testing.T) {
	got := menuDivider("╔", "╗", "═")
	if !strings.HasPrefix(got, "╔") {
		t.Errorf("divider should start with left char, got: %q", got)
	}
	if !strings.HasSuffix(got, "╗") {
		t.Errorf("divider should end with right char, got: %q", got)
	}
	if !strings.Contains(got, "═") {
		t.Errorf("divider should contain line char, got: %q", got)
	}
}

func TestClearScreen_NoPanic(t *testing.T) {
	old := os.Stdout
	dev, _ := os.Open(os.DevNull)
	os.Stdout = dev
	defer func() {
		os.Stdout = old
		dev.Close()
	}()
	clearScreen()
}

func TestPrintMenu_NoPanic(t *testing.T) {
	old := os.Stdout
	dev, _ := os.Open(os.DevNull)
	os.Stdout = dev
	defer func() {
		os.Stdout = old
		dev.Close()
	}()

	items := []menuItem{
		{label: "Start"},
		{label: "Stop"},
	}
	printMenu(items, 0)          // first item selected
	printMenu(items, 1)          // second item selected
	printMenu(items, len(items)) // Exit selected
}
