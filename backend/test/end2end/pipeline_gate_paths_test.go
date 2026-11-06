// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package end2end

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/test/testutil"
)

func TestE2eRelativePathUnderValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		pipelineDir  string
		pipelineFile string
		want         string
	}{
		{
			name:         "essential folder basename",
			pipelineDir:  "valid/essential",
			pipelineFile: "iris_pipeline_compiled.yaml",
			want:         filepath.Clean("essential/iris_pipeline_compiled.yaml"),
		},
		{
			name:         "critical folder basename",
			pipelineDir:  "valid/critical",
			pipelineFile: "flip_coin.yaml",
			want:         filepath.Clean("critical/flip_coin.yaml"),
		},
		{
			name:         "valid root relative path",
			pipelineDir:  "valid",
			pipelineFile: "essential/iris_pipeline_compiled.yaml",
			want:         filepath.Clean("essential/iris_pipeline_compiled.yaml"),
		},
		{
			name:         "dot-slash valid prefix cleaned",
			pipelineDir:  "./valid/essential",
			pipelineFile: "iris_pipeline_compiled.yaml",
			want:         filepath.Clean("essential/iris_pipeline_compiled.yaml"),
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := e2eRelativePathUnderValid(testCase.pipelineDir, testCase.pipelineFile)
			if got != testCase.want {
				t.Fatalf("e2eRelativePathUnderValid(%q, %q) = %q, want %q", testCase.pipelineDir, testCase.pipelineFile, got, testCase.want)
			}
		})
	}
}

func TestE2ePathInSetUsesRelativePathNotBasename(t *testing.T) {
	t.Parallel()

	pathSet := e2ePipelinePathSet([]string{"critical/flip_coin.yaml"})

	if !e2ePathInSet(pathSet, "valid/critical", "flip_coin.yaml") {
		t.Fatal("expected critical/flip_coin.yaml to be in the path set")
	}
	if e2ePathInSet(pathSet, "valid/essential", "flip_coin.yaml") {
		t.Fatal("basename-only match must not skip essential/flip_coin.yaml")
	}
}

// TestE2eSmokeAndTier1GatePathListsConsistent guards the configured gate slices
// against drift with the dedicated Smoke/Tier1 Ginkgo loops and folder-sweep skips.
func TestE2eSmokeAndTier1GatePathListsConsistent(t *testing.T) {
	t.Parallel()

	validRoot := testutil.GetValidPipelineFilesDir()
	assertPathsExistUnique(t, validRoot, e2eSmokePipelinePaths)
	assertPathsExistUnique(t, validRoot, e2eTier1PipelinePaths)
	assertPathsExistUnique(t, validRoot, e2eDisconnectedPipelinePaths)

	for smokePath := range e2eSmokePipelinePathSet {
		if _, overlap := e2eTier1PipelinePathSet[smokePath]; overlap {
			t.Fatalf("path %q is in both smoke and Tier1 gate lists", smokePath)
		}
	}

	for _, relativePath := range e2eDisconnectedPipelinePaths {
		if _, inSmoke := e2eSmokePipelinePathSet[filepath.Clean(relativePath)]; !inSmoke {
			t.Fatalf("disconnected gate path %q must be listed in e2eSmokePipelinePaths", relativePath)
		}
	}

	for _, relativePath := range e2eTier1PipelinePaths {
		pipelineDir := e2eFolderSweepPipelineDir(relativePath)
		pipelineFile := filepath.Base(relativePath)
		if !e2ePathInSet(e2eTier1PipelinePathSet, pipelineDir, pipelineFile) {
			t.Fatalf("Tier1 gate path %q must be skipped in Tier2 folder sweep for %s", relativePath, pipelineDir)
		}
	}

	for _, relativePath := range e2eSmokePipelinePaths {
		pipelineDir := e2eFolderSweepPipelineDir(relativePath)
		pipelineFile := filepath.Base(relativePath)
		if !e2ePathInSet(e2eSmokePipelinePathSet, pipelineDir, pipelineFile) {
			t.Fatalf("Smoke gate path %q must be skipped in folder sweep for %s", relativePath, pipelineDir)
		}
	}
}

func TestE2eMultiUserCriticalPathSet(t *testing.T) {
	t.Parallel()

	if len(e2eMultiUserCriticalPathSet) != 6 {
		t.Fatalf("expected 6 multi-user E2ECritical paths, got %d", len(e2eMultiUserCriticalPathSet))
	}
	for _, relativePath := range e2eSmokePipelinePaths {
		cleaned := filepath.Clean(relativePath)
		inSet := e2eRelativePathInMultiUserCriticalSet(cleaned)
		if strings.HasPrefix(cleaned, "critical/") && !inSet {
			t.Fatalf("expected smoke critical path %q in multi-user critical set", cleaned)
		}
		if !strings.HasPrefix(cleaned, "critical/") && inSet {
			t.Fatalf("expected non-critical smoke path %q outside multi-user critical set", cleaned)
		}
	}
	for _, relativePath := range e2eTier1PipelinePaths {
		cleaned := filepath.Clean(relativePath)
		inSet := e2eRelativePathInMultiUserCriticalSet(cleaned)
		if strings.HasPrefix(cleaned, "critical/") && !inSet {
			t.Fatalf("expected Tier1 critical path %q in multi-user critical set", cleaned)
		}
		if !strings.HasPrefix(cleaned, "critical/") && inSet {
			t.Fatalf("expected non-critical Tier1 path %q outside multi-user critical set", cleaned)
		}
	}
}

// e2eFolderSweepPipelineDir maps a valid/-relative IR path to the pipelineDir used in folder sweeps.
func e2eFolderSweepPipelineDir(relativePath string) string {
	subdir := filepath.Dir(relativePath)
	if subdir == "." {
		return "valid"
	}
	return filepath.Join("valid", subdir)
}

func assertPathsExistUnique(t *testing.T, validRoot string, relativePaths []string) {
	t.Helper()

	seen := make(map[string]struct{}, len(relativePaths))
	for _, relativePath := range relativePaths {
		cleaned := filepath.Clean(relativePath)
		if cleaned != relativePath {
			t.Fatalf("path %q must already be filepath.Clean", relativePath)
		}
		if _, dup := seen[cleaned]; dup {
			t.Fatalf("duplicate gate path %q", cleaned)
		}
		seen[cleaned] = struct{}{}
		if _, err := os.Stat(filepath.Join(validRoot, cleaned)); err != nil {
			t.Fatalf("gate path %q missing under valid/: %v", cleaned, err)
		}
	}
}
