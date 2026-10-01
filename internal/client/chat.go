package client

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image"
)

// ChatPanel implements a simple side‑chat overlay used by the mentor.
// It stores a list of messages (newest last) and allows scrolling.
type ChatPanel struct {
	Open     bool // visibility
	seen     bool // true once the panel has auto-opened on the first message
	msgs     []string
	offset   int // first visible line index
	maxLines int // computed on draw based on screen height
}

// Toggle opens or closes the panel. When opening, reset the scroll offset.
func (c *ChatPanel) Toggle() {
	c.Open = !c.Open
	if c.Open {
		c.offset = 0
	}
}

// AddMessage appends a mentor line. The list is capped to 200 entries to avoid unbounded growth.
func (c *ChatPanel) AddMessage(msg string) {
	const capSize = 200
	if len(c.msgs) >= capSize {
		// drop oldest
		c.msgs = c.msgs[1:]
	}
	c.msgs = append(c.msgs, msg)
}

// Scroll moves the view up/down by delta lines. Positive delta scrolls down.
func (c *ChatPanel) Scroll(delta int) {
	if len(c.msgs) == 0 || c.maxLines == 0 {
		return
	}
	c.offset += delta
	if c.offset < 0 {
		c.offset = 0
	}
	maxOff := len(c.msgs) - c.maxLines
	if maxOff < 0 {
		maxOff = 0
	}
	if c.offset > maxOff {
		c.offset = maxOff
	}
}

// VisibleLines returns the slice of messages that should be rendered.
func (c *ChatPanel) VisibleLines() []string {
	if c.maxLines == 0 || len(c.msgs) == 0 {
		return nil
	}
	start := c.offset
	end := start + c.maxLines
	if end > len(c.msgs) {
		end = len(c.msgs)
	}
	return c.msgs[start:end]
}

// setMaxLines computes how many lines fit given the screen height.
func (c *ChatPanel) setMaxLines(h int) {
	// Rough estimate: each line ~12px (similar to other panels). Leave a margin.
	if h <= 80 {
		c.maxLines = 1
	} else {
		c.maxLines = (h - 80) / 12
	}
	if c.maxLines < 1 {
		c.maxLines = 1
	}
}

// drawChatPanel renders the chat overlay onto the screen.
func (g *Game) drawChatPanel(screen *ebiten.Image) {
	// Ensure maxLines is set based on current height.
	g.chat.setMaxLines(g.h)
	lines := g.chat.VisibleLines()
	// Determine panel dimensions.
	const panelWidth = 200
	// Compute box height: title + margins + lines*12.
	height := 30 + len(lines)*12 + 20
	if height < 70 {
		height = 70
	}
	box := image.Rect((g.w-panelWidth)/2, 30, (g.w+panelWidth)/2, 30+height)
	vector.FillRect(screen, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), dim, false)
	title := "Mentor (M) – press Esc to close"
	drawText(screen, title, (g.w-6*len(title))/2, box.Min.Y+4)
	y := box.Min.Y + 20
	for _, line := range lines {
		drawText(screen, line, box.Min.X+8, y)
		y += 12
	}
}
