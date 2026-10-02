// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package constants

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2eCriticalShardForPipelineCoversCriticalPipelines(t *testing.T) {
	criticalPipelineDirectory := filepath.Join("..", "..", "..", "test_data", "sdk_compiled_pipelines", "valid", "critical")
	entries, err := os.ReadDir(criticalPipelineDirectory)
	if err != nil {
		t.Fatalf("read critical pipeline directory: %v", err)
	}

	shardCounts := map[string]int{}
	assignedToShardA := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		shard := E2eCriticalShardForPipeline(entry.Name())
		shardCounts[shard]++
		if shard == E2eCriticalShardA {
			assignedToShardA[entry.Name()] = true
		}
	}

	for pipelineFile := range e2eCriticalShardAPipelines {
		if !assignedToShardA[pipelineFile] {
			t.Errorf("shard A pipeline %q is missing from critical fixtures", pipelineFile)
		}
	}
	if got := shardCounts[E2eCriticalShardB]; got == 0 {
		t.Error("shard B contains no critical pipelines")
	}
	if got := len(shardCounts); got != 2 {
		t.Errorf("critical pipelines use %d shard labels, want 2: %v", got, shardCounts)
	}
}

func TestE2eCriticalShardForPipelineDefaultsToShardB(t *testing.T) {
	if got := E2eCriticalShardForPipeline("new_pipeline.yaml"); got != E2eCriticalShardB {
		t.Fatalf("new pipeline assigned to %q, want %q", got, E2eCriticalShardB)
	}
}
