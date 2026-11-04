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

package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestCreatePVCStorageClassSemantics verifies the storage-class handling of
// CreatePVC tasks. Precedence: a pipeline-supplied storage_class_name
// (including an explicit empty string opt-out) wins over the server-configured
// default (DefaultPVCStorageClass); when neither supplies a class, the PVC's
// StorageClassName stays nil so Kubernetes applies the cluster default.
func TestCreatePVCStorageClassSemantics(t *testing.T) {
	tests := []struct {
		name string
		// storageClassInput, when non-nil, is passed to the task as the
		// storage_class_name input parameter value.
		storageClassInput *string
		// serverDefault, when non-nil, is configured on the driver as the
		// server-level default StorageClass (DEFAULT_PVC_STORAGE_CLASS).
		serverDefault     *string
		expectedClassName *string
	}{
		{
			name:              "Omitted input leaves StorageClassName nil for Kubernetes defaulting",
			storageClassInput: nil,
			expectedClassName: nil,
		},
		{
			name:              "Explicit class name is preserved",
			storageClassInput: newString("fast-ssd"),
			expectedClassName: newString("fast-ssd"),
		},
		{
			name:              "Explicit empty string is preserved as an opt-out of default selection",
			storageClassInput: newString(""),
			expectedClassName: newString(""),
		},
		{
			name:              "Omitted input uses the configured server default",
			storageClassInput: nil,
			serverDefault:     newString("server-default"),
			expectedClassName: newString("server-default"),
		},
		{
			name:              "Explicit class name overrides the configured server default",
			storageClassInput: newString("fast-ssd"),
			serverDefault:     newString("server-default"),
			expectedClassName: newString("fast-ssd"),
		},
		{
			name:              "Explicit empty string opts out of the configured server default",
			storageClassInput: newString(""),
			serverDefault:     newString("server-default"),
			expectedClassName: newString(""),
		},
		{
			// Defensive: parseOptionalString never produces a pointer to an
			// empty string, but the guard must treat it as unconfigured.
			name:              "Pointer to empty server default is treated as unconfigured",
			storageClassInput: nil,
			serverDefault:     newString(""),
			expectedClassName: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/nested_k8s_pvc_outputs.yaml")
			_, nestedTask := tc.RunDagDriver("nested", tc.RootTask)
			require.NoError(t, tc.Push("createpvc"))
			t.Cleanup(func() {
				_, ok := tc.Pop()
				require.True(t, ok)
			})

			taskSpec := tc.GetLast().GetTaskSpec()
			if tt.storageClassInput != nil {
				if taskSpec.Inputs == nil {
					taskSpec.Inputs = &pipelinespec.TaskInputsSpec{}
				}
				if taskSpec.Inputs.Parameters == nil {
					taskSpec.Inputs.Parameters = map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{}
				}
				taskSpec.Inputs.Parameters["storage_class_name"] = &pipelinespec.TaskInputsSpec_InputParameterSpec{
					Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
						RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
							Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
								Constant: structpb.NewStringValue(*tt.storageClassInput),
							},
						},
					},
				}
			}

			kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
			require.NoError(t, err)
			opts := tc.setupContainerOptions(nestedTask, taskSpec, kubernetesExecutorConfig)
			opts.CacheDisabled = true
			opts.DefaultPVCStorageClass = tt.serverDefault

			execution, err := Container(context.Background(), opts, tc.ClientManager)
			require.NoError(t, err)
			require.NotNil(t, execution)
			require.NotEmpty(t, execution.TaskID)

			task, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
				TaskId: execution.TaskID,
				RunId:  tc.Run.GetRunId(),
			})
			require.NoError(t, err)
			require.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, task.GetState())

			pvcName, ok := taskOutputParameterValue(task, "name")
			require.True(t, ok)
			require.NotEmpty(t, pvcName)

			pvc, err := tc.ClientManager.K8sClient().CoreV1().PersistentVolumeClaims(TestNamespace).Get(
				context.Background(), pvcName, metav1.GetOptions{},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedClassName, pvc.Spec.StorageClassName)
		})
	}
}

func newString(value string) *string {
	return &value
}
