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
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
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
			k8sClient := k8sfake.NewSimpleClientset()
			mlmdClient := newCreatePVCTestMLMDClient()

			parameterValues := map[string]*structpb.Value{
				"access_modes": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{structpb.NewStringValue("ReadWriteOnce")},
				}),
				"size": structpb.NewStringValue("5Mi"),
			}
			if tt.storageClassInput != nil {
				parameterValues["storage_class_name"] = structpb.NewStringValue(*tt.storageClassInput)
			}

			execution := Execution{
				ExecutorInput: &pipelinespec.ExecutorInput{
					Inputs: &pipelinespec.ExecutorInput_Inputs{
						ParameterValues: parameterValues,
					},
				},
			}

			pvcName, _, status, err := createPVC(
				context.Background(),
				k8sClient,
				execution,
				&Options{
					PipelineName:   "pipeline-1",
					RunID:          "run-1",
					TaskName:       "createpvc-1",
					Namespace:      "kubeflow",
					DAGExecutionID: 1,
					Task: &pipelinespec.PipelineTaskSpec{
						TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "createpvc-1"},
					},
					Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
						Image: "argostub/createpvc",
					},
					CacheDisabled:          true,
					PluginDispatcher:       plugins.NoOpDispatcher{},
					DefaultPVCStorageClass: tt.serverDefault,
				},
				&mockCacheClient{},
				mlmdClient,
				&metadata.ExecutionConfig{},
			)
			require.NoError(t, err)
			require.Equal(t, pb.Execution_COMPLETE, status)
			require.NotEmpty(t, pvcName)

			pvc, err := k8sClient.CoreV1().PersistentVolumeClaims("kubeflow").Get(
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

// newCreatePVCTestMLMDClient returns an MLMD client that satisfies the
// pipeline and execution bookkeeping performed by createPVC.
func newCreatePVCTestMLMDClient() *metadata.Client {
	mockSvc := &MockMetadataClient{
		GetContextByTypeAndNameFunc: func(ctx context.Context, in *pb.GetContextByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetContextByTypeAndNameResponse, error) {
			return &pb.GetContextByTypeAndNameResponse{Context: &pb.Context{Id: proto.Int64(1234)}}, nil
		},
		GetParentContextsByContextFunc: func(ctx context.Context, in *pb.GetParentContextsByContextRequest, opts ...grpc.CallOption) (*pb.GetParentContextsByContextResponse, error) {
			return &pb.GetParentContextsByContextResponse{}, nil
		},
		PutParentContextsFunc: func(ctx context.Context, in *pb.PutParentContextsRequest, opts ...grpc.CallOption) (*pb.PutParentContextsResponse, error) {
			return &pb.PutParentContextsResponse{}, nil
		},
		PutExecutionTypeFunc: func(ctx context.Context, in *pb.PutExecutionTypeRequest, opts ...grpc.CallOption) (*pb.PutExecutionTypeResponse, error) {
			return &pb.PutExecutionTypeResponse{TypeId: proto.Int64(7)}, nil
		},
		PutExecutionFunc: func(ctx context.Context, in *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
			return &pb.PutExecutionResponse{}, nil
		},
		GetExecutionsByIDFunc: func(ctx context.Context, in *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
			return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{{Id: proto.Int64(55)}}}, nil
		},
	}
	return metadata.NewTestClient(mockSvc)
}
