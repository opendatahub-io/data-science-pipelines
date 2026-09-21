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
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	// AIPipelinesCRDName is the cluster-scoped CRD for the modular AIPipelines operand.
	AIPipelinesCRDName = "aipipelines.components.platform.opendatahub.io"
	// AIPipelinesInstanceName is the default cluster-scoped AIPipelines CR installed by the platform.
	AIPipelinesInstanceName = "default-aipipelines"
	// AIPipelinesReleasePlatform is the status.releases entry name for the platform bundle version.
	AIPipelinesReleasePlatform = "platform"
	// AIPipelinesPhaseReady is the reconciled phase for a healthy module after upgrade.
	AIPipelinesPhaseReady = "Ready"
)

var (
	aipipelinesGVR = schema.GroupVersionResource{
		Group:    "components.platform.opendatahub.io",
		Version:  "v1alpha1",
		Resource: "aipipelines",
	}
	aipipelinesCRDGVR = schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
)

// AIPipelinesUpgradeExpectations configures post-upgrade module verification.
type AIPipelinesUpgradeExpectations struct {
	ExpectedPlatformReleaseVersion string
	RequireReleaseVersionMatch       bool
}

// AIPipelinesModuleStatus is a snapshot of module status fields used in upgrade verification.
type AIPipelinesModuleStatus struct {
	Generation          int64
	ObservedGeneration  int64
	Phase               string
	PlatformReleaseName string
	PlatformVersion     string
	Conditions          map[string]unstructuredCondition
}

type unstructuredCondition struct {
	Status             string
	ObservedGeneration int64
	Message            string
}

var aipipelinesUpgradeConditionTypes = []string{
	"Ready",
	"ProvisioningSucceeded",
	"DSPOReady",
	"ArgoWorkflowsControllersReady",
}

// GetAIPipelinesCRD returns the modular AIPipelines CRD when it is installed.
func GetAIPipelinesCRD(ctx context.Context, dynamicClient dynamic.Interface) (*unstructured.Unstructured, error) {
	crd, err := dynamicClient.Resource(aipipelinesCRDGVR).Get(ctx, AIPipelinesCRDName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return crd, nil
}

// AIPipelinesCRDExists reports whether the modular AIPipelines CRD is installed.
func AIPipelinesCRDExists(ctx context.Context, dynamicClient dynamic.Interface) (bool, error) {
	_, err := GetAIPipelinesCRD(ctx, dynamicClient)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetAIPipelinesModule fetches the cluster-scoped default AIPipelines custom resource.
func GetAIPipelinesModule(ctx context.Context, dynamicClient dynamic.Interface) (*unstructured.Unstructured, error) {
	module, err := dynamicClient.Resource(aipipelinesGVR).Get(
		ctx, AIPipelinesInstanceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return module, nil
}

// ParseAIPipelinesModuleStatus extracts upgrade-relevant status from an AIPipelines CR.
func ParseAIPipelinesModuleStatus(module *unstructured.Unstructured) (AIPipelinesModuleStatus, error) {
	if module == nil {
		return AIPipelinesModuleStatus{}, fmt.Errorf("module is nil")
	}
	generation := module.GetGeneration()
	status, found, err := unstructured.NestedMap(module.Object, "status")
	if err != nil {
		return AIPipelinesModuleStatus{}, fmt.Errorf("read status: %w", err)
	}
	if !found || status == nil {
		return AIPipelinesModuleStatus{}, fmt.Errorf("status is empty")
	}

	observedGeneration, err := nestedInt64(status, "observedGeneration")
	if err != nil {
		return AIPipelinesModuleStatus{}, err
	}
	phase, _, err := unstructured.NestedString(status, "phase")
	if err != nil {
		return AIPipelinesModuleStatus{}, fmt.Errorf("read phase: %w", err)
	}

	releaseName, releaseVersion, err := platformReleaseFromStatus(status)
	if err != nil {
		return AIPipelinesModuleStatus{}, err
	}

	conditions, err := conditionsFromStatus(status)
	if err != nil {
		return AIPipelinesModuleStatus{}, err
	}

	return AIPipelinesModuleStatus{
		Generation:          generation,
		ObservedGeneration:  observedGeneration,
		Phase:               phase,
		PlatformReleaseName: releaseName,
		PlatformVersion:     releaseVersion,
		Conditions:          conditions,
	}, nil
}

// VerifyAIPipelinesModuleStatus asserts post-upgrade acceptance criteria on module status.
func VerifyAIPipelinesModuleStatus(
	moduleStatus AIPipelinesModuleStatus,
	expectations AIPipelinesUpgradeExpectations,
) error {
	if moduleStatus.ObservedGeneration != moduleStatus.Generation {
		return fmt.Errorf(
			"observedGeneration %d does not match metadata.generation %d",
			moduleStatus.ObservedGeneration, moduleStatus.Generation)
	}
	if moduleStatus.Phase != AIPipelinesPhaseReady {
		return fmt.Errorf("status.phase is %q, expected %q", moduleStatus.Phase, AIPipelinesPhaseReady)
	}
	if moduleStatus.PlatformReleaseName != AIPipelinesReleasePlatform {
		return fmt.Errorf(
			"platform release name is %q, expected %q",
			moduleStatus.PlatformReleaseName, AIPipelinesReleasePlatform)
	}
	if moduleStatus.PlatformVersion == "" {
		return fmt.Errorf("platform release version is empty")
	}
	if expectations.RequireReleaseVersionMatch {
		if expectations.ExpectedPlatformReleaseVersion == "" {
			return fmt.Errorf("expected platform release version is not configured")
		}
		if moduleStatus.PlatformVersion != expectations.ExpectedPlatformReleaseVersion {
			return fmt.Errorf(
				"platform release version %q does not match expected %q",
				moduleStatus.PlatformVersion, expectations.ExpectedPlatformReleaseVersion)
		}
	}

	for _, conditionType := range aipipelinesUpgradeConditionTypes {
		condition, ok := moduleStatus.Conditions[conditionType]
		if !ok {
			return fmt.Errorf("missing condition %s", conditionType)
		}
		if condition.Status != string(metav1.ConditionTrue) {
			return fmt.Errorf("condition %s status is %q: %s", conditionType, condition.Status, condition.Message)
		}
		if condition.ObservedGeneration != moduleStatus.Generation {
			return fmt.Errorf(
				"condition %s observedGeneration %d does not match metadata.generation %d",
				conditionType, condition.ObservedGeneration, moduleStatus.Generation)
		}
	}
	return nil
}

// VerifyAIPipelinesCRDEstablished ensures the CRD exists and has the Established condition.
func VerifyAIPipelinesCRDEstablished(crd *unstructured.Unstructured) error {
	if crd == nil {
		return fmt.Errorf("CRD is nil")
	}
	conditions, found, err := unstructured.NestedSlice(crd.Object, "status", "conditions")
	if err != nil {
		return fmt.Errorf("read CRD status conditions: %w", err)
	}
	if !found {
		return fmt.Errorf("CRD %s has no status conditions", AIPipelinesCRDName)
	}
	for _, rawCondition := range conditions {
		conditionMap, ok := rawCondition.(map[string]interface{})
		if !ok {
			continue
		}
		conditionType, _, err := unstructured.NestedString(conditionMap, "type")
		if err != nil {
			return fmt.Errorf("read CRD condition type: %w", err)
		}
		if conditionType != "Established" {
			continue
		}
		conditionStatus, _, err := unstructured.NestedString(conditionMap, "status")
		if err != nil {
			return fmt.Errorf("read CRD Established status: %w", err)
		}
		if conditionStatus == string(metav1.ConditionTrue) {
			return nil
		}
		return fmt.Errorf("CRD Established condition is %s", conditionStatus)
	}
	return fmt.Errorf("CRD %s has no Established condition", AIPipelinesCRDName)
}

func platformReleaseFromStatus(status map[string]interface{}) (string, string, error) {
	releases, found, err := unstructured.NestedSlice(status, "releases")
	if err != nil {
		return "", "", fmt.Errorf("read releases: %w", err)
	}
	if !found {
		return "", "", fmt.Errorf("status.releases is missing")
	}
	for _, releaseEntry := range releases {
		releaseMap, ok := releaseEntry.(map[string]interface{})
		if !ok {
			continue
		}
		name, _, err := unstructured.NestedString(releaseMap, "name")
		if err != nil {
			return "", "", fmt.Errorf("read release name: %w", err)
		}
		if name != AIPipelinesReleasePlatform {
			continue
		}
		version, _, err := unstructured.NestedString(releaseMap, "version")
		if err != nil {
			return "", "", fmt.Errorf("read release version: %w", err)
		}
		return name, version, nil
	}
	return "", "", fmt.Errorf("status.releases has no entry with name=%s", AIPipelinesReleasePlatform)
}

func conditionsFromStatus(status map[string]interface{}) (map[string]unstructuredCondition, error) {
	rawConditions, found, err := unstructured.NestedSlice(status, "conditions")
	if err != nil {
		return nil, fmt.Errorf("read conditions: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("status.conditions is missing")
	}
	conditions := make(map[string]unstructuredCondition)
	for _, rawCondition := range rawConditions {
		conditionMap, ok := rawCondition.(map[string]interface{})
		if !ok {
			continue
		}
		conditionType, _, err := unstructured.NestedString(conditionMap, "type")
		if err != nil {
			return nil, fmt.Errorf("read condition type: %w", err)
		}
		conditionStatus, _, err := unstructured.NestedString(conditionMap, "status")
		if err != nil {
			return nil, fmt.Errorf("read condition status: %w", err)
		}
		observedGeneration, err := nestedInt64(conditionMap, "observedGeneration")
		if err != nil {
			return nil, err
		}
		message, _, _ := unstructured.NestedString(conditionMap, "message")
		conditions[conditionType] = unstructuredCondition{
			Status:             conditionStatus,
			ObservedGeneration: observedGeneration,
			Message:            message,
		}
	}
	return conditions, nil
}

func nestedInt64(object map[string]interface{}, field string) (int64, error) {
	value, found, err := unstructured.NestedFieldNoCopy(object, field)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", field, err)
	}
	if !found {
		return 0, fmt.Errorf("%s is missing", field)
	}
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case float64:
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("%s has unsupported type %T", field, value)
	}
}
