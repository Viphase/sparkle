package domain

// SkillDef is a filesystem-backed skill loaded from .sparkle/skills/<slug>.md.
// It replaces the hardcoded Skill constants for user-facing code. The Skill
// type alias below is kept for backward-compat with the existing AI/settings wiring.
type SkillDef struct {
	Slug        string // filename slug (no .md)
	Label       string // human-readable label
	Description string // one-line description
	Fragment    string // system prompt fragment
}

// Skill is an injectable prompt fragment that specialises the AI guide for a
// specific project type. Skills are injected between the base system prompt
// and the mode-specific instructions.
type Skill string

const (
	SkillNone       Skill = ""            // generic, no specialisation
	SkillCLITool    Skill = "cli-tool"    // flag design, help text, shell integration
	SkillWebAPI     Skill = "web-api"     // REST/GraphQL shape, auth, rate limiting
	SkillLibrary    Skill = "library"     // API surface, semver, docs, zero-dep
	SkillSoloSaaS   Skill = "solo-saas"  // pricing, retention, onboarding
	SkillOpenSource Skill = "open-source" // contributor experience, governance, licensing
)

// AllSkills returns every skill in display order, starting with the "none" option.
func AllSkills() []Skill {
	return []Skill{SkillNone, SkillCLITool, SkillWebAPI, SkillLibrary, SkillSoloSaaS, SkillOpenSource}
}

// Label returns a short human-readable label suitable for display in the UI.
func (s Skill) Label() string {
	switch s {
	case SkillCLITool:
		return "cli-tool"
	case SkillWebAPI:
		return "web-api"
	case SkillLibrary:
		return "library"
	case SkillSoloSaaS:
		return "solo-saas"
	case SkillOpenSource:
		return "open-source"
	}
	return "none"
}

// Description returns a one-line summary of what the skill specialises on.
func (s Skill) Description() string {
	switch s {
	case SkillCLITool:
		return "flag design, help text, shell integration, exit codes"
	case SkillWebAPI:
		return "REST/GraphQL shape, auth, rate limiting, error contracts"
	case SkillLibrary:
		return "API surface, semver discipline, docs, zero-dependency"
	case SkillSoloSaaS:
		return "pricing, retention, onboarding funnel, churn analysis"
	case SkillOpenSource:
		return "contributor experience, governance, licensing, issue triage"
	}
	return "generic project guidance, no specialisation"
}

// SystemFragment returns the prompt text injected between the base system prompt
// and the mode-specific instructions when this skill is active. Returns "" for
// SkillNone so the inject step is a no-op.
func (s Skill) SystemFragment() string {
	switch s {
	case SkillCLITool:
		return `Project type: CLI TOOL.
Additional focus:
- Command/sub-command naming: noun-verb convention, short aliases where unambiguous.
- Flag design: --double-dash long names, -s short aliases, consistent value types.
- Help text: every flag needs a one-line description; --help must work at every level.
- Exit code contract: 0=success, 1=user error, 2=internal error, 3+=domain-specific.
- Shell completions for bash/zsh/fish — ask whether they are planned.
- Cross-platform: Windows path separators, CRLF, color support via NO_COLOR.
- Distribution: single static binary, brew/apt/winget/scoop formula, --version flag.`

	case SkillWebAPI:
		return `Project type: WEB API.
Additional focus:
- REST resource naming (plural nouns, no verbs) or GraphQL schema design.
- Auth strategy: JWT, API keys, OAuth2 — pick one early and defend it.
- Rate limiting: per-IP, per-user, per-endpoint — define the threat model.
- Error contracts: consistent error envelope with machine-readable codes.
- Pagination: cursor-based (preferred for large sets) vs offset — commit early.
- API versioning: URL prefix (/v1/) vs Accept header — pick one.
- Idempotency keys for mutation endpoints that must be safe to retry.
- OpenAPI/AsyncAPI spec generated from code, not written by hand.`

	case SkillLibrary:
		return `Project type: LIBRARY / PACKAGE.
Additional focus:
- API surface: smallest possible exported API — unexport everything you can.
- Semver discipline: no breaking changes in minor releases; deprecation notices before removal.
- Zero external dependency policy — justify every dep with size + maintenance risk.
- Documentation: godoc comment on every exported symbol before v1.0.
- Compatibility floor: state minimum language/runtime version and test against it.
- Error handling: sentinel errors vs wrapped errors — pick one convention per package.
- Context propagation: every IO-bound function must accept and respect context.Context.
- Changelog: keep CHANGELOG.md from the first tagged release.`

	case SkillSoloSaaS:
		return `Project type: SOLO SAAS.
Additional focus:
- Pricing: freemium vs free-trial vs paid-only — what is the conversion funnel?
- Onboarding: define time-to-first-value (TTFV) and the "magic moment" explicitly.
- Retention hooks: habits, email sequences, integrations — what brings users back?
- Churn signals: what behaviour precedes cancellation? Build alerts for it.
- Support cost: self-serve docs, status page, one-inbox triage — scope to one person.
- Payments: Stripe Billing, dunning emails, VAT/GST compliance from day one.
- Key metrics: MRR, churn rate, LTV/CAC ratio — instrument from the first paying user.`

	case SkillOpenSource:
		return `Project type: OPEN SOURCE.
Additional focus:
- Contributor experience: CONTRIBUTING.md, issue templates, PR checklist.
- Governance: BDFL, steering committee, or lazy consensus — define it before disputes.
- License: MIT/Apache-2.0/GPL-3.0 — know the viral clauses before you commit.
- Issue triage: labels, stale-bot policy, public response-time SLA.
- Release process: release branches, CHANGELOG, GitHub Releases, signed tags.
- Community: CODE_OF_CONDUCT.md, discussion forum, office-hours cadence.
- Sustainability: GitHub Sponsors, Open Collective, dual-licensing options.`
	}
	return ""
}
