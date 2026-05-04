package tui

// Route identifies a top-level surface. v2 has exactly three:
//   - Workspace  — unified sparks + projects + embedded AI panel
//   - Pulse      — ntcharts dashboard
//   - Settings   — sectioned settings modal
type Route int

const (
	RouteWorkspace Route = iota
	RoutePulse
	RouteSettings
)

var orderedRoutes = []Route{
	RoutePulse,
	RouteWorkspace,
	RouteSettings,
}

func AllRoutes() []Route {
	out := make([]Route, len(orderedRoutes))
	copy(out, orderedRoutes)
	return out
}

func (r Route) Next() Route {
	for i, x := range orderedRoutes {
		if x == r {
			return orderedRoutes[(i+1)%len(orderedRoutes)]
		}
	}
	return RoutePulse
}

func (r Route) Prev() Route {
	for i, x := range orderedRoutes {
		if x == r {
			return orderedRoutes[(i-1+len(orderedRoutes))%len(orderedRoutes)]
		}
	}
	return RoutePulse
}

func (r Route) String() string {
	switch r {
	case RouteWorkspace:
		return "Workspace"
	case RoutePulse:
		return "Pulse"
	case RouteSettings:
		return "Settings"
	}
	return "?"
}
