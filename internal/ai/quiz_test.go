package ai

import (
	"strings"
	"testing"
)

func TestParseQuizBlocksExtractsQuiz(t *testing.T) {
	input := `Let me understand your project better.

<quiz>
Who is the primary user?
a) A solo developer
b) A small team (2–5 people)
c) A non-technical creator
d) Someone else — describe
</quiz>

Let me know when you're ready.`

	text, quizzes := parseQuizBlocks(input)

	if len(quizzes) != 1 {
		t.Fatalf("expected 1 quiz, got %d", len(quizzes))
	}
	q := quizzes[0]
	if !strings.Contains(q.Question, "primary user") {
		t.Errorf("question missing expected text: %q", q.Question)
	}
	if len(q.Choices) != 4 {
		t.Fatalf("expected 4 choices, got %d", len(q.Choices))
	}
	if q.Choices[0].Key != "a" {
		t.Errorf("first choice key: want %q, got %q", "a", q.Choices[0].Key)
	}
	if q.Choices[1].Key != "b" {
		t.Errorf("second choice key: want %q, got %q", "b", q.Choices[1].Key)
	}
	if !strings.Contains(q.Choices[2].Text, "non-technical") {
		t.Errorf("third choice text unexpected: %q", q.Choices[2].Text)
	}
	// Quiz block stripped from text.
	if strings.Contains(text, "<quiz") {
		t.Errorf("quiz block not stripped from text: %q", text)
	}
	if !strings.Contains(text, "Let me understand") {
		t.Errorf("surrounding text missing: %q", text)
	}
}

func TestParseQuizBlocksStripsAndPreservesText(t *testing.T) {
	input := "Before.\n\n<quiz>\nQ?\na) A\nb) B\n</quiz>\n\nAfter."
	text, quizzes := parseQuizBlocks(input)
	if len(quizzes) != 1 {
		t.Fatalf("expected 1 quiz, got %d", len(quizzes))
	}
	if strings.Contains(text, "<quiz") {
		t.Errorf("quiz block not stripped: %q", text)
	}
	if !strings.Contains(text, "Before.") || !strings.Contains(text, "After.") {
		t.Errorf("surrounding text damaged: %q", text)
	}
}

func TestParseQuizBlocksIgnoresIncomplete(t *testing.T) {
	// A quiz with only one choice should not be returned (need at least 2).
	input := "<quiz>\nOnly one choice?\na) Just this one\n</quiz>"
	_, quizzes := parseQuizBlocks(input)
	if len(quizzes) != 0 {
		t.Errorf("expected 0 quizzes for single-choice block, got %d", len(quizzes))
	}
}

func TestParseQuizBlocksMultipleQuizzes(t *testing.T) {
	input := `<quiz>
Q1?
a) A1
b) B1
</quiz>

Some text.

<quiz>
Q2?
a) A2
b) B2
c) C2
</quiz>`
	text, quizzes := parseQuizBlocks(input)
	if len(quizzes) != 2 {
		t.Fatalf("expected 2 quizzes, got %d", len(quizzes))
	}
	if quizzes[0].Question != "Q1?" {
		t.Errorf("first question: %q", quizzes[0].Question)
	}
	if len(quizzes[1].Choices) != 3 {
		t.Errorf("second quiz: expected 3 choices, got %d", len(quizzes[1].Choices))
	}
	if strings.Contains(text, "<quiz") {
		t.Errorf("quiz blocks not stripped: %q", text)
	}
}

func TestParseQuizBodyParsesCorrectly(t *testing.T) {
	body := "What is the risk?\na) Overbuilding\nb) Wrong audience\nc) Under-delivering"
	quiz := parseQuizBody(body)
	if quiz.Question != "What is the risk?" {
		t.Errorf("question: %q", quiz.Question)
	}
	if len(quiz.Choices) != 3 {
		t.Fatalf("expected 3 choices, got %d", len(quiz.Choices))
	}
	if quiz.Choices[2].Text != "Under-delivering" {
		t.Errorf("third choice text: %q", quiz.Choices[2].Text)
	}
}
