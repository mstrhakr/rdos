package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Window struct {
	Conn           *xgb.Conn
	Window         xproto.Window
	Screen         *xproto.ScreenInfo
	GC             xproto.Gcontext
	Config         *Config
	Theme          *Theme
	IsVisible      bool
	IsPinned       bool
	RDPWasRunning  bool
	LastActivity   time.Time
	DragStart      image.Point
	IsDragging     bool
	Position       image.Point
	Width          int
	Height         int
	StartTime      time.Time
	LastRedrawTime time.Time
}

func NewWindow(display string, cfg *Config) (*Window, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("x11 connect: %w", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)

	w := &Window{
		Conn:           conn,
		Screen:         screen,
		Config:         cfg,
		Theme:          GetTheme(cfg.Theme),
		Width:          int(screen.WidthInPixels),
		Height:         48,
		Position:       image.Point{0, 0},
		IsVisible:      false,
		IsPinned:       false,
		LastActivity:   time.Now(),
		StartTime:      time.Now(),
		LastRedrawTime: time.Now(),
	}

	// Create window
	window, err := xproto.NewWindowId(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("new window id: %w", err)
	}

	eventMask := uint32(xproto.EventMaskExposure |
		xproto.EventMaskPointerMotion |
		xproto.EventMaskButtonPress |
		xproto.EventMaskButtonRelease |
		xproto.EventMaskStructureNotify)

	xproto.CreateWindow(conn, screen.RootDepth, window, screen.Root,
		0, 0,
		uint16(w.Width), uint16(w.Height),
		0, xproto.WindowClassInputOutput,
		screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{0x00000000, eventMask},
	)

	w.Window = window

	// Set window properties (best effort - skip window manager hints for MVP)
	// Proper implementation would use xproto.InternAtom() with .Reply()
	// For now, just create the graphics context and proceed

	// Create graphics context
	gc, err := xproto.NewGcontextId(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("new gc: %w", err)
	}

	xproto.CreateGC(conn, gc, xproto.Drawable(window),
		xproto.GcForeground|xproto.GcBackground|xproto.GcGraphicsExposures,
		[]uint32{0xffffffff, 0x00000000, 0})

	w.GC = gc

	log.Println("window created successfully")
	return w, nil
}

func (w *Window) Show() {
	if w.IsVisible {
		return
	}
	xproto.MapWindow(w.Conn, w.Window)
	w.IsVisible = true
	w.LastActivity = time.Now()
	w.Redraw()
	log.Println("window shown")
}

func (w *Window) Hide() {
	if !w.IsVisible {
		return
	}
	xproto.UnmapWindow(w.Conn, w.Window)
	w.IsVisible = false
	log.Println("window hidden")
}

func (w *Window) ResetAutoHideTimer() {
	w.LastActivity = time.Now()
}

func (w *Window) SetTheme(themeName string) {
	w.Theme = GetTheme(themeName)
	w.Config.Theme = themeName
	if w.IsVisible {
		w.Redraw()
	}
	log.Printf("theme set to %s", themeName)
}

func (w *Window) Close() {
	xproto.FreeGC(w.Conn, w.GC)
	xproto.DestroyWindow(w.Conn, w.Window)
	w.Conn.Close()
}

func (w *Window) Redraw() {
	if !w.IsVisible {
		return
	}

	// Throttle redraws to 60fps max
	if time.Since(w.LastRedrawTime) < 16*time.Millisecond {
		return
	}
	w.LastRedrawTime = time.Now()

	// Clear window by filling with background color
	setGCForegroundColor(w.Conn, w.GC, w.Theme.BGColor)
	xproto.PolyFillRectangle(w.Conn, xproto.Drawable(w.Window), w.GC,
		[]xproto.Rectangle{{
			X:      0,
			Y:      0,
			Width:  uint16(w.Width),
			Height: uint16(w.Height),
		}},
	)

	// Draw border lines
	setGCForegroundColor(w.Conn, w.GC, w.Theme.LineColor)
	xproto.PolySegment(w.Conn, xproto.Drawable(w.Window), w.GC,
		[]xproto.Segment{
			{X1: 0, Y1: 0, X2: int16(w.Width - 1), Y2: 0},                                     // top
			{X1: 0, Y1: int16(w.Height - 1), X2: int16(w.Width - 1), Y2: int16(w.Height - 1)}, // bottom
			{X1: 0, Y1: 0, X2: 0, Y2: int16(w.Height - 1)},                                    // left
			{X1: int16(w.Width - 1), Y1: 0, X2: int16(w.Width - 1), Y2: int16(w.Height - 1)},  // right
		},
	)

	// Draw session time on the left (HH:MM:SS)
	// TODO: Implement proper text rendering using xgraphics, Cairo, or Pango
	// For now, skip drawing the clock text to avoid unreadable placeholder boxes
	elapsed := time.Since(w.StartTime)
	_ = elapsed // TODO: display when text rendering is available

	// Draw pin button on right side (32x32 button at right-40)
	pinX := int16(w.Width - 40)
	pinY := int16(8)
	pinW := uint16(32)
	pinH := uint16(32)

	// Draw pin button background (filled rectangle)
	setGCForegroundColor(w.Conn, w.GC, w.Theme.AccentColor)
	xproto.PolyFillRectangle(w.Conn, xproto.Drawable(w.Window), w.GC,
		[]xproto.Rectangle{{
			X:      pinX,
			Y:      pinY,
			Width:  pinW,
			Height: pinH,
		}},
	)

	// Draw pin button border
	setGCForegroundColor(w.Conn, w.GC, w.Theme.LineColor)
	xproto.PolySegment(w.Conn, xproto.Drawable(w.Window), w.GC,
		[]xproto.Segment{
			{X1: pinX, Y1: pinY, X2: pinX + int16(pinW-1), Y2: pinY},                                 // top
			{X1: pinX, Y1: pinY + int16(pinH-1), X2: pinX + int16(pinW-1), Y2: pinY + int16(pinH-1)}, // bottom
			{X1: pinX, Y1: pinY, X2: pinX, Y2: pinY + int16(pinH-1)},                                 // left
			{X1: pinX + int16(pinW-1), Y1: pinY, X2: pinX + int16(pinW-1), Y2: pinY + int16(pinH-1)}, // right
		},
	)

	// TODO: Draw pin label text ("PIN"/"UNPIN") once text rendering is implemented
	// For now, skip drawing placeholder boxes to avoid visual confusion
}

func (w *Window) drawImageOnWindow(img *image.RGBA) {
	// MVP: Use XFillRectangle to draw background and basic shapes
	// For full image rendering, would use cairo or similar
	// This is sufficient for top bar MVP
}

// setGCForegroundColor sets the graphics context foreground color
func setGCForegroundColor(conn *xgb.Conn, gc xproto.Gcontext, col color.RGBA) {
	// Convert RGBA to X11 pixel value (RGB 24-bit)
	pixel := uint32((uint32(col.R) << 16) | (uint32(col.G) << 8) | uint32(col.B))
	xproto.ChangeGC(conn, gc, xproto.GcForeground, []uint32{pixel})
}


func (w *Window) ProcessEvents() {
	for {
		ev, err := w.Conn.PollForEvent()
		if err != nil || ev == nil {
			break
		}

		switch e := ev.(type) {
		case xproto.ButtonPressEvent:
			w.handleButtonPress(e)
		case xproto.MotionNotifyEvent:
			w.handleMotionNotify(e)
		case xproto.ButtonReleaseEvent:
			w.handleButtonRelease(e)
		case xproto.ExposeEvent:
			if w.IsVisible {
				w.Redraw()
			}
		}
	}
}

func (w *Window) handleButtonPress(e xproto.ButtonPressEvent) {
	w.LastActivity = time.Now()

	// Check if click is on pin button
	pinX := w.Width - 40
	pinY := 8
	if int(e.EventX) >= pinX && int(e.EventX) <= pinX+32 &&
		int(e.EventY) >= pinY && int(e.EventY) <= pinY+32 {
		w.IsPinned = !w.IsPinned
		log.Printf("pin toggled: %v", w.IsPinned)
		w.Redraw()
		return
	}

	// Otherwise start drag
	if int(e.EventY) <= w.Height {
		w.IsDragging = true
		w.DragStart = image.Point{int(e.EventX), int(e.EventY)}
	}
}

func (w *Window) handleMotionNotify(e xproto.MotionNotifyEvent) {
	w.LastActivity = time.Now()

	if w.IsDragging {
		deltaX := int(e.EventX) - w.DragStart.X
		deltaY := int(e.EventY) - w.DragStart.Y

		w.Position.X += deltaX
		w.Position.Y += deltaY

		// Constrain to screen
		if w.Position.X < 0 {
			w.Position.X = 0
		}
		if w.Position.X+w.Width > int(w.Screen.WidthInPixels) {
			w.Position.X = int(w.Screen.WidthInPixels) - w.Width
		}
		if w.Position.Y < 0 {
			w.Position.Y = 0
		}
		if w.Position.Y+w.Height > int(w.Screen.HeightInPixels) {
			w.Position.Y = int(w.Screen.HeightInPixels) - w.Height
		}

		xproto.ConfigureWindow(w.Conn, w.Window,
			xproto.ConfigWindowX|xproto.ConfigWindowY,
			[]uint32{uint32(w.Position.X), uint32(w.Position.Y)})

		w.DragStart = image.Point{int(e.EventX), int(e.EventY)}
	}
}

func (w *Window) handleButtonRelease(e xproto.ButtonReleaseEvent) {
	w.IsDragging = false
}
