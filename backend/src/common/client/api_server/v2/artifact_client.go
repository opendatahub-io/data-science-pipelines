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

package api_server_v2

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	testconfig "github.com/kubeflow/pipelines/backend/test/config"
	"k8s.io/client-go/tools/clientcmd"
)

type ArtifactClient struct {
	baseURL    string
	httpClient *http.Client
	authHeader string
}

type ArtifactMetadata struct {
	ArtifactID       string `json:"artifactId"`
	ArtifactIDSnake  string `json:"artifact_id"`
	StoragePath      string `json:"storagePath"`
	StoragePathSnake string `json:"storage_path"`
	URI              string `json:"uri"`
}

func NewArtifactClient(clientConfig clientcmd.ClientConfig, debug bool, tlsCfg *tls.Config) (*ArtifactClient, error) {
	httpRuntime, err := api_server.NewHTTPRuntime(clientConfig, debug, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("error occurred when creating artifact client: %w", err)
	}
	return buildArtifactClient(httpRuntime, tlsCfg, ""), nil
}

func NewKubeflowInClusterArtifactClient(namespace string, debug bool, tlsCfg *tls.Config) (*ArtifactClient, error) {
	httpRuntime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, debug, tlsCfg)
	authHeader := getProjectedSATokenAuthHeader()
	return buildArtifactClient(httpRuntime, tlsCfg, authHeader), nil
}

func NewMultiUserArtifactClient(clientConfig clientcmd.ClientConfig, userToken string, debug bool, tlsCfg *tls.Config) (*ArtifactClient, error) {
	httpRuntime, err := api_server.NewHTTPRuntime(clientConfig, debug, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("error occurred when creating artifact client: %w", err)
	}
	return buildArtifactClient(httpRuntime, tlsCfg, normalizeAuthorizationHeader(userToken)), nil
}

func (c *ArtifactClient) ReadArtifact(runID string, nodeID string, artifactName string) ([]byte, error) {
	readEndpoint := fmt.Sprintf("%s/apis/v2beta1/runs/%s/nodes/%s/artifacts/%s:read",
		c.baseURL,
		url.PathEscape(runID),
		url.PathEscape(nodeID),
		url.PathEscape(artifactName),
	)
	body, err := c.getResponseBody(readEndpoint)
	if err != nil {
		return nil, err
	}
	var artifactResponse struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &artifactResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal artifact read response: %w", err)
	}
	if artifactResponse.Data == "" {
		return nil, fmt.Errorf("artifact read response did not include data")
	}
	decodedBody, err := base64.StdEncoding.DecodeString(artifactResponse.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode artifact read payload: %w", err)
	}
	return decodedBody, nil
}

func (c *ArtifactClient) GetArtifactDownloadURL(artifactID string) (string, error) {
	artifactDetailsURL := fmt.Sprintf("%s/apis/v2beta1/artifacts/%s?view=DOWNLOAD",
		c.baseURL,
		url.PathEscape(artifactID),
	)
	body, err := c.getResponseBody(artifactDetailsURL)
	if err != nil {
		return "", err
	}
	var artifactDetails struct {
		DownloadURL      string `json:"downloadUrl"`
		DownloadURLSnake string `json:"download_url"`
	}
	if err := json.Unmarshal(body, &artifactDetails); err != nil {
		return "", fmt.Errorf("failed to unmarshal artifact details response: %w", err)
	}
	downloadURL := artifactDetails.DownloadURL
	if downloadURL == "" {
		downloadURL = artifactDetails.DownloadURLSnake
	}
	if downloadURL == "" {
		return "", fmt.Errorf("artifact details response did not include download url")
	}
	return downloadURL, nil
}

func (c *ArtifactClient) DownloadArtifact(downloadURL string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating download request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed downloading artifact: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("artifact download request failed with status=%d body=%s", response.StatusCode, string(body))
	}
	downloadedBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading downloaded artifact body: %w", err)
	}
	return downloadedBody, nil
}

func (c *ArtifactClient) ListArtifacts(namespace string) ([]ArtifactMetadata, error) {
	var artifacts []ArtifactMetadata
	nextPageToken := ""
	for {
		artifactsURL := fmt.Sprintf("%s/apis/v2beta1/artifacts?namespace=%s&max_result_size=100",
			c.baseURL,
			url.QueryEscape(namespace),
		)
		if nextPageToken != "" {
			artifactsURL += "&next_page_token=" + url.QueryEscape(nextPageToken)
		}
		body, err := c.getResponseBody(artifactsURL)
		if err != nil {
			return nil, err
		}
		var listResponse struct {
			Artifacts     []ArtifactMetadata `json:"artifacts"`
			NextPageToken string             `json:"nextPageToken"`
		}
		if err := json.Unmarshal(body, &listResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal artifacts list response: %w", err)
		}
		artifacts = append(artifacts, listResponse.Artifacts...)
		if listResponse.NextPageToken == "" {
			break
		}
		nextPageToken = listResponse.NextPageToken
	}
	return artifacts, nil
}

func (c *ArtifactClient) getResponseBody(requestURL string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating request for %s: %w", requestURL, err)
	}
	if c.authHeader != "" {
		request.Header.Set("Authorization", c.authHeader)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed sending request to %s: %w", requestURL, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body from %s: %w", requestURL, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request to %s failed with status=%d body=%s", requestURL, response.StatusCode, string(body))
	}
	return body, nil
}

func newArtifactHTTPClient(tlsCfg *tls.Config) *http.Client {
	httpClient := &http.Client{}
	if *testconfig.DisableTLSCheck {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return httpClient
	}
	if tlsCfg != nil {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: tlsCfg,
		}
	}
	return httpClient
}

func normalizeAuthorizationHeader(token string) string {
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmedToken), "bearer ") {
		return trimmedToken
	}
	return "Bearer " + trimmedToken
}

func buildArtifactClient(httpRuntime *httptransport.Runtime, tlsCfg *tls.Config, authHeader string) *ArtifactClient {
	return &ArtifactClient{
		baseURL:    buildBaseURLFromRuntime(httpRuntime, tlsCfg),
		httpClient: newArtifactHTTPClient(tlsCfg),
		authHeader: authHeader,
	}
}

func buildBaseURLFromRuntime(httpRuntime *httptransport.Runtime, tlsCfg *tls.Config) string {
	basePath := strings.TrimSuffix(httpRuntime.BasePath, "/")
	if basePath == "/" {
		basePath = ""
	}
	scheme := "http"
	if parsedURL, parseErr := url.Parse(*testconfig.ApiUrl); parseErr == nil && parsedURL.Scheme != "" {
		scheme = parsedURL.Scheme
	}
	if tlsCfg != nil && !*testconfig.DisableTLSCheck {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, httpRuntime.Host, basePath)
}

func getProjectedSATokenAuthHeader() string {
	const (
		serviceAccountTokenEnvVar = "KF_PIPELINES_SA_TOKEN_PATH"
		defaultServiceAccountPath = "/var/run/secrets/kubeflow/pipelines/token"
	)
	tokenPath := os.Getenv(serviceAccountTokenEnvVar)
	if tokenPath == "" {
		tokenPath = defaultServiceAccountPath
	}
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		return ""
	}
	return normalizeAuthorizationHeader(string(tokenBytes))
}
