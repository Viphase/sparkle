package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/workspace"
)

const skillsSubDir = "skills"
const promptsSubDir = "prompts"

// SkillsDir returns the path to the skills directory inside the workspace meta dir.
func SkillsDir(root string) string {
	return filepath.Join(root, workspace.MetaDirName, skillsSubDir)
}

// LoadSkills reads all *.md files from .sparkle/skills/ and returns them as
// domain.Skill values. The slug is derived from the filename (sans extension).
// If the directory does not exist an empty slice is returned without error.
func LoadSkills(root string) ([]domain.SkillDef, error) {
	dir := SkillsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	var skills []domain.SkillDef
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read skill %s: %w", e.Name(), err)
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		skills = append(skills, parseSkillFile(slug, string(raw)))
	}
	return skills, nil
}

// SeedBuiltinSkills writes the 5 built-in skill files to .sparkle/skills/ if
// they do not already exist. Existing files are never overwritten.
func SeedBuiltinSkills(root string) error {
	dir := SkillsDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir skills: %w", err)
	}
	for _, def := range builtinSkillDefs() {
		path := filepath.Join(dir, def.Slug+".md")
		if _, err := os.Stat(path); err == nil {
			continue // already exists
		}
		content := renderSkillFile(def)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("seed skill %s: %w", def.Slug, err)
		}
	}
	return nil
}

// LoadSystemPrompt reads .sparkle/prompts/system.md. Returns the built-in
// default if the file does not exist.
func LoadSystemPrompt(root string) (string, error) {
	path := filepath.Join(root, workspace.MetaDirName, promptsSubDir, "system.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultSystemPrompt(), nil
		}
		return "", fmt.Errorf("read system prompt: %w", err)
	}
	if s := strings.TrimSpace(string(raw)); s != "" {
		return s, nil
	}
	return defaultSystemPrompt(), nil
}

// SeedSystemPrompt writes the default system prompt if the file is absent.
func SeedSystemPrompt(root string) error {
	dir := filepath.Join(root, workspace.MetaDirName, promptsSubDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir prompts: %w", err)
	}
	path := filepath.Join(dir, "system.md")
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}
	return os.WriteFile(path, []byte(defaultSystemPrompt()), 0o644)
}

// parseSkillFile extracts label, description, and system fragment from a skill
// Markdown file. The format is:
//
//	# Label
//	> One-line description
//
//	Full system prompt content follows...
func parseSkillFile(slug, content string) domain.SkillDef {
	def := domain.SkillDef{Slug: slug, Label: slug}
	lines := strings.Split(content, "\n")
	bodyStart := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if def.Label == slug && strings.HasPrefix(trimmed, "# ") {
			def.Label = strings.TrimPrefix(trimmed, "# ")
			bodyStart = i + 1
			continue
		}
		if def.Description == "" && strings.HasPrefix(trimmed, "> ") {
			def.Description = strings.TrimPrefix(trimmed, "> ")
			bodyStart = i + 1
			continue
		}
		if def.Label != slug && def.Description != "" {
			bodyStart = i
			break
		}
	}
	// Remaining content is the system fragment.
	if bodyStart < len(lines) {
		def.Fragment = strings.TrimSpace(strings.Join(lines[bodyStart:], "\n"))
	}
	return def
}

func renderSkillFile(def domain.SkillDef) string {
	var b strings.Builder
	b.WriteString("# " + def.Label + "\n")
	if def.Description != "" {
		b.WriteString("> " + def.Description + "\n")
	}
	b.WriteString("\n")
	b.WriteString(def.Fragment + "\n")
	return b.String()
}

func defaultSystemPrompt() string {
	return `You are Sparkle's local project guide.
Help turn rough ideas into practical, structured project work.
Be concise. Ask one clarifying question at a time when context is thin.
Never invent facts. Never write files without explicit permission.`
}

func builtinSkillDefs() []domain.SkillDef {
	return []domain.SkillDef{
		{
			Slug:        "cli-tool",
			Label:       "cli-tool",
			Description: "flag design, help text, shell integration, exit codes",
			Fragment: `Project type: CLI TOOL.
Additional focus:
- Command/sub-command naming: noun-verb convention, short aliases where unambiguous.
- Flag design: --double-dash long names, -s short aliases, consistent value types.
- Help text: every flag needs a one-line description; --help must work at every level.
- Exit code contract: 0=success, 1=user error, 2=internal error, 3+=domain-specific.
- Shell completions for bash/zsh/fish — ask whether they are planned.
- Cross-platform: Windows path separators, CRLF, color support via NO_COLOR.
- Distribution: single static binary, brew/apt/winget/scoop formula, --version flag.`,
		},
		{
			Slug:        "web-api",
			Label:       "web-api",
			Description: "REST/GraphQL shape, auth, rate limiting, error contracts",
			Fragment: `Project type: WEB API.
Additional focus:
- REST resource naming (plural nouns, no verbs) or GraphQL schema design.
- Auth strategy: JWT, API keys, OAuth2 — pick one early and defend it.
- Rate limiting: per-IP, per-user, per-endpoint — define the threat model.
- Error contracts: consistent error envelope with machine-readable codes.
- Pagination: cursor-based (preferred for large sets) vs offset — commit early.
- API versioning: URL prefix (/v1/) vs Accept header — pick one.
- Idempotency keys for mutation endpoints that must be safe to retry.
- OpenAPI/AsyncAPI spec generated from code, not written by hand.`,
		},
		{
			Slug:        "library",
			Label:       "library",
			Description: "API surface, semver discipline, docs, zero-dependency",
			Fragment: `Project type: LIBRARY / PACKAGE.
Additional focus:
- API surface: smallest possible exported API — unexport everything you can.
- Semver discipline: no breaking changes in minor releases; deprecation notices before removal.
- Zero external dependency policy — justify every dep with size + maintenance risk.
- Documentation: godoc comment on every exported symbol before v1.0.
- Compatibility floor: state minimum language/runtime version and test against it.
- Error handling: sentinel errors vs wrapped errors — pick one convention per package.
- Context propagation: every IO-bound function must accept and respect context.Context.
- Changelog: keep CHANGELOG.md from the first tagged release.`,
		},
		{
			Slug:        "solo-saas",
			Label:       "solo-saas",
			Description: "pricing, retention, onboarding funnel, churn analysis",
			Fragment: `Project type: SOLO SAAS.
Additional focus:
- Pricing: freemium vs free-trial vs paid-only — what is the conversion funnel?
- Onboarding: define time-to-first-value (TTFV) and the "magic moment" explicitly.
- Retention hooks: habits, email sequences, integrations — what brings users back?
- Churn signals: what behaviour precedes cancellation? Build alerts for it.
- Support cost: self-serve docs, status page, one-inbox triage — scope to one person.
- Payments: Stripe Billing, dunning emails, VAT/GST compliance from day one.
- Key metrics: MRR, churn rate, LTV/CAC ratio — instrument from the first paying user.`,
		},
		{
			Slug:        "open-source",
			Label:       "open-source",
			Description: "contributor experience, governance, licensing, issue triage",
			Fragment: `Project type: OPEN SOURCE.
Additional focus:
- Contributor experience: CONTRIBUTING.md, issue templates, PR checklist.
- Governance: BDFL, steering committee, or lazy consensus — define it before disputes.
- License: MIT/Apache-2.0/GPL-3.0 — know the viral clauses before you commit.
- Issue triage: labels, stale-bot policy, public response-time SLA.
- Release process: release branches, CHANGELOG, GitHub Releases, signed tags.
- Community: CODE_OF_CONDUCT.md, discussion forum, office-hours cadence.
- Sustainability: GitHub Sponsors, Open Collective, dual-licensing options.`,
		},
	}
}
