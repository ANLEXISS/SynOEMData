# SynoraOEM Repository Instructions

## Milestone completion, review, and publication gates

For each milestone task in this repository:

1. Read [`SYNORA_DOCUMENTATION_WORKFLOW.md`](SYNORA_DOCUMENTATION_WORKFLOW.md) before changing or publishing milestone documentation.
2. Complete the implementation and run the validation required by that milestone. Record actual results and limitations in the appropriate report under `docs/`.
3. Do not describe incomplete or failed validation as a successful milestone. Record a failed, pending, or hardware-unverified status accurately.
4. Validation and documentation do not constitute human approval or authorization to commit, push, or publish. Keep milestone reports local with a review-required status until the owner explicitly approves the milestone.
5. After approval, obtain separate explicit authorization before creating a source commit, before pushing source changes, and before publishing documentation. Do not infer one authorization from another.
6. Only after documentation publication is explicitly authorized, review the Markdown security exclusions and run the secret scan through the synchronization command.
7. Run `tools/sync-documentation.sh --validated` only after explicit publication authorization. The script commits and pushes synchronized Markdown; its `--validated` flag is not approval or publication authorization.
8. If authorized synchronization fails, preserve the local worktree and commit, report `PENDING`, and retry only during a later authorized execution. Never claim publication without a confirmed push and commit hash.

Do not add scheduled tasks, background services, global Git settings, credentials, or production changes for documentation publishing.
