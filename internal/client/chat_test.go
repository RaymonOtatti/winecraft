package client

import "testing"

func TestChatPanelCapsMessages(t *testing.T) {
	var c ChatPanel
	for i := 0; i < 250; i++ {
		c.AddMessage("msg")
	}
	if got := len(c.msgs); got != 200 {
		t.Fatalf("cap = 200, got %d", got)
	}
}

func TestChatPanelVisibleLines(t *testing.T) {
	c := ChatPanel{}
	c.setMaxLines(216)
	for i := 0; i < 5; i++ {
		c.AddMessage("line")
	}
	if got := len(c.VisibleLines()); got != 5 {
		t.Fatalf("want 5 visible lines, got %d", got)
	}
}

func TestChatPanelScrollClamps(t *testing.T) {
	c := ChatPanel{}
	c.setMaxLines(100) // fits a few lines
	for i := 0; i < 20; i++ {
		c.AddMessage("line")
	}
	c.Scroll(-100)
	if c.offset != 0 {
		t.Fatalf("scroll below zero: got %d", c.offset)
	}
	c.Scroll(1000)
	want := len(c.msgs) - c.maxLines
	if c.offset != want {
		t.Fatalf("scroll above max: got %d, want %d", c.offset, want)
	}
}

func TestChatPanelToggle(t *testing.T) {
	c := ChatPanel{}
	if c.Open {
		t.Fatal("starts closed")
	}
	c.Toggle()
	if !c.Open {
		t.Fatal("toggle opens")
	}
	c.Toggle()
	if c.Open {
		t.Fatal("toggle closes")
	}
}
