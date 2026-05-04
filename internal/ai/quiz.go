package ai

import (
	"regexp"
	"strings"

	"github.com/viphase/sparkle/internal/domain"
)

// quizBlockRE matches fenced quiz blocks:
//
//	<quiz>
//	Your question?
//	a) First option
//	b) Second option
//	</quiz>
var quizBlockRE = regexp.MustCompile(`(?s)<quiz>\n?(.*?)\n?</quiz>`)

// choiceLineRE matches a single choice line like "a) Some text" or "b) Other".
var choiceLineRE = regexp.MustCompile(`^([a-z])\)\s+(.+)$`)

// parseQuizBlocks strips <quiz> blocks from text and returns them as domain.Quiz
// values. The returned string is the text with all quiz blocks removed.
func parseQuizBlocks(text string) (string, []domain.Quiz) {
	var quizzes []domain.Quiz
	clean := quizBlockRE.ReplaceAllStringFunc(text, func(match string) string {
		sub := quizBlockRE.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		body := strings.TrimSpace(sub[1])
		quiz := parseQuizBody(body)
		if quiz.Question != "" && len(quiz.Choices) >= 2 {
			quizzes = append(quizzes, quiz)
		}
		return "" // strip from text
	})
	return strings.TrimSpace(clean), quizzes
}

// parseQuizBody turns the raw content between <quiz> tags into a Quiz.
// Lines before the first choice line are the question; lines starting with
// a letter-paren are choices.
func parseQuizBody(body string) domain.Quiz {
	lines := strings.Split(body, "\n")
	var quiz domain.Quiz
	var questionLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		m := choiceLineRE.FindStringSubmatch(trimmed)
		if m != nil {
			quiz.Choices = append(quiz.Choices, domain.QuizChoice{
				Key:  m[1],
				Text: strings.TrimSpace(m[2]),
			})
		} else if len(quiz.Choices) == 0 {
			// Lines before the first choice form the question.
			questionLines = append(questionLines, trimmed)
		}
	}
	quiz.Question = strings.Join(questionLines, " ")
	return quiz
}
