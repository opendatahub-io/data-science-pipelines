// Copyright 2021 Arrikto Inc.
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

package resource

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// TestIsAuthorized_TokenReviewWinsOverSpoofedHeader is a regression test for
// CWE-290 (header-spoofing bypass). It exercises ResourceManager.IsAuthorized
// end-to-end: when a request carries both a valid bearer token and a spoofed
// kubeflow-userid header, the SAR must receive the token-derived identity, not
// the attacker-controlled header value.
func TestIsAuthorized_TokenReviewWinsOverSpoofedHeader(t *testing.T) {
	const spoofedUser = "attacker@evil.com"

	previousMultiUserMode := viper.GetString(common.MultiUserMode)
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() {
		viper.Set(common.MultiUserMode, previousMultiUserMode)
	})

	recordingSARClient := &client.RecordingSubjectAccessReviewClient{}

	fakeClientManager := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	fakeClientManager.SubjectAccessReviewClientFake = recordingSARClient

	resourceManager := NewResourceManager(fakeClientManager, &ResourceManagerOptions{CollectMetrics: false})

	md := metadata.New(map[string]string{
		common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "valid-service-account-token",
		common.GetKubeflowUserIDHeader():      common.GetKubeflowUserIDPrefix() + spoofedUser,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	err := resourceManager.IsAuthorized(ctx, &authorizationv1.ResourceAttributes{
		Namespace: "test-ns",
		Verb:      common.RbacResourceVerbGet,
		Resource:  "pipelines",
	})

	require.NoError(t, err)
	assert.Equal(t, "test", recordingSARClient.LastUser,
		"SAR must receive the token-derived identity, not the spoofed header")
	assert.NotEqual(t, spoofedUser, recordingSARClient.LastUser,
		"SAR must not receive the attacker-controlled header value")
}
