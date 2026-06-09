## 1 Agent Initialization (Mandatory)
Upon starting ANY session or receiving the first task, the agent **MUST** automatically invoke the following skills before any response or action:
1.  **`using-superpowers`**: To establish the superpower-driven workflow.
1.  **`karpathy-guidelines`**: To ensure surgical, minimal, and verifiable code changes.
1.  **`golang-patterns`**: To ensure idiomatic and high-quality Go code.
1.  **`golang-testing`**: To maintain robust test coverage and follow Go testing best practices.

## 2 Guidelines
1. Use English to write code and comments
2. If lint issues block `make check`, run `make fix` to auto-format and fix common issues.
3. Before any commit, MUST run `make check` and confirm both lint (0 issues) and tests (all pass) before committing.


## reference project
1. litellm: ../litellm
1. new-api: ../new-api
1. gomodel: ../GoModel