package ai

import (
	"regexp"
	"strings"
)

// stageCompleteRE matches the self-closing or paired <stage-complete> tag that
// the AI may emit to signal the current pipeline stage is done.
var stageCompleteRE = regexp.MustCompile(`(?i)<stage-complete\s*/?>(\s*</stage-complete>)?`)

// parseStageComplete strips any <stage-complete> tag from text and reports
// whether the AI signalled stage completion. Whitespace is normalised after
// removal so the visible response is not affected.
func parseStageComplete(text string) (clean string, signalled bool) {
	signalled = stageCompleteRE.MatchString(text)
	if signalled {
		text = stageCompleteRE.ReplaceAllString(text, "")
	}
	clean = strings.TrimSpace(text)
	return clean, signalled
}
