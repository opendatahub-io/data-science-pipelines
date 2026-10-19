// Copyright 2026 The Kubeflow Authors
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

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowPodLabels(t *testing.T) {
	t.Run("omits the application name when it is empty", func(t *testing.T) {
		viper.Set(common.DSPANameFlag, "")
		t.Cleanup(func() { viper.Set(common.DSPANameFlag, "") })

		labels := workflowPodLabels()

		assert.Equal(t, "true", labels[v2ComponentLabelKey])
		_, hasApplicationName := labels[applicationNameLabelKey]
		assert.False(t, hasApplicationName)
	})

	t.Run("writes the application name when the API server has one", func(t *testing.T) {
		viper.Set(common.DSPANameFlag, "sample")
		t.Cleanup(func() { viper.Set(common.DSPANameFlag, "") })

		labels := workflowPodLabels()

		assert.Equal(t, "true", labels[v2ComponentLabelKey])
		assert.Equal(t, "sample", labels[applicationNameLabelKey])
	})
}
