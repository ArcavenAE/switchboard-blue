# Demo Evidence Policy

## Rule

**Do not commit rendered demo binaries (`.gif`, `.webm`, `.mp4`, `.png`,
`.jpg`, `.jpeg`) to this repo.** `.tape` scripts, `evidence-report.md`
files, and `manifest.yaml` files under `docs/demo-evidence/<STORY-ID>/`
are welcome and encouraged — they are plain text, diffable, and small.

Rendered binaries are gitignored under `docs/demo-evidence/**` (see
`.gitignore`).

## Why

- Rendered VHS artifacts bloat the repo (~200-300 KB per AC × dozens of
  ACs; ~13 MB before this policy was ratified).
- Binaries are not useful in `git diff` — reviewers cannot tell what
  changed between two GIFs; a modified `.tape` script tells them exactly.
- The value of a demo is the *script*, not the render. Anyone can
  `vhs docs/demo-evidence/<STORY-ID>/AC-NNN.tape` locally to regenerate
  the animation on demand.
- Rendered artifacts belong on the *deliverable*, not in the *source*.
  If we ever want animated demos on the release page or the docs site,
  CI can render them from `.tape` sources at publish time.

## What agents should do

The `vsdd-factory:demo-recorder` agent's default behavior renders both
`.gif` and `.webm` for every AC. Under this project's policy, the
recorder should:

1. **Produce and commit** the `.tape` script for each AC (source of truth).
2. **Produce and commit** the `evidence-report.md` mapping ACs to tapes.
3. **Skip rendering** (`.gif` / `.webm`) — or if rendering is helpful for
   local verification, leave the rendered files in the worktree; git will
   ignore them.

Orchestrators dispatching `demo-recorder` should include this instruction
verbatim in the task prompt:

> **Repo policy: do not render or commit `.gif`/`.webm`/`.mp4`/`.png`.
> Commit `.tape` scripts and `evidence-report.md` only. If you need to
> verify a tape renders, do so locally; the rendered artifacts are
> gitignored under `docs/demo-evidence/**`.**

## Regenerating a demo locally

```bash
# Inside a story worktree or the main checkout:
cd docs/demo-evidence/<STORY-ID>
vhs AC-NNN-description.tape
# → produces AC-NNN-description.gif and AC-NNN-description.webm,
#   both ignored by git.
```

## History

- **2026-07-04** — Policy ratified after 62 tracked binaries
  (~12.3 MB) accumulated across Wave-6 and Phase-3 demo-recorder
  backfill. Prior state: `vsdd-factory:demo-recorder` hardcodes
  VHS→GIF/WebM output; no built-in disable knob. Upstream defect filed
  (drbothen/vsdd-factory) requesting a `demo_artifact_format` project
  configuration surface.

## Related

- `.gitignore` — `docs/demo-evidence/**/*.gif|*.webm|*.mp4|*.png|*.jpg|*.jpeg`
- `vsdd-factory:demo-recorder` — the agent whose default we override
- Upstream defect: (linked from `.vsdd-factory-issues-pending.md`)
