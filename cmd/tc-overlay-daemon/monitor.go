package main

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func IsXFreerdpRunning() bool {
	// Use pgrep to check if xfreerdp3 is running
	cmd := exec.Command("pgrep", "-f", "xfreerdp3")
	err := cmd.Run()
	// err == nil means process found (exit code 0)
	if err == nil {
		return true
	}
	return false
}

func GetXFreerdpPID() (int, error) {
	cmd := exec.Command("pgrep", "-f", "xfreerdp3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, nil
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		log.Printf("parse pid: %v", err)
		return 0, err
	}

	return pid, nil
}
