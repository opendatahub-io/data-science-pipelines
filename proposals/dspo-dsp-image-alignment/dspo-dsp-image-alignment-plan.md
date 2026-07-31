# DSPO/DSP Image Alignment Plan

## Current Status

### Current Image Production and Consumption

| Org  | Branch    | Actually produced                                     | Built by      | DSP CI expects                       | Nightly builds expect                | Status                 |
|------|-----------|-------------------------------------------------------|---------------|--------------------------------------|--------------------------------------|------------------------|
| ODH  | `main`    | `quay.io/opendatahub/...:main`                        | GHA + Konflux | `quay.io/opendatahub/...:main`       | *not used*                           | ✅ Currently aligned    |
| ODH  | `stable`  | `quay.io/opendatahub/...:odh-stable`                  | Konflux       | `quay.io/opendatahub/...:odh-stable` | `quay.io/opendatahub/...:odh-stable` | ✅ Fully aligned        |
| ODH  | PRs       | `quay.io/opendatahub/...:pr-<number>`                 | GHA           | *not consumed*                       | *not used*                           | ❌ Not consumed         |
| RHDS | `main`    | `quay.io/opendatahub/...:stable`                      | Konflux       | `quay.io/rhoai/...:main`             | *uses versioned releases*            | ❌ Wrong registry + tag |
| RHDS | `rhoai-*` | *nothing*                                             | —             | `quay.io/rhoai/...:rhoai-*`          | *uses versioned releases*            | ❌ Missing images       |
| RHDS | PRs       | `quay.io/rhoai/pull-request-pipelines:...-<revision>` | Konflux       | *not consumed*                       | *not used*                           | ❌ Not consumed         |

### Nightly Build Expectations

**ODH Nightly Builds:** 
- Consume ODH Konflux images with `odh-` prefixed tags: `quay.io/opendatahub/data-science-pipelines-operator:odh-stable`
- Source: ODH-Build-Config workflows, Konflux integration tests, distributed workloads
- The `odh-` prefix is intentional to distinguish ODH builds from other builds

**RHDS Nightly Builds:**
- Produce versioned releases using a different image name pattern
- Example: `quay.io/rhoai/odh-data-science-pipelines-operator-controller-rhel8:rhoai-2.16`
- Source: RHOAI-Build-Config workflows (`trigger-nightly-bundle-build.yaml`, `trigger-nightly-fbc-build.yaml`)
- Uses long-form image names with RHEL version suffix

## Expected Status

### Expected Image Production and Consumption

| Org  | Branch    | Produced image                                                        | Produced by   | DSP CI consumes                                                  | Nightly builds consume    |
|------|-----------|-----------------------------------------------------------------------|---------------|------------------------------------------------------------------|---------------------------|
| ODH  | `main`    | `quay.io/opendatahub/data-science-pipelines-operator:odh-main`        | GHA + Konflux | `quay.io/opendatahub/data-science-pipelines-operator:odh-main`   | *not used*                |
| ODH  | `stable`  | `quay.io/opendatahub/data-science-pipelines-operator:odh-stable`      | Konflux       | `quay.io/opendatahub/data-science-pipelines-operator:odh-stable` | `...:odh-stable`          |
| ODH  | PRs       | `quay.io/opendatahub/data-science-pipelines-operator:odh-pr-<number>` | GHA           | `quay.io/opendatahub/data-science-pipelines-operator:odh-main`   | *not used*                |
| RHDS | `main`    | `quay.io/rhoai/data-science-pipelines-operator:main`                  | Konflux       | `quay.io/rhoai/data-science-pipelines-operator:main`             | *uses versioned releases* |
| RHDS | `rhoai-*` | `quay.io/rhoai/data-science-pipelines-operator:rhoai-*`               | Konflux       | `quay.io/rhoai/data-science-pipelines-operator:rhoai-*`          | *uses versioned releases* |
| RHDS | PRs       | unchanged (`quay.io/rhoai/pull-request-pipelines:...-{{revision}}`)   | Konflux       | `quay.io/rhoai/data-science-pipelines-operator:main`             | *not used*                |

DSP CI resolves the DSPO image from the PR's **target branch** (`github_base_ref`), not a PR-specific tag. No workflow passes a PR-specific `operator_image_tag`.

## Goals of This Alignment

1. **Registry separation** — ODH uses `quay.io/opendatahub/`, RHDS uses `quay.io/rhoai/` (enforced by `konflux-central` CI)
2. **Tag-based distinction** — `odh-` prefix for ODH tags; plain tags for RHDS
3. **Complete coverage** — main/master, release branches, and PRs have corresponding images consumed by DSP CI
4. **Preserved nightly builds** — maintain `odh-*` prefix compatibility  
5. **konflux-central compliance** — all Tekton changes go through `red-hat-data-services/konflux-central`

## How DSP CI Resolves the DSPO Image

`operator_deployer.py` determines the DSPO image at deploy time.

**Current code** (after PR 2 — hardcodes `quay.io/opendatahub/` for all orgs):

```python
operator_image_tag = getattr(self.args, 'operator_image_tag', '') or ''
if operator_image_tag:
    dspo_tag = operator_image_tag
elif self.target_branch == 'stable':
    dspo_tag = 'odh-stable'
elif self.target_branch == 'master':
    dspo_tag = 'odh-main' if self.repo_owner == 'opendatahub-io' else 'main'
else:
    dspo_tag = self.target_branch
operator_image = f'quay.io/opendatahub/data-science-pipelines-operator:{dspo_tag}'
```

**Target code** (PR 5 — selects registry by org):

```python
repo = 'opendatahub' if self.repo_owner == 'opendatahub-io' else 'rhoai'
operator_image = f'quay.io/{repo}/data-science-pipelines-operator:{dspo_tag}'
```

The tag logic stays the same. Only the registry changes: `opendatahub-io` → `quay.io/opendatahub/`, `red-hat-data-services` → `quay.io/rhoai/`.

## Implementation Plan

### PR Sequence

**PR 1 - DSPO (`opendatahub-io/data-science-pipelines-operator` main): ✅ MERGED**
- Change ODH main push builds: `:main` → `:odh-main` (both GHA and Konflux)
- Change ODH PR builds: `:pr-<number>` → `:odh-pr-<number>` (GHA `build-prs.yml`)

**PR 2 - DSP (`opendatahub-io/data-science-pipelines` master): ✅ MERGED (needs follow-up)**
- Update `operator_deployer.py` for new tag patterns:
  - ODH main: `:main` → `:odh-main`
  - ODH PRs: `:pr-<number>` → `:odh-pr-<number>`
  - ODH stable: `:odh-stable` (no change)
  - RHDS branches: `:main`, `:rhoai-*`
- ⚠️ **Issue:** Hardcoded `quay.io/opendatahub/` for all orgs. RHDS needs `quay.io/rhoai/`. Fixed in PR 5.

**PR 3 - DSPO (`opendatahub-io/data-science-pipelines-operator` merge main → stable): ✅ MERGED**
- Merged main → stable (with conflict resolution for Konflux configs)
- Stable branch continues producing `:odh-stable`

**PR 4 - DSP (`opendatahub-io/data-science-pipelines` merge master → stable): ✅ MERGED**
- Merged master → stable (brought PR 2 changes to stable, including the registry issue)

**ODH stable → RHDS sync** (triggered by PR 3 and PR 4)

**PR 5 - DSP (`opendatahub-io/data-science-pipelines` master, fix PR 2):**
- Fix `operator_deployer.py`: use `quay.io/rhoai/` when `repo_owner != 'opendatahub-io'`
- Fix `test_operator_deployer.py`: update RHDS test expectations from `quay.io/opendatahub/` to `quay.io/rhoai/`

**PR 6 - DSP (`opendatahub-io/data-science-pipelines` merge master → stable):**
- Merge master → stable to bring PR 5's registry fix to stable
- Resolve any conflicts from the merge

**ODH stable → RHDS sync** (triggered by PR 6)

**PR 7 - DSP (`red-hat-data-services/data-science-pipelines`, after sync):**
- Verify `operator_deployer.py` registry fix synced correctly from ODH stable
- If sync creates conflicts, resolve them so RHDS DSP CI uses `quay.io/rhoai/data-science-pipelines-operator:{dspo_tag}`

**PR 8 - konflux-central (`red-hat-data-services/konflux-central`):**
- Add push config with `quay.io/rhoai/data-science-pipelines-operator:{{target_branch}}`, CEL matches `main` + `rhoai-*`
- PR config: no change needed (already uses `quay.io/rhoai/pull-request-pipelines:`)
- `validate-pipelineruns.yml`: pass `--branch main` for PRs targeting main
- Push config will be synced automatically to the DSPO component repo by konflux-central automation

**PR 9 - DSPO (`red-hat-data-services/data-science-pipelines-operator` main):**
- Disable GHA build workflows (`build-main.yml`, `build-prs.yml`, `build-prs-trigger.yaml`) — RHDS uses Konflux exclusively for builds; these arrive from ODH sync but are not needed
- Image refs: `IMAGES_DSPO` in params.env, Makefile, kustomization.yaml → `quay.io/rhoai/data-science-pipelines-operator:main`
- No `.tekton/` changes — those are managed by konflux-central (PR 8)

**PR 10 - DSPO (`red-hat-data-services/data-science-pipelines-operator` merge main → stable):**
- Merge main → stable to bring PR 9's changes to stable
- Resolve any conflicts from the merge

### Critical Timing Requirements

**Simultaneous merges required to avoid breaking DSP CI:**

- **PR 1 & PR 2:** Must merge together - changing ODH main image tags without updating DSP CI breaks ODH main CI ✅ Done
- **PR 3 & PR 4:** Must merge together - bringing new tags to stable without bringing DSP CI changes breaks ODH stable CI ✅ Done

**Note:** PRs 5-10 can merge independently since RHDS DSP CI is already broken today. However, DSP CI registry fix (PR 5) should land before DSPO starts producing images at `quay.io/rhoai/` (PR 8/9), so CI knows where to find them.
