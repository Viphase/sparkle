package theme

// PastelDark — soft purples on deep aubergine. Default theme.
func PastelDark() Theme {
	return Theme{
		Name:         "pastel-dark",
		Background:   "#1a1625",
		Surface:      "#231a36",
		Overlay:      "#2d2342",
		Border:       "#3a2f50",
		BorderFocus:  "#c4b5fd",
		Foreground:   "#ece7f2",
		Muted:        "#9f94b8",
		Subtle:       "#6e6585",
		Primary:      "#c4b5fd",
		PrimaryGlow:  "#a78bfa",
		Accent:       "#f9a8d4",
		AccentGlow:   "#ec4899",
		GradientFrom: "#a78bfa",
		GradientTo:   "#f0abfc",
		Success:      "#86efac",
		Warning:      "#fcd34d",
		Danger:       "#fca5a5",
		Info:         "#93c5fd",
	}
}

// PastelLight — warm cream surface with violet/rose accents.
func PastelLight() Theme {
	return Theme{
		Name:         "pastel-light",
		Background:   "#faf7ff",
		Surface:      "#f3eefb",
		Overlay:      "#e8e0f4",
		Border:       "#d6cce6",
		BorderFocus:  "#7c3aed",
		Foreground:   "#2d2438",
		Muted:        "#6e6585",
		Subtle:       "#a59cba",
		Primary:      "#7c3aed",
		PrimaryGlow:  "#5b21b6",
		Accent:       "#db2777",
		AccentGlow:   "#9d174d",
		GradientFrom: "#7c3aed",
		GradientTo:   "#db2777",
		Success:      "#15803d",
		Warning:      "#b45309",
		Danger:       "#b91c1c",
		Info:         "#1d4ed8",
	}
}

// Nova — high-energy cyan-to-magenta on near-black.
func Nova() Theme {
	return Theme{
		Name:         "nova",
		Background:   "#0a0a1f",
		Surface:      "#13132e",
		Overlay:      "#1a1a3e",
		Border:       "#2a2a52",
		BorderFocus:  "#22d3ee",
		Foreground:   "#e0f7ff",
		Muted:        "#8b9bc4",
		Subtle:       "#5a6a8a",
		Primary:      "#22d3ee",
		PrimaryGlow:  "#06b6d4",
		Accent:       "#f472b6",
		AccentGlow:   "#ec4899",
		GradientFrom: "#22d3ee",
		GradientTo:   "#f472b6",
		Success:      "#34d399",
		Warning:      "#fbbf24",
		Danger:       "#f87171",
		Info:         "#60a5fa",
	}
}

// AllPalettes returns the built-in themes in display order.
func AllPalettes() []Theme {
	return []Theme{PastelDark(), PastelLight(), Nova()}
}
