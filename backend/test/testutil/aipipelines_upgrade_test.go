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
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseAIPipelinesModuleStatusHappyPath(t *testing.T) {
	module := validAIPipelinesModuleUnstructured(3, "3.2.0", nil)
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

func TestParseAIPipelinesModuleStatusFloat64ObservedGeneration(t *testing.T) {
	module := validAIPipelinesModuleUnstructured(3, "3.2.0", nil)
	statusMap, _, _ := unstructured.NestedMap(module.Object, "status")
	statusMap["observedGeneration"] = float64(3)
	for _, rawCondition := range statusMap["conditions"].([]interface{}) {
		conditionMap := rawCondition.(map[string]interface{})
		conditionMap["observedGeneration"] = float64(3)
	}

	status, err := ParseAIPipelinesModuleStatus(module)
	if err != nil {
		t.Fatalf("ParseAIPipelinesModuleStatus: %v", err)
	}
	if status.ObservedGeneration != 3 {
		t.Fatalf("expected observedGeneration 3, got %d", status.ObservedGeneration)
	}
}

func TestParseAIPipelinesModuleStatusErrors(t *testing.T) {
	tests := []struct {
		name   string
		module *unstructured.Unstructured
		want   string
	}{
		{
			name:   "nil module",
			module: nil,
			want:   "module is nil",
		},
		{
			name: "missing status",
			module: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{"name": AIPipelinesInstanceName, "generation": int64(1)},
				},
			},
			want: "status is empty",
		},
		{
			name:   "missing releases",
			module: moduleWithStatus(map[string]interface{}{"observedGeneration": int64(1), "phase": AIPipelinesPhaseReady}),
			want:   "status.releases is missing",
		},
		{
			name: "no platform release entry",
			module: moduleWithStatus(map[string]interface{}{
				"observedGeneration": int64(1),
				"phase":              AIPipelinesPhaseReady,
				"releases": []interface{}{
					map[string]interface{}{"name": "other", "version": "1.0.0"},
				},
			}),
			want: "status.releases has no entry with name=platform",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ParseAIPipelinesModuleStatus(testCase.module)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

func TestVerifyAIPipelinesModuleStatusFailures(t *testing.T) {
	fullConditions := allTrueConditions(3)

	tests := []struct {
		name         string
		status       AIPipelinesModuleStatus
		expectations AIPipelinesUpgradeExpectations
		want         string
	}{
		{
			name: "stale status observedGeneration",
			status: AIPipelinesModuleStatus{
				Generation: 2, ObservedGeneration: 1, Phase: AIPipelinesPhaseReady,
				PlatformReleaseName: AIPipelinesReleasePlatform, PlatformVersion: "3.2.0",
				Conditions: fullConditions,
			},
			expectations: AIPipelinesUpgradeExpectations{},
			want:         "observedGeneration",
		},
		{
			name: "wrong platform version",
			status: AIPipelinesModuleStatus{
				Generation: 3, ObservedGeneration: 3, Phase: AIPipelinesPhaseReady,
				PlatformReleaseName: AIPipelinesReleasePlatform, PlatformVersion: "3.1.0",
				Conditions: fullConditions,
			},
			expectations: AIPipelinesUpgradeExpectations{
				ExpectedPlatformReleaseVersion: "3.2.0",
				RequireReleaseVersionMatch:       true,
			},
			want: "does not match expected",
		},
		{
			name: "empty platform version with match required",
			status: AIPipelinesModuleStatus{
				Generation: 3, ObservedGeneration: 3, Phase: AIPipelinesPhaseReady,
				PlatformReleaseName: AIPipelinesReleasePlatform, PlatformVersion: "",
				Conditions: fullConditions,
			},
			expectations: AIPipelinesUpgradeExpectations{RequireReleaseVersionMatch: true, ExpectedPlatformReleaseVersion: "3.2.0"},
			want:         "platform release version is empty",
		},
		{
			name: "missing DSPOReady condition",
			status: AIPipelinesModuleStatus{
				Generation: 3, ObservedGeneration: 3, Phase: AIPipelinesPhaseReady,
				PlatformReleaseName: AIPipelinesReleasePlatform, PlatformVersion: "3.2.0",
				Conditions: map[string]unstructuredCondition{
					"Ready":                         {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
					"ProvisioningSucceeded":           {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
					"ArgoWorkflowsControllersReady": {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
				},
			},
			expectations: AIPipelinesUpgradeExpectations{},
			want:         "missing condition DSPOReady",
		},
		{
			name: "condition not True includes message",
			status: AIPipelinesModuleStatus{
				Generation: 3, ObservedGeneration: 3, Phase: AIPipelinesPhaseReady,
				PlatformReleaseName: AIPipelinesReleasePlatform, PlatformVersion: "3.2.0",
				Conditions: map[string]unstructuredCondition{
					"Ready":                         {Status: string(metav1.ConditionFalse), ObservedGeneration: 3, Message: "still reconciling"},
					"ProvisioningSucceeded":           {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
					"DSPOReady":                       {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
					"ArgoWorkflowsControllersReady": {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
				},
			},
			expectations: AIPipelinesUpgradeExpectations{},
			want:         "still reconciling",
		},
		{
			name: "condition observedGeneration mismatch",
			status: AIPipelinesModuleStatus{
				Generation: 3, ObservedGeneration: 3, Phase: AIPipelinesPhaseReady,
				PlatformReleaseName: AIPipelinesReleasePlatform, PlatformVersion: "3.2.0",
				Conditions: map[string]unstructuredCondition{
					"Ready":                         {Status: string(metav1.ConditionTrue), ObservedGeneration: 2},
					"ProvisioningSucceeded":           {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
					"DSPOReady":                       {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
					"ArgoWorkflowsControllersReady": {Status: string(metav1.ConditionTrue), ObservedGeneration: 3},
				},
			},
			expectations: AIPipelinesUpgradeExpectations{},
			want:         "condition Ready observedGeneration",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := VerifyAIPipelinesModuleStatus(testCase.status, testCase.expectations)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

func TestVerifyAIPipelinesCRDEstablished(t *testing.T) {
	tests := []struct {
		name    string
		crd     *unstructured.Unstructured
		wantErr string
	}{
		{
			name:    "nil CRD",
			crd:     nil,
			wantErr: "CRD is nil",
		},
		{
			name: "Established True",
			crd: crdWithEstablishedConditions([]map[string]interface{}{
				{"type": "Established", "status": "True"},
			}),
			wantErr: "",
		},
		{
			name: "Established False",
			crd: crdWithEstablishedConditions([]map[string]interface{}{
				{"type": "Established", "status": "False"},
			}),
			wantErr: "Established condition is False",
		},
		{
			name:    "Established absent",
			crd:     crdWithEstablishedConditions([]map[string]interface{}{{"type": "NamesAccepted", "status": "True"}}),
			wantErr: "has no Established condition",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := VerifyAIPipelinesCRDEstablished(testCase.crd)
			if testCase.wantErr == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("expected error containing %q, got %v", testCase.wantErr, err)
			}
		})
	}
}

func validAIPipelinesModuleUnstructured(
	generation int64,
	platformVersion string,
	overrideStatus map[string]interface{},
) *unstructured.Unstructured {
	status := map[string]interface{}{
		"observedGeneration": generation,
		"phase":              AIPipelinesPhaseReady,
		"releases": []interface{}{
			map[string]interface{}{
				"name":    AIPipelinesReleasePlatform,
				"version": platformVersion,
			},
		},
		"conditions": conditionsSlice(allTrueConditions(generation)),
	}
	for key, value := range overrideStatus {
		status[key] = value
	}
	module := moduleWithStatus(status)
	module.SetGeneration(generation)
	return module
}

func moduleWithStatus(status map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":       AIPipelinesInstanceName,
				"generation": status["observedGeneration"],
			},
			"status": status,
		},
	}
}

func allTrueConditions(generation int64) map[string]unstructuredCondition {
	conditions := make(map[string]unstructuredCondition)
	for _, conditionType := range aipipelinesUpgradeConditionTypes {
		conditions[conditionType] = unstructuredCondition{
			Status:             string(metav1.ConditionTrue),
			ObservedGeneration: generation,
		}
	}
	return conditions
}

func conditionsSlice(conditions map[string]unstructuredCondition) []interface{} {
	slice := make([]interface{}, 0, len(conditions))
	for _, conditionType := range aipipelinesUpgradeConditionTypes {
		condition := conditions[conditionType]
		slice = append(slice, map[string]interface{}{
			"type":               conditionType,
			"status":             condition.Status,
			"observedGeneration": condition.ObservedGeneration,
			"message":            condition.Message,
		})
	}
	return slice
}

func crdWithEstablishedConditions(rawConditions []map[string]interface{}) *unstructured.Unstructured {
	conditions := make([]interface{}, len(rawConditions))
	for index, condition := range rawConditions {
		conditions[index] = condition
	}
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{"name": AIPipelinesCRDName},
			"status":   map[string]interface{}{"conditions": conditions},
		},
	}
}
