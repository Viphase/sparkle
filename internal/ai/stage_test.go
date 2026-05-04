package ai

import "testing"

func TestParseStageComplete_DetectsTag(t *testing.T) {
	text := "Great, we have enough context now. <stage-complete />"
	clean, ok := parseStageComplete(text)
	if !ok {
		t.Fatal("expected stageComplete=true")
	}
	if clean != "Great, we have enough context now." {
		t.Fatalf("unexpected clean text: %q", clean)
	}
}

func TestParseStageComplete_PairedTag(t *testing.T) {
	text := "Done! <stage-complete></stage-complete> Moving on."
	_, ok := parseStageComplete(text)
	if !ok {
		t.Fatal("expected stageComplete=true for paired tag")
	}
}

func TestParseStageComplete_NoTag(t *testing.T) {
	text := "Tell me more about your target user."
	clean, ok := parseStageComplete(text)
	if ok {
		t.Fatal("expected stageComplete=false when tag absent")
	}
	if clean != text {
		t.Fatalf("text should be unchanged, got %q", clean)
	}
}

func TestParseStageComplete_CaseInsensitive(t *testing.T) {
	text := "All done. <STAGE-COMPLETE />"
	_, ok := parseStageComplete(text)
	if !ok {
		t.Fatal("expected case-insensitive match")
	}
}
