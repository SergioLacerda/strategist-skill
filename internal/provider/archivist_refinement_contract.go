package provider

// ArchivistRefinementExecutionContract adapts a prompt-only planning skill to
// the Archivist's single-shot, read-only Weapon boundary (ADR-0067). The host
// executes the compiled payload; Strategist keeps every write, the user
// interaction and the Approval Gate.
const ArchivistRefinementExecutionContract = "Execute the compiled Weapon payload once as a single-shot, read-only Archivist refinement call through the current host. " +
	"Read only the Ranger analysis at input.analysis_artifact_path and plan only what it supports; mark a gap [NEEDS CLARIFICATION: <question>] or [INSUFFICIENT EVIDENCE: <what is missing>] instead of inventing it. " +
	"Do not ask the user questions, wait for an approval, invoke another skill, write files, save a plan under docs/plans/ or any other repository directory, " +
	"run or schedule git commands, add commit steps, or perform implementation; drop the payload's execution-mode question and its subagent hand-off. " +
	"Return only the refinement package required by output_contract."

// ArchivistRefinementOutputContract is the shape the embedded publisher parses
// and validates before anything is written. A completion carries the typed
// handoff facts and exactly three documents; Strategist adds analysis.md from
// the pending Ranger artifact and owns provenance and mission status.
const ArchivistRefinementOutputContract = "Return Markdown that starts with a YAML frontmatter containing only a `handoff_policy_facts` mapping with " +
	"schema_version `strategist-handoff-policy-facts/v1`, the lists mandatory_constraints, unresolved_questions and forbidden_scope " +
	"(use an empty list, never omit one) and the booleans destructive_operation_possible, security_sensitive_task and informational_only. " +
	"The body must contain exactly these three second-level sections, once each and nothing outside them: `## proposal.md`, `## design.md` and `## tasks.md`; " +
	"headings inside a document start at level three. Every checkbox line in tasks.md must be `- [ ] N.M [task_type: <type>] <task and how to verify it>` " +
	"with <type> one of documentation_target, analysis_artifact, implementation_handoff or out_of_scope, and a documentation_target task must name a backtick-quoted repository-relative path. " +
	"No section may direct the plan to a repository planning directory such as docs/plans/, and no task may run a git mutation command."
