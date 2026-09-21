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

package api

import (
	"context"
	"os"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"
)

const (
	aipipelinesUpgradePollInterval = 5 * time.Second
	aipipelinesUpgradeTimeout      = 8 * time.Minute
)

var _ = Describe("Upgrade Test Verification >", Label(constants.UpgradeVerification, constants.FullRegression), func() {
	Context("Verify modular AIPipelines after platform upgrade >", func() {
		It("verifies the default AIPipelines module is reconciled and reports the upgraded platform release", func() {
			ctx := context.Background()
			restConfig, err := util.GetKubernetesConfig()
			Expect(err).NotTo(HaveOccurred(), "failed to build Kubernetes REST config")
			dynamicClient, err := dynamic.NewForConfig(restConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to create dynamic Kubernetes client")

			crdExists, err := testutil.AIPipelinesCRDExists(ctx, dynamicClient)
			Expect(err).NotTo(HaveOccurred(), "failed to look up AIPipelines CRD")
			if !crdExists {
				Skip("AIPipelines CRD is not installed; skipping modular upgrade verification")
			}

			crd, err := testutil.GetAIPipelinesCRD(ctx, dynamicClient)
			Expect(err).NotTo(HaveOccurred(), "failed to get AIPipelines CRD")
			Expect(testutil.VerifyAIPipelinesCRDEstablished(crd)).To(Succeed(), "AIPipelines CRD should be Established")

			expectedVersion := resolveExpectedPlatformReleaseVersion()
			requireVersionMatch := expectedVersion != ""
			if !requireVersionMatch {
				logger.Log(
					"EXPECTED_PLATFORM_RELEASE_VERSION is unset; verifying platform release presence without version equality")
			}

			expectations := testutil.AIPipelinesUpgradeExpectations{
				ExpectedPlatformReleaseVersion: expectedVersion,
				RequireReleaseVersionMatch:       requireVersionMatch,
			}

			Eventually(func(g Gomega) {
				module, getErr := testutil.GetAIPipelinesModule(ctx, dynamicClient)
				g.Expect(getErr).NotTo(HaveOccurred(), "failed to get cluster-scoped AIPipelines %s",
					testutil.AIPipelinesInstanceName)

				moduleStatus, parseErr := testutil.ParseAIPipelinesModuleStatus(module)
				g.Expect(parseErr).NotTo(HaveOccurred(), "failed to parse AIPipelines status")

				verifyErr := testutil.VerifyAIPipelinesModuleStatus(moduleStatus, expectations)
				g.Expect(verifyErr).NotTo(HaveOccurred())
			}).WithTimeout(aipipelinesUpgradeTimeout).
				WithPolling(aipipelinesUpgradePollInterval).
				Should(Succeed())
		})
	})
})

func resolveExpectedPlatformReleaseVersion() string {
	if config.ExpectedPlatformReleaseVersion != nil && *config.ExpectedPlatformReleaseVersion != "" {
		return *config.ExpectedPlatformReleaseVersion
	}
	return os.Getenv("EXPECTED_PLATFORM_RELEASE_VERSION")
}
