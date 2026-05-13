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

| Org  | Branch    | Expected image                                                        | Produced by         | DSP CI consumes       | Nightly builds consume    |
|------|-----------|-----------------------------------------------------------------------|---------------------|-----------------------|---------------------------|
| ODH  | `main`    | `quay.io/opendatahub/data-science-pipelines-operator:odh-main`        | GHA + Konflux       | `...:odh-main`        | *not used*                |
| ODH  | `stable`  | `quay.io/opendatahub/data-science-pipelines-operator:odh-stable`      | Konflux (no change) | `...:odh-stable`      | `...:odh-stable`          |
| ODH  | PRs       | `quay.io/opendatahub/data-science-pipelines-operator:odh-pr-<number>` | GHA                 | `...:odh-pr-<number>` | *not used*                |
| RHDS | `main`    | `quay.io/opendatahub/data-science-pipelines-operator:main`            | Konflux             | `...:main`            | *uses versioned releases* |
| RHDS | `rhoai-*` | `quay.io/opendatahub/data-science-pipelines-operator:rhoai-*`         | Konflux             | `...:rhoai-*`         | *uses versioned releases* |
| RHDS | PRs       | `quay.io/opendatahub/data-science-pipelines-operator:pr-<number>`     | Konflux             | `...:pr-<number>`     | *not used*                |


## Goals of This Alignment

1. **Unified registry** - consolidate all images in `quay.io/opendatahub/` temporarily. We will not use the `quay.io/rhoai/` registry since RHDS already uses `quay.io/opendatahub/` today. Focus on making DSP CI work correctly first, then address registry separation in a follow-up effort.
2. **Tag-based distinction** - use naming convention to distinguish source org
3. **Complete coverage** - ensure main/master, release branches, and PRs have corresponding images consumed by DSP CI
4. **Preserved nightly builds** - maintain `odh-*` prefix compatibility  
5. **Registry migration foundation** - establish consistent patterns for future `rhoai/` migration

## Implementation Plan

### PR Sequence

**PR 1 - DSPO (ODH main):**
- Change ODH main push builds: `:main` → `:odh-main` (both GHA and Konflux)
- Change ODH PR builds: `:pr-<number>` → `:odh-pr-<number>` (GHA `build-prs.yml`)

**PR 2 - DSP (ODH main):**
- Update `operator_deployer.py` for new tag patterns:
  - ODH main: `:main` → `:odh-main`
  - ODH PRs: `:pr-<number>` → `:odh-pr-<number>`
  - ODH stable: `:odh-stable` (no change)
  - RHDS branches: `:main`, `:rhoai-*` at `quay.io/opendatahub/`
  - RHDS PRs: `:pr-<number>` at `quay.io/opendatahub/`
- **Registry change:** Update RHDS DSP CI to expect images from `quay.io/opendatahub/` instead of `quay.io/rhoai/`

**PR 3 - DSPO (ODH):**
- Merge main → stable (with conflict resolution for Konflux configs)
- Ensure stable branch continues producing `:odh-stable`

**PR 4 - DSP (ODH):**
- Merge main → stable (brings DSP CI changes to stable)

**ODH stable → RHDS sync** (triggered by PR 3 and PR 4)

**PR 5 - DSPO (RHDS, after sync):**
- Change RHDS main push builds: `:odh-stable` → `:main` (Konflux push config arrives from sync with `:odh-stable`; the existing RHDS config producing `:stable` will conflict and must be resolved)
- Change RHDS PR builds: update existing Konflux `pull-request.yaml` to produce `quay.io/opendatahub/data-science-pipelines-operator:pr-<number>` (currently produces `quay.io/rhoai/pull-request-pipelines:...-<revision>`)
- Disable GHA build workflows (`build-main.yml`, `build-prs.yml`) on RHDS — these arrive from sync but RHDS uses Konflux for all builds
- Add RHDS rhoai-* → `:rhoai-*` Konflux config

**PR 6 - DSP (RHDS, after PR 5):**
- **Registry change:** Ensure RHDS DSP CI expects images from `quay.io/opendatahub/` (will inherit this change from ODH sync in PR 4)
- Apply any additional RHDS-specific DSP adjustments if needed

### Critical Timing Requirements

**Simultaneous merges required to avoid breaking DSP CI:**

- **PR 1 & PR 2:** Must merge together - changing ODH main image tags without updating DSP CI breaks ODH main CI
- **PR 3 & PR 4:** Must merge together - bringing new tags to stable without bringing DSP CI changes breaks ODH stable CI  

**Note:** PR 5 & PR 6 can merge independently since RHDS DSP CI is already broken today.