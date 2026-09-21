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

package testutil

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseAIPipelinesModuleStatus(t *testing.T) {
	module := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":       AIPipelinesInstanceName,
				"generation": int64(3),
			},
			"status": map[string]interface{}{
				"observedGeneration": int64(3),
				"phase":              AIPipelinesPhaseReady,
				"releases": []interface{}{
					map[string]interface{}{
						"name":    AIPipelinesReleasePlatform,
						"version": "3.2.0",
					},
				},
				"conditions": []interface{}{
					map[string]interface{}{
						"type":               "Ready",
						"status":             "True",
						"observedGeneration": int64(3),
					},
					map[string]interface{}{
						"type":               "ProvisioningSucceeded",
						"status":             "True",
						"observedGeneration": int64(3),
					},
					map[string]interface{}{
						"type":               "DSPOReady",
						"status":             "True",
						"observedGeneration": int64(3),
					},
					map[string]interface{}{
						"type":               "ArgoWorkflowsControllersReady",
						"status":             "True",
						"observedGeneration": int64(3),
					},
				},
			},
		},
	}
	module.SetGeneration(3)

	status, err := ParseAIPipelinesModuleStatus(module)
	if err != nil {
		t.Fatalf("ParseAIPipelinesModuleStatus: %v", err)
	}
	expectations := AIPipelinesUpgradeExpectations{
		ExpectedPlatformReleaseVersion: "3.2.0",
		RequireReleaseVersionMatch:       true,
	}
	if err := VerifyAIPipelinesModuleStatus(status, expectations); err != nil {
		t.Fatalf("VerifyAIPipelinesModuleStatus: %v", err)
	}
}

func TestVerifyAIPipelinesModuleStatusRejectsStaleGeneration(t *testing.T) {
	status := AIPipelinesModuleStatus{
		Generation:          2,
		ObservedGeneration:  1,
		Phase:               AIPipelinesPhaseReady,
		PlatformReleaseName: AIPipelinesReleasePlatform,
		PlatformVersion:     "3.2.0",
		Conditions: map[string]unstructuredCondition{
			"Ready": {Status: string(metav1.ConditionTrue), ObservedGeneration: 1},
		},
	}
	err := VerifyAIPipelinesModuleStatus(status, AIPipelinesUpgradeExpectations{})
	if err == nil {
		t.Fatal("expected observedGeneration mismatch error")
	}
}
