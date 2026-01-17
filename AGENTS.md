<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:

- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:

- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

## AI Interaction Guidelines & Rules

> [!IMPORTANT]
> **CRITICAL INSTRUCTIONS FOR AI AGENTS**
>
> 1. **Language Consistency**: Communicate in the same language as the user. If the user speaks Traditional Chinese, ALL outputs MUST be in Traditional Chinese. This includes but is not limited to:
>    - `Thinking` process and internal analysis.
>    - `TaskName`, `TaskStatus`, and `TaskSummary` in the `task_boundary` tool.
>    - All artifacts (`Task Lists`, `Implementation Plans`, `Walkthroughs`, `Report`).
>    - All tool descriptions and summaries.

## Project Information

For detailed project context, tech stack, architecture, and conventions, see [openspec/project.md](openspec/project.md).
