package main

import (
	"image/color"
)

type Theme struct {
	Name         string
	BGColor      color.RGBA
	SurfaceColor color.RGBA
	TextColor    color.RGBA
	MutedColor   color.RGBA
	AccentColor  color.RGBA
	LineColor    color.RGBA
}

func GetTheme(name string) *Theme {
	if name == "light" {
		return &Theme{
			Name:         "light",
			BGColor:      color.RGBA{245, 245, 247, 255},
			SurfaceColor: color.RGBA{245, 245, 247, 235}, // ~92% opacity
			TextColor:    color.RGBA{29, 29, 31, 255},
			MutedColor:   color.RGBA{134, 134, 139, 255},
			AccentColor:  color.RGBA{0, 132, 212, 255},
			LineColor:    color.RGBA{0, 0, 0, 30}, // ~12% opacity
		}
	}

	// Default dark theme (matches web UI)
	return &Theme{
		Name:         "dark",
		BGColor:      color.RGBA{4, 8, 13, 255},
		SurfaceColor: color.RGBA{8, 17, 24, 217}, // ~85% opacity
		TextColor:    color.RGBA{234, 248, 255, 255},
		MutedColor:   color.RGBA{163, 191, 203, 255},
		AccentColor:  color.RGBA{92, 224, 255, 255}, // cyan accent
		LineColor:    color.RGBA{147, 223, 255, 91}, // ~36% opacity
	}
}
