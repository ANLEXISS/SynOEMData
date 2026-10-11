# SynoraOEM Repository Instructions

## Milestone completion and report publishing

For each milestone task in this repository:

1. Read [`SYNORA_DOCUMENTATION_WORKFLOW.md`](SYNORA_DOCUMENTATION_WORKFLOW.md) before changing or publishing milestone documentation.
2. Complete the implementation and run the validation required by that milestone. Record actual results and limitations in the appropriate report under `docs/`.
3. Do not describe incomplete or failed validation as a successful milestone. Record a failed, pending, or hardware-unverified status accurately.
4. Before publishing, review the Markdown security exclusions and run the secret scan through the synchronization command.
5. After validation and report updates, run `tools/sync-documentation.sh --validated`. The command copies only approved Markdown, commits changes, and pushes to `ANLEXISS/SynOEMData`.
6. If synchronization fails, preserve the local worktree and commit, report `PENDING`, and retry at the next authorized execution. Never claim publication without a confirmed push and commit hash.

Do not add scheduled tasks, background services, global Git settings, credentials, or production changes for documentation publishing.
