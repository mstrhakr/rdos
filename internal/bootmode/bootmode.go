package bootmode

import "strings"

const (
	ModeWeb    = "web"
	ModeLegacy = "legacy"
)

func ParseKernelCmdline(cmdline string) string {
	for _, token := range strings.Fields(strings.TrimSpace(cmdline)) {
		switch token {
		case "rdos.ui=web":
			return ModeWeb
		case "rdos.ui=legacy":
			return ModeLegacy
		}
	}
	return ""
}

func Resolve(cmdline, configuredDefault string) string {
	if fromCmdline := ParseKernelCmdline(cmdline); fromCmdline != "" {
		return fromCmdline
	}

	if configuredDefault == ModeLegacy {
		return ModeLegacy
	}
	return ModeWeb
}
