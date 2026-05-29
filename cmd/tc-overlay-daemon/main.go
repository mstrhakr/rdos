package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	tconfigPath := flag.String("tcconfig", "/home/thinclient/tcconfig", "path to tcconfig")
	logPath := flag.String("log", "/tmp/tc-overlay.log", "log file path")
	displayEnv := flag.String("display", "", "X11 DISPLAY")
	flag.Parse()

	// Setup logging
	f, err := os.OpenFile(*logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open log: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("=== tc-overlay-daemon starting ===")

	// Load initial config
	cfg, err := LoadConfig(*tconfigPath)
	if err != nil {
		log.Printf("load config: %v (using defaults)", err)
		cfg = &Config{}
		cfg.SetDefaults()
	}
	log.Printf("loaded config: theme=%s enabled=%v", cfg.Theme, cfg.Enabled)

	// Initialize X11 connection
	display := *displayEnv
	if display == "" {
		display = os.Getenv("DISPLAY")
		if display == "" {
			display = ":0"
		}
	}

	log.Printf("connecting to display %s", display)

	win, err := NewWindow(display, cfg)
	if err != nil {
		log.Fatalf("create window: %v", err)
	}
	defer win.Close()

	log.Println("X11 window created, starting monitor loop")

	// Monitor loop
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	configTicker := time.NewTicker(2 * time.Second)
	defer configTicker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	for {
		select {
		case <-ticker.C:
			// Check if xfreerdp is running
			rdpRunning := IsXFreerdpRunning()
			if rdpRunning != win.RDPWasRunning {
				win.RDPWasRunning = rdpRunning
				if rdpRunning {
					log.Println("xfreerdp detected, showing top bar")
					win.StartTime = time.Now()
					win.Show()
					win.ResetAutoHideTimer()
				} else {
					log.Println("xfreerdp stopped, hiding top bar")
					win.Hide()
				}
			}

			// Handle auto-hide
			if rdpRunning && !win.IsPinned && win.IsVisible {
				if time.Since(win.LastActivity) > time.Duration(win.Config.PinTimeout)*time.Second {
					win.Hide()
					log.Println("auto-hide triggered")
				}
			}

			// Process X11 events
			win.ProcessEvents()

		case <-configTicker.C:
			// Check if tcconfig changed
			newCfg, err := LoadConfig(*tconfigPath)
			if err == nil && newCfg.Theme != cfg.Theme {
				log.Printf("theme changed to %s, updating window", newCfg.Theme)
				cfg = newCfg
				win.SetTheme(cfg.Theme)
			}

		case sig := <-sigChan:
			if sig == syscall.SIGHUP {
				log.Println("received SIGHUP, reloading config")
				newCfg, err := LoadConfig(*tconfigPath)
				if err == nil {
					cfg = newCfg
					win.SetTheme(cfg.Theme)
				}
			} else {
				log.Println("shutdown signal received")
				return
			}
		}
	}
}
