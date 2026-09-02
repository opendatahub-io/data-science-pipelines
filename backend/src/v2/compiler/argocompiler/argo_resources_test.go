// Copyright 2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
)

func TestGetDriverResources_Defaults(t *testing.T) {
	res := GetDriverResources()

	assert.Equal(t, k8sres.MustParse("0.1"), res.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("64Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("0.5"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("0.5Gi"), res.Limits[k8score.ResourceMemory])
}

func TestGetLauncherResources_Defaults(t *testing.T) {
	res := GetLauncherResources()

	assert.Equal(t, k8sres.MustParse("0.1"), res.Requests[k8score.ResourceCPU])
	_, hasMemReq := res.Requests[k8score.ResourceMemory]
	assert.False(t, hasMemReq, "default launcher should not have a memory request")
	assert.Equal(t, k8sres.MustParse("0.5"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("256Mi"), res.Limits[k8score.ResourceMemory])
}

func TestGetDriverResources_AllEnvVars(t *testing.T) {
	t.Setenv("V2_DRIVER_RESOURCE_REQUESTS_CPU", "250m")
	t.Setenv("V2_DRIVER_RESOURCE_REQUESTS_MEMORY", "128Mi")
	t.Setenv("V2_DRIVER_RESOURCE_LIMITS_CPU", "1")
	t.Setenv("V2_DRIVER_RESOURCE_LIMITS_MEMORY", "1Gi")

	res := GetDriverResources()

	assert.Equal(t, k8sres.MustParse("250m"), res.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("128Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("1"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("1Gi"), res.Limits[k8score.ResourceMemory])
}

func TestGetLauncherResources_AllEnvVars(t *testing.T) {
	t.Setenv("V2_LAUNCHER_RESOURCE_REQUESTS_CPU", "200m")
	t.Setenv("V2_LAUNCHER_RESOURCE_REQUESTS_MEMORY", "64Mi")
	t.Setenv("V2_LAUNCHER_RESOURCE_LIMITS_CPU", "2")
	t.Setenv("V2_LAUNCHER_RESOURCE_LIMITS_MEMORY", "512Mi")

	res := GetLauncherResources()

	assert.Equal(t, k8sres.MustParse("200m"), res.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("64Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("2"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("512Mi"), res.Limits[k8score.ResourceMemory])
}

func TestGetDriverResources_PartialOverride(t *testing.T) {
	t.Setenv("V2_DRIVER_RESOURCE_REQUESTS_CPU", "300m")
	t.Setenv("V2_DRIVER_RESOURCE_LIMITS_MEMORY", "2Gi")

	res := GetDriverResources()

	// Overridden values
	assert.Equal(t, k8sres.MustParse("300m"), res.Requests[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("2Gi"), res.Limits[k8score.ResourceMemory])

	// Default values for the rest
	assert.Equal(t, k8sres.MustParse("64Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("0.5"), res.Limits[k8score.ResourceCPU])
}

func TestGetDriverResources_InvalidEnvVarFallsBack(t *testing.T) {
	t.Setenv("V2_DRIVER_RESOURCE_REQUESTS_CPU", "not-a-quantity")

	res := GetDriverResources()

	// Should fall back to default for the invalid env var
	assert.Equal(t, k8sres.MustParse("0.1"), res.Requests[k8score.ResourceCPU])
	// Others should remain at defaults
	assert.Equal(t, k8sres.MustParse("64Mi"), res.Requests[k8score.ResourceMemory])
	assert.Equal(t, k8sres.MustParse("0.5"), res.Limits[k8score.ResourceCPU])
	assert.Equal(t, k8sres.MustParse("0.5Gi"), res.Limits[k8score.ResourceMemory])
}
