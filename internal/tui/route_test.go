package tui

import "testing"

func TestRouteNextWraps(t *testing.T) {
	if got := RouteDashboard.Next(); got != RouteProjects {
		t.Errorf("Dashboard.Next() = %v, want Projects", got)
	}
	last := orderedRoutes[len(orderedRoutes)-1]
	if got := last.Next(); got != orderedRoutes[0] {
		t.Errorf("last route should wrap to first; got %v want %v", got, orderedRoutes[0])
	}
}

func TestRoutePrevWraps(t *testing.T) {
	if got := RouteSparks.Prev(); got != RouteAI {
		t.Errorf("Sparks.Prev() = %v, want AI", got)
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
		RouteAI:        "AI",
		RouteSettings:  "Settings",
	}
	for r, want := range cases {
		if got := r.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", r, got, want)
		}
	}
}

func TestRouteOrderKeepsDashboardCenteredInPrimaryNav(t *testing.T) {
	want := []Route{RouteAI, RouteSparks, RouteDashboard, RouteProjects, RouteSettings}
	if len(orderedRoutes) != len(want) {
		t.Fatalf("len(orderedRoutes)=%d, want %d", len(orderedRoutes), len(want))
	}
	for i := range want {
		if orderedRoutes[i] != want[i] {
			t.Fatalf("orderedRoutes[%d]=%v, want %v", i, orderedRoutes[i], want[i])
		}
	}
}

func TestAllRoutesIsCopy(t *testing.T) {
	got := AllRoutes()
	got[0] = RouteSettings
	if orderedRoutes[0] != RouteAI {
		t.Error("AllRoutes() must return a copy; mutation leaked back to package state")
	}
}

func TestNumberRoute(t *testing.T) {
	cases := []struct {
		in   string
		want Route
		ok   bool
	}{
		{"1", RouteAI, true},
		{"2", RouteSparks, true},
		{"3", RouteDashboard, true},
		{"4", RouteProjects, true},
		{"5", RouteSettings, true},
		{"6", 0, false},
		{"0", 0, false},
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
