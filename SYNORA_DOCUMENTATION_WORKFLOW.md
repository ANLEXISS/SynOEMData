# SynoraOEM Documentation Publishing Workflow

SynOEMData is the permanent documentation repository for approved SynoraOEM reports:

- Repository: `git@github.com:ANLEXISS/SynOEMData.git`
- Local documentation worktree: `/home/rock/SynOEMData`
- Default branch: `main` (the source repository was empty before initial publication)
- Source tree: `/home/rock/SynoraOEM/docs/`

## After each validated milestone

Complete implementation and required validation first. Update the relevant milestone report with actual results, including failures, pending work, and hardware limitations. Review all new/changed Markdown for credentials, private or personal data, recordings, runtime state, and supplier-confidential information. Then run:

```sh
cd /home/rock/SynoraOEM
tools/sync-documentation.sh --validated
```

The script checks SSH-based repository access, fast-forwards the local documentation clone, scans approved Markdown for common credential formats, copies the source `docs/` hierarchy without deleting destination files, commits only synchronized reports and pushes the selected branch. It does not use or change global Git settings. It refuses to publish if the documentation worktree has unrelated changes, detects unowned destination conflicts, and preserves a local commit with `SYNC=PENDING` if the push fails. Set `SYNOEMDATA_WORKTREE` or `SYNOEMDATA_BRANCH` only when a different local checkout or branch is intentionally required.

The script requires `--validated` as a human/workflow gate; it does not infer that tests passed. The person running it must ensure the reports accurately reflect completed validation.

## Initial security exclusions

These documents remain in SynoraOEM because they contain material that is not approved for public synchronization:

- `docs/security-auth.md` — resident face-profile sample includes a person name and home-zone path.
- `docs/security-network.md` — detailed pairing/firewall design and unresolved security details.

The synchronization command publishes Markdown only. It never copies configuration templates, source code, credentials, keys, datasets, recordings, or runtime state. Re-review exclusions when document content changes; do not remove an exclusion without an explicit security review.

No dependencies, daemons, scheduled tasks, production services, or global Git configuration are part of this workflow.
