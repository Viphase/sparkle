package tui

import "testing"

func TestRouteNextWraps(t *testing.T) {
	if got := RouteDashboard.Next(); got != RouteSparks {
		t.Errorf("Dashboard.Next() = %v, want Sparks", got)
	}
	last := orderedRoutes[len(orderedRoutes)-1]
	if got := last.Next(); got != orderedRoutes[0] {
		t.Errorf("last route should wrap to first; got %v want %v", got, orderedRoutes[0])
	}
}

func TestRoutePrevWraps(t *testing.T) {
	if got := RouteSparks.Prev(); got != RouteDashboard {
		t.Errorf("Sparks.Prev() = %v, want Dashboard", got)
	}
	first := orderedRoutes[0]
	last := orderedRoutes[len(orderedRoutes)-1]
	if got := first.Prev(); got != last {
		t.Errorf("first route should wrap to last; got %v want %v", got, last)
	}
}

func TestRouteString(t *testing.T) {
	cases := map[Route]string{
		RouteDashboard: "Dashboard",
		RouteSparks:    "Sparks",
		RouteProjects:  "Projects",
		RouteTracker:   "Tracker",
		RouteAI:        "AI",
		RouteSettings:  "Settings",
	}
	for r, want := range cases {
		if got := r.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", r, got, want)
		}
	}
}

func TestAllRoutesIsCopy(t *testing.T) {
	got := AllRoutes()
	got[0] = RouteSettings
	if orderedRoutes[0] != RouteDashboard {
		t.Error("AllRoutes() must return a copy; mutation leaked back to package state")
	}
}

func TestNumberRoute(t *testing.T) {
	cases := []struct {
		in    string
		want  Route
		ok    bool
	}{
		{"1", RouteDashboard, true},
		{"2", RouteSparks, true},
		{"6", RouteSettings, true},
		{"0", 0, false},
		{"7", 0, false},
		{"a", 0, false},
		{"", 0, false},
		{"12", 0, false},
	}
	for _, c := range cases {
		got, ok := numberRoute(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("numberRoute(%q) = (%v, %v); want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
