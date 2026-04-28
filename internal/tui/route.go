package tui

type Route int

const (
	RouteDashboard Route = iota
	RouteSparks
	RouteProjects
	RouteAI
	RouteSettings
)

var orderedRoutes = []Route{
	RouteAI,
	RouteSparks,
	RouteDashboard,
	RouteProjects,
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
	return RouteDashboard
}

func (r Route) Prev() Route {
	for i, x := range orderedRoutes {
		if x == r {
			return orderedRoutes[(i-1+len(orderedRoutes))%len(orderedRoutes)]
		}
	}
	return RouteDashboard
}

func (r Route) String() string {
	switch r {
	case RouteDashboard:
		return "Dashboard"
	case RouteSparks:
		return "Sparks"
	case RouteProjects:
		return "Projects"
	case RouteAI:
		return "AI"
	case RouteSettings:
		return "Settings"
	}
	return "?"
}
