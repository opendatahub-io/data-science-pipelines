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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
<<<<<<< HEAD
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
=======
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
>>>>>>> upstream/master
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
<<<<<<< HEAD
	"google.golang.org/protobuf/types/known/structpb"
=======
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
>>>>>>> upstream/master
)

const (
	// DefaultExperimentName is the MLflow experiment name used when the user
	// and admin configuration do not specify one.
<<<<<<< HEAD
	DefaultExperimentName        = "KFP-Default"
	DefaultExperimentDescription = "Created by Kubeflow Pipelines"
	PluginName                   = "MLflow"
	TagKFPRunID                  = "kfp.pipeline_run_id"
	TagKFPRunURL                 = "kfp.pipeline_run_url"
	TagKFPPipelineID             = "kfp.pipeline_id"
	TagKFPPipelineVersionID      = "kfp.pipeline_version_id"
)

=======
	DefaultExperimentName = "KFP-Default"
	// DefaultTimeout is the default HTTP request timeout for the MLflow client.
	DefaultTimeout = "30s"
	PluginName     = "mlflow"
)

const (
	LauncherConfigMapName = "kfp-launcher"
	LauncherConfigKey     = "plugins.mlflow"
)

// LauncherNamespaceMLflowConfig is the restricted MLflow override shape allowed in the
// namespace-scoped kfp-launcher ConfigMap.
type LauncherNamespaceMLflowConfig struct {
	Settings *LauncherNamespaceMLflowSettings `json:"settings,omitempty"`
}

// LauncherNamespaceMLflowSettings lists the only MLflow settings that a namespace may
// override through the kfp-launcher ConfigMap.
type LauncherNamespaceMLflowSettings struct {
	ExperimentDescription *string                            `json:"experimentDescription,omitempty"`
	DefaultExperimentName string                             `json:"defaultExperimentName,omitempty"`
	InjectUserEnvVars     *bool                              `json:"injectUserEnvVars,omitempty"`
	CredentialSecretRef   *commonplugins.CredentialSecretRef `json:"credentialSecretRef,omitempty"`
}

// ApplyMLflowSettingsDefaults applies default values to a parsed MLflowPluginSettings.
func ApplyMLflowSettingsDefaults(settings *commonmlflow.MLflowPluginSettings) *commonmlflow.MLflowPluginSettings {
	if settings == nil {
		settings = &commonmlflow.MLflowPluginSettings{}
	}
	if settings.AuthType == "" {
		settings.AuthType = commonmlflow.AuthTypeKubernetes
	}
	if settings.WorkspacesEnabled == nil {
		defaultEnabled := settings.AuthType == commonmlflow.AuthTypeKubernetes
		settings.WorkspacesEnabled = &defaultEnabled
	}
	if settings.DefaultExperimentName == "" {
		settings.DefaultExperimentName = DefaultExperimentName
	}
	if settings.ExperimentDescription == nil {
		d := DefaultExperimentDescription
		settings.ExperimentDescription = &d
	}
	return settings
}

// ResolvedMLflowConfig bundles the merged, defaulted plugin configuration and its
// resolved credentials.
type ResolvedMLflowConfig struct {
	Config      *commonmlflow.MLflowPluginConfig
	Credentials commonmlflow.MLflowCredentials
}

func newResolvedMLflowConfig(config *commonmlflow.MLflowPluginConfig, credentials commonmlflow.MLflowCredentials) (*ResolvedMLflowConfig, error) {
	if config == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow config is nil"), "resolved MLflow config requires plugin config")
	}
	if config.Settings == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow plugin settings are nil"), "resolved MLflow config requires plugin settings")
	}
	if credentials.AuthType == "" {
		return nil, util.NewInternalServerError(
			fmt.Errorf("missing resolved credentials for auth type %q", config.Settings.AuthType),
			"resolved MLflow config requires credentials",
		)
	}
	return &ResolvedMLflowConfig{
		Config:      config,
		Credentials: credentials,
	}, nil
}

>>>>>>> upstream/master
// MLflowPluginInput represents the user-facing plugins_input.mlflow schema.
type MLflowPluginInput struct {
	ExperimentName string `json:"experiment_name,omitempty"`
	ExperimentID   string `json:"experiment_id,omitempty"`
	Disabled       bool   `json:"disabled,omitempty"`
}

// IsEnabled reports whether the global plugins.mlflow configuration is present,
// indicating the API server has opted in to MLflow integration.
func IsEnabled() bool {
	return viper.IsSet("plugins.mlflow")
}

<<<<<<< HEAD
type Experiment struct {
	ID   string
	Name string
}

// ResolveMLflowPluginConfig builds an MLflowPluginConfig for the given input generic PluginConfig.
func ResolveMLflowPluginConfig(runPluginConfig *apiserverPlugins.PluginConfig, resolvedMLflowSettings *commonmlflow.MLflowPluginSettings) (*commonmlflow.MLflowPluginConfig, error) {
	if runPluginConfig == nil || resolvedMLflowSettings == nil {
		return nil, fmt.Errorf("runPluginConfig and resolvedMLflowSettings must be non-nil")
	}

	resolvedTimeout := runPluginConfig.Timeout
	if resolvedTimeout == "" {
		resolvedTimeout = apiserverPlugins.DefaultTimeout
	}

	resolvedMLflowCfg := &commonmlflow.MLflowPluginConfig{
		Endpoint: runPluginConfig.Endpoint,
		Timeout:  resolvedTimeout,
		TLS:      runPluginConfig.TLS,
		Settings: resolvedMLflowSettings,
	}
	return resolvedMLflowCfg, nil
=======
// GetGlobalMLflowConfig reads the global plugins.mlflow configuration
func GetGlobalMLflowConfig() (commonmlflow.MLflowPluginConfig, bool, error) {
	if !viper.IsSet("plugins.mlflow") {
		return commonmlflow.MLflowPluginConfig{}, false, nil
	}
	raw := viper.Get("plugins.mlflow")
	data, err := json.Marshal(raw)
	if err != nil {
		return commonmlflow.MLflowPluginConfig{}, false, util.NewInvalidInputError("failed to marshal global plugins.mlflow config: %v", err)
	}
	var cfg commonmlflow.MLflowPluginConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return commonmlflow.MLflowPluginConfig{}, false, util.NewInvalidInputError("failed to parse global plugins.mlflow config: %v", err)
	}
	return cfg, true, nil
}

// GetServerSideNamespaceMLflowConfig reads an optional per-namespace MLflow
// override from the API server's plugins.mlflow.namespaces config.
func GetServerSideNamespaceMLflowConfig(namespace string) (*commonmlflow.MLflowPluginConfig, error) {
	if namespace == "" || !viper.IsSet("plugins.mlflow.namespaces") {
		return nil, nil
	}
	raw := viper.Get("plugins.mlflow.namespaces")
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, util.NewInvalidInputError("failed to marshal global plugins.mlflow.namespaces config: %v", err)
	}
	var namespaceCfgs map[string]json.RawMessage
	if err := json.Unmarshal(data, &namespaceCfgs); err != nil {
		return nil, util.NewInvalidInputError("failed to parse global plugins.mlflow.namespaces config: %v", err)
	}
	namespaceRaw, ok := namespaceCfgs[namespace]
	if !ok {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(namespaceRaw))
	decoder.DisallowUnknownFields()
	var cfg commonmlflow.MLflowPluginConfig
	if err := decoder.Decode(&cfg); err != nil {
		return nil, util.NewInvalidInputError("failed to parse global plugins.mlflow.namespaces[%q] config: %v", namespace, err)
	}
	var trailing json.RawMessage
	if trailingErr := decoder.Decode(&trailing); trailingErr != io.EOF {
		if trailingErr == nil {
			trailingErr = fmt.Errorf("unexpected trailing JSON content")
		}
		return nil, util.NewInvalidInputError("failed to parse global plugins.mlflow.namespaces[%q] config: %v", namespace, trailingErr)
	}
	return &cfg, nil
}

// GetLauncherNamespaceMLflowConfig reads the namespace-level MLflow launcher
// fragment from the kfp-launcher ConfigMap. Returns nil (no error) when the
// ConfigMap or key is absent.
func GetLauncherNamespaceMLflowConfig(namespace string, launcherNamespaceCfgOverride string) (*LauncherNamespaceMLflowConfig, error) {
	if launcherNamespaceCfgOverride == "" {
		return nil, nil
	}

	decoder := json.NewDecoder(bytes.NewReader([]byte(launcherNamespaceCfgOverride)))
	decoder.DisallowUnknownFields()
	var cfg LauncherNamespaceMLflowConfig
	if err := decoder.Decode(&cfg); err != nil {
		return nil, util.NewInternalServerError(err, "failed to parse MLflow config from key %q in configmap %q/%q", LauncherConfigKey, namespace, LauncherConfigMapName)
	}
	var trailing json.RawMessage
	if trailingErr := decoder.Decode(&trailing); trailingErr != io.EOF {
		if trailingErr == nil {
			trailingErr = fmt.Errorf("unexpected trailing JSON content")
		}
		return nil, util.NewInternalServerError(trailingErr, "failed to parse MLflow config from key %q in configmap %q/%q", LauncherConfigKey, namespace, LauncherConfigMapName)
	}
	return &cfg, nil
}

func applyLauncherNamespaceOverrides(base commonmlflow.MLflowPluginConfig, launcherCfg *LauncherNamespaceMLflowConfig) commonmlflow.MLflowPluginConfig {
	if launcherCfg == nil {
		return base
	}
	base.Settings = mergeLauncherNamespaceSettings(base.Settings, launcherCfg.Settings)
	return base
}

func mergeLauncherNamespaceSettings(base *commonmlflow.MLflowPluginSettings, overrides *LauncherNamespaceMLflowSettings) *commonmlflow.MLflowPluginSettings {
	if overrides == nil {
		return base
	}
	if base == nil {
		base = &commonmlflow.MLflowPluginSettings{}
	}
	merged := *base
	if overrides.ExperimentDescription != nil {
		merged.ExperimentDescription = overrides.ExperimentDescription
	}
	if overrides.DefaultExperimentName != "" {
		merged.DefaultExperimentName = overrides.DefaultExperimentName
	}
	if overrides.InjectUserEnvVars != nil {
		merged.InjectUserEnvVars = overrides.InjectUserEnvVars
	}
	if overrides.CredentialSecretRef != nil {
		merged.CredentialSecretRef = overrides.CredentialSecretRef
	}
	return &merged
}

// ResolveMLflowRequestConfig builds a merged and validated ResolvedConfig for the
// given namespace, combining global config, optional server-side namespace
// overrides, and the restricted launcher fragment.
func ResolveMLflowRequestConfig(ctx context.Context, clientSet kubernetes.Interface, launcherNamespaceConfig string, namespace string) (*ResolvedMLflowConfig, error) {
	globalCfg, hasGlobal, err := GetGlobalMLflowConfig()
	if err != nil {
		return nil, err
	}
	if !hasGlobal {
		return nil, nil
	}

	mergedCfg := globalCfg
	var launcherNamespaceCfg *LauncherNamespaceMLflowConfig
	if common.IsMultiUserMode() {
		serverSideNamespaceCfg, err := GetServerSideNamespaceMLflowConfig(namespace)
		if err != nil {
			return nil, err
		}
		mergedCfg = commonmlflow.MergePluginConfig(mergedCfg, serverSideNamespaceCfg)
		if mergedCfg.Settings != nil {
			// In multi-user mode, secret refs are namespace-owned: clear inherited refs so
			// only the namespace launcher ConfigMap can opt back in.
			mergedCfg.Settings.CredentialSecretRef = nil
		}

		launcherNamespaceCfg, err = GetLauncherNamespaceMLflowConfig(namespace, launcherNamespaceConfig)
		if err != nil {
			return nil, err
		}
		mergedCfg = applyLauncherNamespaceOverrides(mergedCfg, launcherNamespaceCfg)
	}
	if mergedCfg.Timeout == "" {
		mergedCfg.Timeout = DefaultTimeout
	}
	settings := ApplyMLflowSettingsDefaults(mergedCfg.Settings)
	mergedCfg.Settings = settings
	credentials, err := resolveConfiguredCredentials(ctx, clientSet, namespace, settings)
	if err != nil {
		return nil, err
	}
	return newResolvedMLflowConfig(&mergedCfg, credentials)
>>>>>>> upstream/master
}

// BuildMLflowRunRequestContext constructs a fully initialized RequestContext by
// performing API-server-specific validation and then delegating to the common
// BuildRequestContext.
<<<<<<< HEAD
func BuildMLflowRunRequestContext(namespace string, requestCfg *commonmlflow.MLflowPluginConfig) (*commonmlflow.RequestContext, error) {
	if requestCfg == nil {
=======
func BuildMLflowRunRequestContext(namespace string, requestCfg *ResolvedMLflowConfig) (*commonmlflow.RequestContext, error) {
	if requestCfg == nil || requestCfg.Config == nil {
>>>>>>> upstream/master
		return nil, util.NewInternalServerError(errors.New("MLflow config is nil"), "cannot build MLflow request context without a resolved config")
	}
	if requestCfg.Endpoint == "" {
		return nil, util.NewInvalidInputError("plugins.mlflow endpoint must be set")
	}
	settings := requestCfg.Config.Settings
	if settings == nil {
		return nil, util.NewInternalServerError(errors.New("MLflow plugin settings are nil"), "BuildMLflowRequestContext requires resolved settings")
	}
	if err := validateBaseURLs(settings); err != nil {
		return nil, err
	}
	workspacesEnabled := settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled
<<<<<<< HEAD
	return commonmlflow.BuildMLflowRequestContext(*requestCfg, namespace, workspacesEnabled)
=======
	return commonmlflow.BuildMLflowRequestContext(*requestCfg.Config, requestCfg.Credentials, namespace, workspacesEnabled)
}

// validateBaseURLs validates the kfpBaseURL and mlflowBaseURL fields in settings
// to prevent broken URL concatenation in hash-router URLs.
func validateBaseURLs(settings *commonmlflow.MLflowPluginSettings) error {
	if settings == nil {
		return nil
	}
	if settings.KFPBaseURL != "" {
		if err := commonmlflow.ValidateHTTPSBaseURL(settings.KFPBaseURL, "plugins.mlflow.settings.kfpBaseURL"); err != nil {
			return err
		}
	}
	if settings.MLflowBaseURL != "" {
		if err := commonmlflow.ValidateHTTPSBaseURL(settings.MLflowBaseURL, "plugins.mlflow.settings.mlflowBaseURL"); err != nil {
			return err
		}
	}
	return nil
>>>>>>> upstream/master
}

// ResolveMLflowPluginInput parses the plugins_input.mlflow JSON from a run model,
// and validates it against the MLflowPluginInput schema.
func ResolveMLflowPluginInput(pluginsInputString *string) (*MLflowPluginInput, error) {
	defaultInput := &MLflowPluginInput{ExperimentName: DefaultExperimentName}

	if pluginsInputString == nil || *pluginsInputString == "" {
<<<<<<< HEAD
		return defaultInput, nil
=======
		return &MLflowPluginInput{}, nil
>>>>>>> upstream/master
	}

	var pluginInputs apiserverPlugins.PluginsInputMap
	if err := json.Unmarshal([]byte(*pluginsInputString), &pluginInputs); err != nil {
		return nil, util.NewInvalidInputError("plugins_input must be a valid JSON object: %v", err)
	}
	var mlflowRaw json.RawMessage
	for key, value := range pluginInputs {
<<<<<<< HEAD
		if strings.EqualFold(key, PluginName) {
=======
		if key == PluginName {
>>>>>>> upstream/master
			mlflowRaw = value
			break
		}
	}
	if len(mlflowRaw) == 0 {
<<<<<<< HEAD
		return defaultInput, nil
=======
		return &MLflowPluginInput{}, nil
>>>>>>> upstream/master
	}

	decoder := json.NewDecoder(bytes.NewReader(mlflowRaw))
	decoder.DisallowUnknownFields()
	input := &MLflowPluginInput{}
	if err := decoder.Decode(input); err != nil {
		return nil, util.NewInvalidInputError("plugins_input.mlflow must follow schema {experiment_name?: string, experiment_id?: string, disabled?: bool}: %v", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, util.NewInvalidInputError("plugins_input.mlflow must be a single JSON object")
	}

	return input, nil
}

// ResolvePluginSettings parses and validates MLflow plugin settings from raw map, and applies default values where
// necessary.
func ResolvePluginSettings(rawSettings map[string]interface{}) *commonmlflow.MLflowPluginSettings {
	var settings commonmlflow.MLflowPluginSettings
	for key, value := range rawSettings {
		switch strings.ToLower(key) {
		case "workspacesenabled":
			settings.WorkspacesEnabled = asBoolPointer(key, value)
		case "experimentdescription":
			settings.ExperimentDescription = asStringPointer(key, value)
		case "defaultexperimentname":
			if s, ok := asString(key, value); ok {
				settings.DefaultExperimentName = s
			}
		case "kfpbaseurl":
			if s, ok := asString(key, value); ok {
				settings.KFPBaseURL = s
			}
		case "kfprunurlpathtemplate":
			if s, ok := asString(key, value); ok {
				settings.KFPRunURLPathTemplate = s
			}
		case "mlflowbaseurl":
			if s, ok := asString(key, value); ok {
				settings.MLflowBaseURL = s
			}
		case "mlflowuipathprefix":
			if s, ok := asString(key, value); ok {
				settings.MLflowUIPathPrefix = s
			}
		case "injectuserenvvars":
			settings.InjectUserEnvVars = asBoolPointer(key, value)
		default:
			glog.Warningf("unrecognized MLflow plugin setting: %s", key)
		}
	}
	return ApplySettingsDefaults(&settings)
}

func asBoolPointer(key string, value interface{}) *bool {
	switch v := value.(type) {
	case bool:
		return &v
	case *bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			glog.Errorf("failed to parse %s as bool from MLflow plugin settings: %v", key, err)
			return nil
		}
		return &parsed
	default:
		glog.Errorf("unexpected type %T for MLflow plugin setting %s", value, key)
		return nil
	}
}

func asStringPointer(key string, value interface{}) *string {
	if s, ok := asString(key, value); ok {
		return &s
	}
	return nil
}

func asString(key string, value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case *string:
		if v != nil {
			return *v, true
		}
		return "", false
	default:
		glog.Errorf("unexpected type %T for MLflow plugin setting %s", value, key)
		return "", false
	}
}

// GetGlobalMLflowConfig reads the global plugins.mlflow configuration
func GetGlobalMLflowConfig() (apiserverPlugins.PluginConfig, bool, error) {
	if !viper.IsSet("plugins.mlflow") {
		return apiserverPlugins.PluginConfig{}, false, nil
	}
	raw := viper.Get("plugins.mlflow")
	data, err := json.Marshal(raw)
	if err != nil {
		return apiserverPlugins.PluginConfig{}, false, util.NewInvalidInputError("failed to marshal global plugins.mlflow config: %v", err)
	}
	var cfg apiserverPlugins.PluginConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return apiserverPlugins.PluginConfig{}, false, util.NewInvalidInputError("failed to parse global plugins.mlflow config: %v", err)
	}
	return cfg, true, nil
}

// EnsureExperimentExists looks up the MLflow experiment by ID or name, and creates it
// if it does not already exist.
func EnsureExperimentExists(ctx context.Context, requestCtx *commonmlflow.RequestContext, experimentID, experimentName string, description *string) (*Experiment, error) {
	if requestCtx == nil || requestCtx.Client == nil {
		return nil, util.NewInvalidInputError("MLflow request context is required")
	}
	if experimentID != "" {
		exp, err := requestCtx.Client.GetExperiment(ctx, experimentID)
		if err != nil {
			return nil, fmt.Errorf("experiment ID %q not found in MLflow: %w", experimentID, err)
		}
		return &Experiment{ID: exp.ID, Name: exp.Name}, nil
	}
	existing, err := requestCtx.Client.GetExperimentByName(ctx, experimentName)
	if err == nil {
		return &Experiment{ID: existing.ID, Name: existing.Name}, nil
	}
	if !commonmlflow.IsNotFoundError(err) {
		return nil, err
	}
	return CreateExperiment(ctx, requestCtx, experimentName, description)
}

// CreateExperiment creates an MLflow experiment and handles the race condition
// where another request may have created the same experiment concurrently.
func CreateExperiment(ctx context.Context, requestCtx *commonmlflow.RequestContext, experimentName string, description *string) (*Experiment, error) {
	createdID, createErr := requestCtx.Client.CreateExperiment(ctx, experimentName, description)
	if createErr == nil {
		return &Experiment{ID: createdID, Name: experimentName}, nil
	}
	if commonmlflow.IsAlreadyExistsError(createErr) {
		// Race-safe fallback: another request created it between get-by-name and create.
		existing, err := requestCtx.Client.GetExperimentByName(ctx, experimentName)
		if err != nil {
			return nil, err
		}
		return &Experiment{ID: existing.ID, Name: existing.Name}, nil
	}
	return nil, createErr
}

// BuildKFPRunURL builds a link from kfpBaseURL to the pipeline run details page.
func BuildKFPRunURL(runID, namespace, kfpBaseURL, pathTemplate string) string {
	if runID == "" || kfpBaseURL == "" {
		glog.V(4).Infof(
			"BuildKFPRunURL returned empty URL due to missing input(s): runID_empty=%t kfpBaseURL_empty=%t",
			runID == "",
			kfpBaseURL == "",
		)
		return ""
	}
	pathTemplate = strings.TrimSpace(pathTemplate)
	if pathTemplate == "" {
		base := strings.TrimRight(kfpBaseURL, "/")
		return fmt.Sprintf("%s/#/runs/details/%s", base, url.PathEscape(runID))
	}
	if namespace == "" && strings.Contains(pathTemplate, "{namespace}") {
		glog.V(4).Infof("BuildKFPRunURL returned empty URL: namespace required when template contains {namespace}")
		return ""
	}
	base := strings.TrimRight(kfpBaseURL, "/")
	rendered := strings.ReplaceAll(pathTemplate, "{run_id}", url.PathEscape(runID))
	rendered = strings.ReplaceAll(rendered, "{namespace}", url.PathEscape(namespace))
	if !strings.HasPrefix(rendered, "/") && !strings.HasPrefix(rendered, "#") {
		rendered = "/" + rendered
	}
	return base + rendered
}

// BuildKFPTags builds MLflow tags containing KFP metadata for a pipeline run.
func BuildKFPTags(run *apiserverPlugins.PendingRun, kfpBaseURL, kfpRunURLPathTemplate string) []commonmlflow.Tag {
	if run == nil {
		return nil
	}
	tags := []commonmlflow.Tag{
		{Key: TagKFPRunID, Value: run.RunID},
		{Key: TagKFPRunURL, Value: BuildKFPRunURL(run.RunID, run.Namespace, kfpBaseURL, kfpRunURLPathTemplate)},
	}
	if run.PipelineID != "" {
		tags = append(tags, commonmlflow.Tag{Key: TagKFPPipelineID, Value: run.PipelineID})
	}
	if run.PipelineVersionID != "" {
		tags = append(tags, commonmlflow.Tag{Key: TagKFPPipelineVersionID, Value: run.PipelineVersionID})
	}
	return tags
}

// ApplySettingsDefaults applies default values to a parsed MLflowPluginSettings.
func ApplySettingsDefaults(settings *commonmlflow.MLflowPluginSettings) *commonmlflow.MLflowPluginSettings {
	if settings == nil {
		settings = &commonmlflow.MLflowPluginSettings{}
	}
	if settings.WorkspacesEnabled == nil {
		defaultEnabled := true
		settings.WorkspacesEnabled = &defaultEnabled
	}
	if settings.DefaultExperimentName == "" {
		settings.DefaultExperimentName = DefaultExperimentName
	}
	if settings.ExperimentDescription == nil {
		d := DefaultExperimentDescription
		settings.ExperimentDescription = &d
	}
	return settings
}

// SelectMLflowExperiment chooses the selector used for MLflow experiment resolution.
// Priority: user-provided experiment_id > user-provided experiment_name >
// admin-configured defaultExperimentName > hardcoded "KFP-Default".
func SelectMLflowExperiment(input *MLflowPluginInput, settings *commonmlflow.MLflowPluginSettings) (experimentID string, experimentName string) {
	if input != nil {
		if input.ExperimentID != "" {
			return input.ExperimentID, ""
		}
		if input.ExperimentName != "" {
			return "", input.ExperimentName
		}
	}
	if settings != nil && settings.DefaultExperimentName != "" {
		return "", settings.DefaultExperimentName
	}
	return "", DefaultExperimentName
}

<<<<<<< HEAD
=======
func resolveConfiguredCredentials(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	settings *commonmlflow.MLflowPluginSettings,
) (commonmlflow.MLflowCredentials, error) {
	if settings == nil {
		return commonmlflow.MLflowCredentials{}, util.NewInternalServerError(
			fmt.Errorf("settings are nil"),
			"MLflow settings must be provided when resolving credentials",
		)
	}
	switch settings.AuthType {
	case commonmlflow.AuthTypeKubernetes:
		return commonmlflow.ResolveMLflowCredentials()
	case commonmlflow.AuthTypeBearer:
		if settings.CredentialSecretRef == nil {
			return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
				"plugins.mlflow.settings.credentialSecretRef is required for authType %q",
				commonmlflow.AuthTypeBearer,
			)
		}
		return resolveBearerSecretCredentials(ctx, clientSet, namespace, settings.CredentialSecretRef)
	case commonmlflow.AuthTypeBasicAuth:
		if settings.CredentialSecretRef == nil {
			return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
				"plugins.mlflow.settings.credentialSecretRef is required for authType %q",
				commonmlflow.AuthTypeBasicAuth,
			)
		}
		return resolveBasicAuthSecretCredentials(ctx, clientSet, namespace, settings.CredentialSecretRef)
	case commonmlflow.AuthTypeNone:
		return commonmlflow.MLflowCredentials{AuthType: commonmlflow.AuthTypeNone}, nil
	default:
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
			"unsupported plugins.mlflow.settings.authType %q",
			settings.AuthType,
		)
	}
}

func resolveBearerSecretCredentials(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	ref *commonplugins.CredentialSecretRef,
) (commonmlflow.MLflowCredentials, error) {
	if ref == nil {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError("MLflow bearer auth requires credentialSecretRef")
	}
	if ref.TokenKey == "" {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
			"plugins.mlflow.settings.credentialSecretRef.tokenKey is required for authType %q",
			commonmlflow.AuthTypeBearer,
		)
	}
	secret, err := getMLflowCredentialSecret(ctx, clientSet, namespace)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	token, err := readRequiredSecretKey(secret, namespace, ref.TokenKey)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	return commonmlflow.MLflowCredentials{
		AuthType:    commonmlflow.AuthTypeBearer,
		BearerToken: token,
	}, nil
}

func resolveBasicAuthSecretCredentials(
	ctx context.Context,
	clientSet kubernetes.Interface,
	namespace string,
	ref *commonplugins.CredentialSecretRef,
) (commonmlflow.MLflowCredentials, error) {
	if ref == nil {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError("MLflow basic auth requires credentialSecretRef")
	}
	if ref.UsernameKey == "" {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
			"plugins.mlflow.settings.credentialSecretRef.usernameKey is required for authType %q",
			commonmlflow.AuthTypeBasicAuth,
		)
	}
	if ref.PasswordKey == "" {
		return commonmlflow.MLflowCredentials{}, util.NewInvalidInputError(
			"plugins.mlflow.settings.credentialSecretRef.passwordKey is required for authType %q",
			commonmlflow.AuthTypeBasicAuth,
		)
	}
	secret, err := getMLflowCredentialSecret(ctx, clientSet, namespace)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	username, err := readRequiredSecretKey(secret, namespace, ref.UsernameKey)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	password, err := readRequiredSecretKey(secret, namespace, ref.PasswordKey)
	if err != nil {
		return commonmlflow.MLflowCredentials{}, err
	}
	return commonmlflow.MLflowCredentials{
		AuthType: commonmlflow.AuthTypeBasicAuth,
		Username: username,
		Password: password,
	}, nil
}

// getMLflowCredentialSecret reads the fixed MLflow credentials Secret from the
// given namespace.
func getMLflowCredentialSecret(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*corev1.Secret, error) {
	if clientSet == nil {
		return nil, util.NewInternalServerError(
			fmt.Errorf("clientSet is nil"),
			"Kubernetes clientset must be provided when reading MLflow credentials secret",
		)
	}
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, commonmlflow.CredentialSecretName, v1.GetOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(
			err,
			"failed to read MLflow credentials secret %q in namespace %q",
			commonmlflow.CredentialSecretName,
			namespace,
		)
	}
	return secret, nil
}

// readRequiredSecretKey returns the trimmed value for key from secret, returning
// an error if the key is missing or resolves to an empty value.
func readRequiredSecretKey(secret *corev1.Secret, namespace, key string) (string, error) {
	valueBytes, ok := secret.Data[key]
	if !ok {
		return "", util.NewInvalidInputError(
			"secret %q in namespace %q does not contain key %q",
			commonmlflow.CredentialSecretName,
			namespace,
			key,
		)
	}
	value := strings.TrimSpace(string(valueBytes))
	if value == "" {
		return "", util.NewInvalidInputError(
			"secret %q in namespace %q has an empty value for key %q",
			commonmlflow.CredentialSecretName,
			namespace,
			key,
		)
	}
	return value, nil
}

// InjectMLflowRuntimeEnv sets KFP_MLFLOW_CONFIG on driver and launcher
// containers.
func InjectMLflowRuntimeEnv(executionSpec util.ExecutionSpec, envVars []corev1.EnvVar) error {
	if len(envVars) == 0 || executionSpec == nil {
		return nil
	}
	return executionSpec.UpsertRuntimeEnvVars(envVars,
		util.ExecutionRuntimeRoleDriver,
		util.ExecutionRuntimeRoleLauncher,
	)
}

>>>>>>> upstream/master
// ToMLflowTerminalStatus converts a KFP RuntimeState string to an MLflow
// terminal status.
func ToMLflowTerminalStatus(stateV2 string) string {
	switch stateV2 {
	case "SUCCEEDED":
		return "FINISHED"
	case "CANCELED", "CANCELING":
		return "KILLED"
	default:
		return "FAILED"
	}
}

<<<<<<< HEAD
// mlflowTrackingUIMountBase resolves the UI link prefix for MLflow Tracking UI links.
func mlflowTrackingUIMountBase(requestCtx *commonmlflow.RequestContext, settings *commonmlflow.MLflowPluginSettings) string {
	if settings != nil {
		if b := strings.TrimSpace(settings.MLflowBaseURL); b != "" {
			return strings.TrimRight(b, "/")
		}
	}
	if requestCtx != nil && requestCtx.BaseURL != nil {
		return strings.TrimRight(requestCtx.BaseURL.String(), "/")
	}
	return ""
}

func normalizeMlflowUIPathPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimRight(prefix, "/")
}

// BuildRunURL returns the MLflow Tracking UI URL for a run.
func BuildRunURL(requestCtx *commonmlflow.RequestContext, experimentID, runID string, settings *commonmlflow.MLflowPluginSettings) string {
	if experimentID == "" || runID == "" {
		glog.V(4).Infof(
			"BuildRunURL returned empty URL due to missing input(s): experimentID_empty=%t runID_empty=%t",
			experimentID == "",
			runID == "",
		)
		return ""
	}
	trackingUIBase := mlflowTrackingUIMountBase(requestCtx, settings)
	if trackingUIBase == "" {
		glog.V(4).Infof(
			"BuildRunURL returned empty URL: no mlflowBaseURL and requestCtx.BaseURL is unavailable",
		)
		return ""
	}
	uiPathPrefix := ""
	if settings != nil {
		uiPathPrefix = normalizeMlflowUIPathPrefix(settings.MLflowUIPathPrefix)
	}

	trackingMlflowRunPath := fmt.Sprintf(
		"/experiments/%s/runs/%s",
		url.PathEscape(experimentID),
		url.PathEscape(runID),
	)
	if requestCtx != nil && requestCtx.WorkspacesEnabled && requestCtx.Workspace != "" {
		trackingMlflowRunPath = fmt.Sprintf("%s?workspace=%s", trackingMlflowRunPath, url.QueryEscape(strings.ToLower(requestCtx.Workspace)))
	}
	return trackingUIBase + uiPathPrefix + "/#" + trackingMlflowRunPath
}

func SuccessfulPluginOutput(experimentID, experimentName, runID, runURL, endpoint string) *apiv2beta1.PluginOutput {
	return buildPluginOutput(experimentID, experimentName, runID, runURL, endpoint, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
}

func FailedPluginOutput(experimentID, experimentName, runID, runURL, endpoint, stateMessage string) *apiv2beta1.PluginOutput {
	return buildPluginOutput(experimentID, experimentName, runID, runURL, endpoint, apiv2beta1.PluginState_PLUGIN_FAILED, stateMessage)
}

// maxSearchPages caps SearchRuns pagination to prevent infinite loops.
const maxSearchPages = 100

// maxNestingDepth caps recursive nested run traversal.
const maxNestingDepth = 4

func SyncParentAndNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID string, mode apiserverPlugins.RunSyncMode, terminalStatus string, endTimeMs *int64) []string {
	if requestCtx == nil || requestCtx.Client == nil {
		return []string{"MLflow request context is required"}
	}
	if parentRunID == "" {
		return []string{"MLflow parent run_id is required"}
	}
	targetStatus := terminalStatus
	parentAction := "update parent run status"
	switch mode {
	case apiserverPlugins.RunSyncModeRetry:
		targetStatus = "RUNNING"
		parentAction = "reopen parent run"
	case apiserverPlugins.RunSyncModeTerminal:
		// keep caller-provided terminal status
	default:
		return []string{fmt.Sprintf("unsupported MLflow run sync mode %q", mode)}
	}
	var syncErrors []string
	if err := requestCtx.Client.UpdateRun(ctx, parentRunID, targetStatus, endTimeMs); err != nil {
		syncErrors = append(syncErrors, fmt.Sprintf("failed to %s: %v", parentAction, err))
	}
	if experimentID == "" {
		return syncErrors
	}
	// Recursively update all nested runs
	nestedErrors := syncNestedRuns(ctx, requestCtx, parentRunID, experimentID, mode, targetStatus, endTimeMs, 0)
	syncErrors = append(syncErrors, nestedErrors...)
	return syncErrors
}

// syncNestedRuns searches for MLflow runs tagged with the given parentRunID and
// updates their status. It recurses into each found run to handle deeper nesting
// (e.g., parent → loop nested run → iteration nested run).
func syncNestedRuns(ctx context.Context, requestCtx *commonmlflow.RequestContext, parentRunID, experimentID string, mode apiserverPlugins.RunSyncMode, targetStatus string, endTimeMs *int64, depth int) []string {
	if depth >= maxNestingDepth {
		return []string{fmt.Sprintf("max nesting depth (%d) reached when syncing children of run %s", maxNestingDepth, parentRunID)}
	}
	action := "close nested run"
	if mode == apiserverPlugins.RunSyncModeRetry {
		action = "reopen nested run"
	}
	var syncErrors []string
	filter := fmt.Sprintf(`tags.%q = '%s'`, commonmlflow.TagNestedRunParentRunID, parentRunID)
	pageToken := ""
	for page := 0; page < maxSearchPages; page++ {
		searchResp, err := requestCtx.Client.SearchRuns(ctx, []string{experimentID}, filter, 1000, pageToken)
		if err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("failed to search nested runs of %s: %v", parentRunID, err))
			break
		}
		for _, runPayload := range searchResp.Runs {
			mlflowRun := &searchRunPayload{}
			if err := json.Unmarshal(runPayload, mlflowRun); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("failed to decode nested run payload: %v", err))
				continue
			}
			nestedRunID := mlflowRun.Info.RunID
			if nestedRunID == "" {
				nestedRunID = mlflowRun.Info.RunUUID
			}
			if nestedRunID == "" || nestedRunID == parentRunID || !shouldSyncNestedRun(mode, mlflowRun.Info.Status) {
				continue
			}
			childErrors := syncNestedRuns(ctx, requestCtx, nestedRunID, experimentID, mode, targetStatus, endTimeMs, depth+1)
			syncErrors = append(syncErrors, childErrors...)
			if err := requestCtx.Client.UpdateRun(ctx, nestedRunID, targetStatus, endTimeMs); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("failed to %s %s: %v", action, nestedRunID, err))
			}
		}
		if searchResp.NextPageToken == "" {
			break
		}
		pageToken = searchResp.NextPageToken
	}
	return syncErrors
}

func buildPluginOutput(experimentID, experimentName, runID, runURL, endpoint string, state apiv2beta1.PluginState, stateMessage string) *apiv2beta1.PluginOutput {
	entries := map[string]*apiv2beta1.MetadataValue{}
	if experimentName != "" {
		entries[apiserverPlugins.EntryExperimentName] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(experimentName)}
	}
	if experimentID != "" {
		entries[apiserverPlugins.EntryExperimentID] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(experimentID)}
	}
	if runID != "" {
		entries[apiserverPlugins.EntryRootRunID] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(runID)}
	}
	if runURL != "" {
		entries[apiserverPlugins.EntryRunURL] = &apiv2beta1.MetadataValue{
			Value:      structpb.NewStringValue(runURL),
			RenderType: apiv2beta1.MetadataValue_URL.Enum(),
		}
	}
	if endpoint != "" {
		entries[apiserverPlugins.EntryEndpoint] = &apiv2beta1.MetadataValue{Value: structpb.NewStringValue(endpoint)}
	}
	return &apiv2beta1.PluginOutput{
		Entries:      entries,
		State:        state,
		StateMessage: stateMessage,
	}
}

func shouldSyncNestedRun(mode apiserverPlugins.RunSyncMode, status string) bool {
	upperStatus := strings.ToUpper(status)
	switch mode {
	case apiserverPlugins.RunSyncModeTerminal:
		return upperStatus != "FINISHED" && upperStatus != "FAILED" && upperStatus != "KILLED"
	case apiserverPlugins.RunSyncModeRetry:
		return upperStatus == "FAILED" || upperStatus == "KILLED"
	default:
		return false
	}
}

type searchRunPayload struct {
	Info struct {
		RunID   string `json:"run_id"`
		RunUUID string `json:"run_uuid"`
		Status  string `json:"status"`
	} `json:"info"`
=======
// maxSequentialRetriedCallsPerOperation is the number of idempotent, individually
// retried MLflow calls the longest plugin operation performs in sequence.
// OnBeforeRunCreation looks up the experiment (get-by-name), then creates the
// experiment, then creates the parent run; all share one context, so the budget
// must cover each retrying independently rather than being consumed by an
// earlier call.
const maxSequentialRetriedCallsPerOperation = 3

// mlflowOperationBudget is the overall context budget for an MLflow plugin
// operation. It is sized so every sequential idempotent call in the operation
// gets its full commonmlflow.MaxIdempotentAttempts of per-call timeouts, rather
// than an earlier call exhausting the budget and cutting a later call's retries
// short. It is only fully consumed under sustained failure; the common path
// returns as soon as the calls succeed.
func mlflowOperationBudget(resolvedTimeout time.Duration) time.Duration {
	return resolvedTimeout * time.Duration(commonmlflow.MaxIdempotentAttempts*maxSequentialRetriedCallsPerOperation)
>>>>>>> upstream/master
}
