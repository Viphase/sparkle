package theme

import "strings"

// PastelDark — cool blue on a quiet dark base. Default theme.
func PastelDark() Theme {
	return Theme{
		Name:         "pastel-dark",
		Background:   "#0d1524",
		Surface:      "#121d2e",
		Overlay:      "#19283d",
		Border:       "#29435f",
		BorderFocus:  "#72c7ff",
		Foreground:   "#e6f2ff",
		Muted:        "#9fb5cc",
		Subtle:       "#60758d",
		Primary:      "#72c7ff",
		PrimaryGlow:  "#4aa8e8",
		Accent:       "#8edfd2",
		AccentGlow:   "#54bfb2",
		GradientFrom: "#8edfd2",
		GradientTo:   "#72c7ff",
		Success:      "#8fd5a6",
		Warning:      "#d6bf72",
		Danger:       "#de8d8d",
		Info:         "#72c7ff",
	}
}

// PastelLight — warm aged-paper surface with muted teal-blue accents.
// The background leans amber to avoid the harshness of pure white terminals.
func PastelLight() Theme {
	return Theme{
		Name:         "pastel-light",
		Background:   "#e8e0d2",
		Surface:      "#ddd6c5",
		Overlay:      "#d0c9b6",
		Border:       "#b8b0a0",
		BorderFocus:  "#4270b8",
		Foreground:   "#1a1e2a",
		Muted:        "#5e6a78",
		Subtle:       "#8c9aa8",
		Primary:      "#2e5ea8",
		PrimaryGlow:  "#1f4d96",
		Accent:       "#206070",
		AccentGlow:   "#155060",
		GradientFrom: "#206070",
		GradientTo:   "#2e5ea8",
		Success:      "#256638",
		Warning:      "#7a5200",
		Danger:       "#9a1a28",
		Info:         "#2e5ea8",
	}
}

// Nova — cosmic burst: deep space purple with electric neon energy.
// Vivid magenta/cyan/gold make it unmistakably distinct from the dark theme.
func Nova() Theme {
	return Theme{
		Name:         "nova",
		Background:   "#060010",
		Surface:      "#0f0025",
		Overlay:      "#1a003d",
		Border:       "#4400aa",
		BorderFocus:  "#ff00ff",
		Foreground:   "#f5e8ff",
		Muted:        "#cc88ff",
		Subtle:       "#7744bb",
		Primary:      "#ff00ff",
		PrimaryGlow:  "#cc00cc",
		Accent:       "#00ffcc",
		AccentGlow:   "#00ccaa",
		GradientFrom: "#ff00ee",
		GradientTo:   "#00ccff",
		Success:      "#00ff77",
		Warning:      "#ffaa00",
		Danger:       "#ff1133",
		Info:         "#00eeff",
	}
}

// AllPalettes returns the built-in themes in display order.
func AllPalettes() []Theme {
	return []Theme{PastelDark(), PastelLight(), Nova()}
}

// ByName returns a built-in theme by config name. Unknown or empty names fall
// back to the default so a typo in config does not make the app unusable.
func ByName(name string) Theme {
	name = strings.TrimSpace(strings.ToLower(name))
	for _, t := range AllPalettes() {
		if t.Name == name {
			return t
		}
	}
	return PastelDark()
}
