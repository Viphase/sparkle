package domain

import (
	"strings"
	"testing"
)

func TestAllSkillsIncludesNone(t *testing.T) {
	skills := AllSkills()
	if skills[0] != SkillNone {
		t.Fatalf("first skill should be SkillNone, got %q", skills[0])
	}
	if len(skills) < 6 {
		t.Fatalf("expected at least 6 skills, got %d", len(skills))
	}
}

func TestSkillLabels(t *testing.T) {
	cases := []struct {
		skill Skill
		want  string
	}{
		{SkillNone, "none"},
		{SkillCLITool, "cli-tool"},
		{SkillWebAPI, "web-api"},
		{SkillLibrary, "library"},
		{SkillSoloSaaS, "solo-saas"},
		{SkillOpenSource, "open-source"},
	}
	for _, c := range cases {
		if got := c.skill.Label(); got != c.want {
			t.Errorf("skill %q label = %q, want %q", c.skill, got, c.want)
		}
	}
}

func TestSkillDescriptionsNonEmpty(t *testing.T) {
	for _, s := range AllSkills() {
		if s == SkillNone {
			continue
		}
		if strings.TrimSpace(s.Description()) == "" {
			t.Errorf("skill %q has empty description", s)
		}
	}
}

func TestSkillSystemFragmentNoneIsEmpty(t *testing.T) {
	if SkillNone.SystemFragment() != "" {
		t.Fatalf("SkillNone.SystemFragment() should be empty")
	}
}

func TestSkillSystemFragmentContainsFocusAreas(t *testing.T) {
	for _, s := range AllSkills() {
		if s == SkillNone {
			continue
		}
		frag := s.SystemFragment()
		if !strings.Contains(frag, "Additional focus") {
			t.Errorf("skill %q fragment missing 'Additional focus' heading: %q", s, frag[:min(60, len(frag))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
