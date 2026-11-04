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

package template

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// containerDriverArgsFromWorkflow extracts the container args of the
// system-container-driver template from a marshaled compiled workflow or
// ScheduledWorkflow.
func containerDriverArgsFromWorkflow(t *testing.T, marshaled []byte) []any {
	t.Helper()
	var root map[string]any
	require.NoError(t, json.Unmarshal(marshaled, &root))
	spec, ok := root["spec"].(map[string]any)
	require.True(t, ok, "workflow should have a spec")
	// A ScheduledWorkflow embeds the Argo workflow as a JSON string under
	// spec.workflow.spec.
	if workflow, ok := spec["workflow"].(map[string]any); ok {
		workflowJSON, ok := workflow["spec"].(string)
		require.True(t, ok, "embedded workflow should have a spec")
		require.NoError(t, json.Unmarshal([]byte(workflowJSON), &root))
		spec, ok = root["spec"].(map[string]any)
		require.True(t, ok, "embedded workflow should have a spec")
	}
	templates, ok := spec["templates"].([]any)
	require.True(t, ok, "workflow should have spec.templates")
	for _, tmpl := range templates {
		templateMap, ok := tmpl.(map[string]any)
		if !ok || templateMap["name"] != "system-container-driver" {
			continue
		}
		container, ok := templateMap["container"].(map[string]any)
		require.True(t, ok, "container driver template should have a container")
		args, ok := container["args"].([]any)
		require.True(t, ok, "container driver template should have container args")
		return args
	}
	t.Fatal("workflow should contain a system-container-driver template")
	return nil
}

// flagValue returns the value following flag in args, or false when the flag
// is absent.
func flagValue(args []any, flag string) (string, bool) {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			if value, ok := args[i+1].(string); ok {
				return value, true
			}
		}
	}
	return "", false
}

// TestTemplateOptionsDefaultPVCStorageClassPropagates verifies that the
// server-configured default StorageClass set on TemplateOptions reaches the
// compiled workflow's container-driver arguments, for both one-off runs and
// recurring runs (ScheduledWorkflow).
func TestTemplateOptionsDefaultPVCStorageClassPropagates(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	specBytes, err := os.ReadFile("testdata/pipeline_with_volume.yaml")
	require.NoError(t, err)

	for _, test := range []struct {
		name         string
		defaultClass *string
		wantKey      bool
	}{
		{name: "configured propagates to container driver args", defaultClass: newString("server-default"), wantKey: true},
		{name: "unset omits the flag", wantKey: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpl, err := New(specBytes, TemplateOptions{DefaultPVCStorageClass: test.defaultClass})
			require.NoError(t, err)

			t.Run("RunWorkflow", func(t *testing.T) {
				executionSpec, err := tmpl.RunWorkflow(&model.Run{DisplayName: "run"}, RunWorkflowOptions{})
				require.NoError(t, err)
				marshaled, err := json.Marshal(executionSpec)
				require.NoError(t, err)
				args := containerDriverArgsFromWorkflow(t, marshaled)
				value, ok := flagValue(args, "--default_pvc_storage_class")
				if test.wantKey {
					require.True(t, ok, "container driver args should include --default_pvc_storage_class")
					assert.Equal(t, *test.defaultClass, value)
				} else {
					assert.False(t, ok, "container driver args should not include --default_pvc_storage_class")
				}
			})

			t.Run("ScheduledWorkflow", func(t *testing.T) {
				swf, err := tmpl.ScheduledWorkflow(&model.Job{DisplayName: "job", K8SName: "job"}, nil)
				require.NoError(t, err)
				marshaled, err := json.Marshal(swf)
				require.NoError(t, err)
				args := containerDriverArgsFromWorkflow(t, marshaled)
				value, ok := flagValue(args, "--default_pvc_storage_class")
				if test.wantKey {
					require.True(t, ok, "container driver args should include --default_pvc_storage_class")
					assert.Equal(t, *test.defaultClass, value)
				} else {
					assert.False(t, ok, "container driver args should not include --default_pvc_storage_class")
				}
			})
		})
	}
}

func newString(value string) *string {
	return &value
}
