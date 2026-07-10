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

package mlflow

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// EnvMLflowConfig is the single environment variable injected into Argo
// Workflow templates by the API server.
const EnvMLflowConfig = "KFP_MLFLOW_CONFIG"

// TagNestedRunParentRunID is the MLflow tag used for nested run parent linkage.
const TagNestedRunParentRunID = "mlflow.parentRunId"

// DefaultCABundleDir is the directory where the API server deployment
// mounts the optional mlflow-tracking-ca ConfigMap. When
// TLSConfig.CABundlePath is not set, BuildHTTPClient probes this
// directory for PEM certificate files so that internal CAs (e.g.
// the OpenShift service-serving-cert-signer) are trusted automatically.
const DefaultCABundleDir = "/etc/mlflow-tracking-ca"

// MLflowRuntimeConfig is the JSON payload marshaled into KFP_MLFLOW_CONFIG.
type MLflowRuntimeConfig struct {
	Endpoint           string                   `json:"endpoint"`
	WorkspacesEnabled  bool                     `json:"workspacesEnabled,omitempty"`
	Workspace          string                   `json:"workspace,omitempty"`
	ParentRunID        string                   `json:"parentRunId"`
	ExperimentID       string                   `json:"experimentId"`
	AuthType           string                   `json:"authType"`
	Timeout            string                   `json:"timeout,omitempty"`
	InsecureSkipVerify bool                     `json:"insecureSkipVerify,omitempty"`
	InjectUserEnvVars  bool                     `json:"injectUserEnvVars,omitempty"`
	TLS                *commonplugins.TLSConfig `json:"tls,omitempty" mapstructure:"tls"`
}

// MLflowPluginConfig represents the MLflow plugin configuration.
type MLflowPluginConfig struct {
	Endpoint string                   `json:"endpoint,omitempty" mapstructure:"endpoint"`
	Timeout  string                   `json:"timeout,omitempty" mapstructure:"timeout"`
	TLS      *commonplugins.TLSConfig `json:"tls,omitempty" mapstructure:"tls"`
	Settings *MLflowPluginSettings    `json:"settings,omitempty" mapstructure:"settings"`
}

// MLflowCredentials holds the resolved authentication credentials for an MLflow endpoint.
type MLflowCredentials struct {
	AuthType    string
	BearerToken string
}

// RequestContext holds a fully resolved MLflow connection: the parsed
// endpoint URL, the shared HTTP client, and workspace settings.
type RequestContext struct {
	BaseURL           *url.URL
	Client            *Client
	Workspace         string
	WorkspacesEnabled bool
}

// MLflowPluginSettings contains MLflow-specific settings parsed from
// PluginConfig.Settings.
type MLflowPluginSettings struct {
	WorkspacesEnabled     *bool   `json:"workspacesEnabled,omitempty"`
	ExperimentDescription *string `json:"experimentDescription,omitempty"`
	DefaultExperimentName string  `json:"defaultExperimentName,omitempty"`
	KFPBaseURL            string  `json:"kfpBaseURL,omitempty"`
	KFPRunURLPathTemplate string  `json:"kfpRunURLPathTemplate,omitempty"`
	MLflowBaseURL         string  `json:"mlflowBaseURL,omitempty"`
	MLflowUIPathPrefix    string  `json:"mlflowUIPathPrefix,omitempty"`
	InjectUserEnvVars     *bool   `json:"injectUserEnvVars,omitempty"`
}

// BuildHTTPClient configures an http.Client with the given timeout and TLS settings.
// When tlsCfg is nil or CABundlePath is empty, the function probes
// DefaultCABundleDir for PEM certificate files and appends any found
// certificates to the system cert pool. This allows the API server to
// trust internal CAs (e.g. the OpenShift service-serving-cert-signer)
// without requiring an explicit caBundlePath configuration.
//
// InsecureSkipVerify is rejected: callers must configure proper CA
// certificates instead of disabling TLS verification.
func BuildHTTPClient(timeout time.Duration, tlsCfg *commonplugins.TLSConfig) (*http.Client, error) {
	return buildHTTPClientWithDefaultCADir(timeout, tlsCfg, DefaultCABundleDir)
}

// buildHTTPClientWithDefaultCADir is the internal implementation of
// BuildHTTPClient, accepting the default CA directory as a parameter
// to support deterministic testing.
func buildHTTPClientWithDefaultCADir(timeout time.Duration, tlsCfg *commonplugins.TLSConfig, defaultCADir string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	explicitCAPath := ""
	if tlsCfg != nil {
		if tlsCfg.InsecureSkipVerify {
			return nil, fmt.Errorf("plugins.mlflow.tls.insecureSkipVerify is not supported: configure a CA certificate bundle instead")
		}
		explicitCAPath = tlsCfg.CABundlePath
	}

	tlsConfig := &tls.Config{}

	if explicitCAPath != "" {
		caBundle, err := os.ReadFile(explicitCAPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read plugins.mlflow.tls.caBundlePath %q: %w", explicitCAPath, err)
		}
		certPool, err := x509.SystemCertPool()
		if err != nil {
			certPool = x509.NewCertPool()
		}
		if !certPool.AppendCertsFromPEM(caBundle) {
			return nil, fmt.Errorf("plugins.mlflow.tls.caBundlePath %q did not contain valid PEM certificates", explicitCAPath)
		}
		tlsConfig.RootCAs = certPool
	} else if extraCerts := loadCertsFromDir(defaultCADir); len(extraCerts) > 0 {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			certPool = x509.NewCertPool()
		}
		if !certPool.AppendCertsFromPEM(extraCerts) {
			return nil, fmt.Errorf("default CA directory %q did not contain valid PEM certificates", defaultCADir)
		}
		tlsConfig.RootCAs = certPool
	}

	transport.TLSClientConfig = tlsConfig
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// loadCertsFromDir reads all regular files in dir that have a .crt or
// .pem extension and returns their concatenated contents. Errors are
// logged and skipped so that a single unreadable file does not prevent
// the remaining certificates from being loaded. Returns nil if the
// directory does not exist or contains no matching files.
func loadCertsFromDir(dir string) []byte {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var combined []byte
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".crt" && ext != ".pem" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			glog.Warningf("skipping unreadable CA file %s/%s: %v", dir, entry.Name(), err)
			continue
		}
		combined = append(combined, data...)
	}
	if len(combined) > 0 {
		glog.Infof("loaded additional CA certificates from %s", dir)
	}
	return combined
}

// ResolveMLflowCredentials resolves the Kubernetes service account token used
// to authenticate with the MLflow endpoint.
func ResolveMLflowCredentials() (MLflowCredentials, error) {
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return MLflowCredentials{}, util.NewInternalServerError(err, "failed to get Kubernetes config for MLflow auth")
	}
	token := restConfig.BearerToken
	if token == "" && restConfig.BearerTokenFile != "" {
		tokenBytes, err := os.ReadFile(restConfig.BearerTokenFile)
		if err != nil {
			return MLflowCredentials{}, util.NewInternalServerError(err, "failed to read bearer token file %q for MLflow auth", restConfig.BearerTokenFile)
		}
		token = strings.TrimSpace(string(tokenBytes))
	}
	if token == "" {
		return MLflowCredentials{}, util.NewInvalidInputError("Kubernetes bearer token is empty for MLflow auth")
	}
	return MLflowCredentials{
		AuthType:    AuthTypeKubernetes,
		BearerToken: token,
	}, nil
}

// BuildMLflowRequestContext is the shared core that validates the MLflowPluginConfig,
// resolves credentials, builds the HTTP client and MLflow client, and returns
// a ready-to-use RequestContext. The workspace and workspacesEnabled values
// are caller-specific and passed in directly.
func BuildMLflowRequestContext(pluginCfg MLflowPluginConfig, workspace string, workspacesEnabled bool) (*RequestContext, error) {
	baseURL, err := url.Parse(pluginCfg.Endpoint)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, util.NewInvalidInputError("invalid plugins.mlflow endpoint %q", pluginCfg.Endpoint)
	}
	timeout, err := time.ParseDuration(pluginCfg.Timeout)
	if err != nil {
		return nil, util.NewInvalidInputError("invalid plugins.mlflow timeout %q: %v", pluginCfg.Timeout, err)
	}
	if timeout <= 0 {
		return nil, util.NewInvalidInputError("plugins.mlflow timeout must be > 0")
	}
	authMaterial, err := ResolveMLflowCredentials()
	if err != nil {
		return nil, err
	}
	httpClient, err := BuildHTTPClient(timeout, pluginCfg.TLS)
	if err != nil {
		return nil, err
	}
	retrySettings := RetryPolicy{
		InitialInterval: DefaultRetryInitial,
		MaxInterval:     DefaultRetryMax,
		MaxElapsedTime:  DefaultRetryElapsed,
		Multiplier:      2.0,
	}
	sharedClient, err := NewClient(Config{
		Endpoint:          pluginCfg.Endpoint,
		HTTPClient:        httpClient,
		BearerToken:       authMaterial.BearerToken,
		WorkspacesEnabled: workspacesEnabled,
		Workspace:         workspace,
		Retry:             retrySettings,
	})
	if err != nil {
		return nil, util.NewInvalidInputError("failed to build MLflow client: %v", err)
	}
	return &RequestContext{
		BaseURL:           baseURL,
		Workspace:         workspace,
		WorkspacesEnabled: workspacesEnabled,
		Client:            sharedClient,
	}, nil
}
