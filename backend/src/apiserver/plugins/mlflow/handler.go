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

// Package mlflow implements MLflow API server plugin handlers.
package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	corev1 "k8s.io/api/core/v1"
)

var _ apiserverPlugins.RunPluginHandler = (*Handler)(nil)

// Handler implements PluginHandler for the MLflow integration.
type Handler struct{}

<<<<<<< HEAD
// NewMLflowRunHandler creates a new MLflow plugin handler.
func NewMLflowRunHandler() *Handler {
	return &Handler{}
=======
	// RunStartEnvVars is populated by OnBeforeRunCreation with runtime env vars
	// for the driver and launcher.
	RunStartEnvVars []corev1.EnvVar
>>>>>>> upstream/master
}

// Name returns the name of the MLflow plugin handler.
func (h *Handler) Name() string {
	return "MLflow"
}

// GetGlobalPluginConfig re-reads the global plugin configuration from Viper so
// that runtime config changes (e.g. admin removing the MLflow config) are
// reflected without restarting the API server.
func (h *Handler) GetGlobalPluginConfig() (*apiserverPlugins.PluginConfig, error) {
	cfg, ok, err := GetGlobalMLflowConfig()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &cfg, nil
}

// OnBeforeRunCreation creates the MLflow experiment and parent run, tags it
// with KFP metadata, and returns runtime env vars for the driver and launcher.
func (h *Handler) OnBeforeRunCreation(ctx context.Context, run *apiserverPlugins.PendingRun, runCfg *apiserverPlugins.PluginConfig) (*apiv2beta1.PluginOutput, map[string]string, error) {
	if h == nil || run == nil || runCfg == nil {
		return nil, nil, nil
	}
<<<<<<< HEAD
	mlflowPluginInput, err := ResolveMLflowPluginInput(run.PluginsInput)
	if err != nil {
		return nil, nil, fmt.Errorf("MLflow run canceled due to error retrieving run-level plugin input: %s", err)
	}
	if mlflowPluginInput == nil || mlflowPluginInput.Disabled {
		return nil, nil, nil
	}

	endpoint := runCfg.Endpoint

	settings := ResolvePluginSettings(runCfg.Settings)

	experimentID, experimentName := SelectMLflowExperiment(mlflowPluginInput, settings)
	if experimentID != "" {
		glog.V(4).Infof("Resolved MLflow experiment selector for run creation: experiment_id=%q (create-by-name skipped)", experimentID)
	} else {
		glog.V(4).Infof("Resolved MLflow experiment selector for run creation: experiment_name=%q", experimentName)
	}

	resolvedCfg, err := ResolveMLflowPluginConfig(runCfg, settings)
	if err != nil {
		message := "MLflow config resolution failed; run creation will continue: " + err.Error()
		glog.Warningf("MLflow OnBeforeRunCreation failed for run %q (%s)", run.RunID, message)
		return FailedPluginOutput(experimentID, experimentName, "", "", "", message), nil, err
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(run.Namespace, resolvedCfg)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", endpoint, fmt.Sprintf("failed to build MLflow request context: %v", err)), nil, err
=======
	resolvedCfg := resolveHandlerConfig(config)
	if resolvedCfg == nil || resolvedCfg.Config == nil {
		return nil, nil
	}
	pluginConfig := resolvedCfg.Config

	experimentID, experimentName := SelectMLflowExperiment(h.input, pluginConfig.Settings)

	settings := pluginConfig.Settings
	if settings == nil {
		err := fmt.Errorf("resolved MLflow settings are missing")
		return FailedPluginOutput(experimentID, experimentName, "", "", err.Error()), err
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(ctx, h.namespace, resolvedCfg)
	if err != nil {
		return FailedPluginOutput(experimentID, experimentName, "", "", fmt.Sprintf("failed to build MLflow request context: %v", err)), err
>>>>>>> upstream/master
	}

	mlflowExperiment, err := EnsureExperimentExists(
		ctx,
		mlflowRequestCtx,
		experimentID,
		experimentName,
		settings.ExperimentDescription,
	)
	if err != nil {
<<<<<<< HEAD
		return FailedPluginOutput(experimentID, experimentName, "", "", endpoint, err.Error()), nil, err
=======
		return FailedPluginOutput(experimentID, experimentName, "", "", err.Error()), err
>>>>>>> upstream/master
	}

	tags := BuildKFPTags(run, settings.KFPBaseURL, settings.KFPRunURLPathTemplate)
	parentRunID, err := mlflowRequestCtx.Client.CreateRun(ctx, mlflowExperiment.ID, run.DisplayName, tags)
	if err != nil {
<<<<<<< HEAD
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, "", "", endpoint, err.Error()), nil, err
=======
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, "", "", err.Error()), err
>>>>>>> upstream/master
	}

	workspace := ""
	if settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled {
		workspace = run.Namespace
	}
	// TLS.InsecureSkipVerify is intentionally omitted: disabling TLS
	// verification is not supported in the driver/launcher (CWE-295).
	// TLS.CABundlePath IS propagated so that driver/launcher pods can
	// verify certificates signed by an internal CA (e.g., cert-manager).
	// The operator is responsible for ensuring this path is available in
	// driver/launcher pods (via platform spec volume mounts, trusted CA
	// injection, or similar mechanisms).
	var runtimeTLS *commonplugins.TLSConfig
	if runCfg.TLS != nil && runCfg.TLS.CABundlePath != "" {
		runtimeTLS = &commonplugins.TLSConfig{
			CABundlePath: runCfg.TLS.CABundlePath,
		}
	}
	mlflowRuntimeConfig := commonmlflow.MLflowRuntimeConfig{
<<<<<<< HEAD
		Endpoint:          mlflowRequestCtx.BaseURL.String(),
		Workspace:         workspace,
		WorkspacesEnabled: settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled,
		ParentRunID:       parentRunID,
		ExperimentID:      mlflowExperiment.ID,
		AuthType:          commonmlflow.AuthTypeKubernetes,
		Timeout:           resolvedCfg.Timeout,
		InjectUserEnvVars: settings.InjectUserEnvVars != nil && *settings.InjectUserEnvVars,
		TLS:               runtimeTLS,
	}
	mlflowConfigJSON, err := json.Marshal(mlflowRuntimeConfig)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, "", endpoint, fmt.Sprintf("failed to marshal MLflow runtime config: %v", err)), nil, err
	}

	runStartEnv := map[string]string{
		commonmlflow.EnvMLflowConfig: string(mlflowConfigJSON),
=======
		Endpoint:            mlflowRequestCtx.BaseURL.String(),
		Workspace:           workspace,
		WorkspacesEnabled:   settings.WorkspacesEnabled != nil && *settings.WorkspacesEnabled,
		ParentRunID:         parentRunID,
		ExperimentID:        mlflowExperiment.ID,
		AuthType:            settings.AuthType,
		CredentialSecretRef: runtimeCredentialSecretRef(settings),
		Timeout:             pluginConfig.Timeout,
		InsecureSkipVerify:  insecureSkipVerify,
		InjectUserEnvVars:   settings.InjectUserEnvVars != nil && *settings.InjectUserEnvVars,
	}
	mlflowConfigJSON, err := json.Marshal(mlflowRuntimeConfig)
	if err != nil {
		return FailedPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, "", fmt.Sprintf("failed to marshal MLflow runtime config: %v", err)), err
	}

	h.RunStartEnvVars = []corev1.EnvVar{{
		Name:  commonmlflow.EnvMLflowConfig,
		Value: string(mlflowConfigJSON),
	}}
	credentialEnvVars, err := commonmlflow.BuildCredentialEnvVars(settings.CredentialSecretRef, settings.AuthType)
	if err != nil {
		return FailedPluginOutput(
			mlflowExperiment.ID,
			mlflowExperiment.Name,
			parentRunID,
			"",
			fmt.Sprintf("failed to build MLflow credential env vars: %v", err),
		), err
>>>>>>> upstream/master
	}
	h.RunStartEnvVars = append(h.RunStartEnvVars, credentialEnvVars...)

	runURL := BuildRunURL(mlflowRequestCtx, mlflowExperiment.ID, parentRunID, settings)
<<<<<<< HEAD
	return SuccessfulPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, runURL, endpoint), runStartEnv, nil
}

// OnRunEnd marks the MLflow parent run and any active nested runs as
// complete/failed when the KFP run reaches a terminal state.
func (h *Handler) OnRunEnd(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg *apiserverPlugins.PluginConfig) error {
	if h == nil || run == nil || runCfg == nil {
		return nil
	}
	resolvedSettings := ResolvePluginSettings(runCfg.Settings)
	resolvedMLflowCfg, err := ResolveMLflowPluginConfig(runCfg, resolvedSettings)
	if err != nil {
		return err
	}
	return h.syncOnRunTerminal(ctx, run, resolvedMLflowCfg, run.Namespace)
}

// syncOnRunTerminal marks the MLflow parent and nested runs as complete/failed.
func (h *Handler) syncOnRunTerminal(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg *commonmlflow.MLflowPluginConfig, namespace string) error {
=======
	return SuccessfulPluginOutput(mlflowExperiment.ID, mlflowExperiment.Name, parentRunID, runURL), nil
}

// OnRunEnd marks the MLflow parent run and any active nested runs as
// complete/failed when the KFP run reaches a terminal state. The returned
// bool reports whether a failed sync is worth retrying: transient MLflow
// call failures request a retry, while permanent problems (missing parent
// run id, unavailable or invalid config) are recorded in the plugin output
// and must not block run finalization.
func (h *Handler) OnRunEnd(ctx context.Context, run *apiserverPlugins.PersistedRun, config interface{}) (bool, error) {
	if h == nil || run == nil {
		return false, nil
	}
	return h.syncOnRunTerminal(ctx, run, resolveHandlerConfig(config)), nil
}

// syncOnRunTerminal marks the MLflow parent and nested runs as complete/failed.
// It returns true when the sync failed transiently and should be retried.
func (h *Handler) syncOnRunTerminal(ctx context.Context, run *apiserverPlugins.PersistedRun, config *ResolvedConfig) bool {
>>>>>>> upstream/master
	endTimeMs := int64(0)
	endTimeRef := (*int64)(nil)
	if run.FinishedAt != nil {
		endTimeMs = run.FinishedAt.UnixMilli()
		endTimeRef = &endTimeMs
	}
	terminalStatus := ToMLflowTerminalStatus(run.State)
<<<<<<< HEAD
	h.syncMLflowRuns(ctx, run, runCfg, apiserverPlugins.RunSyncModeTerminal, terminalStatus, endTimeRef, "terminal", namespace)
	return nil
}

// HandleRetry reopens the MLflow parent run and any failed/killed nested runs.
func (h *Handler) HandleRetry(ctx context.Context, run *apiserverPlugins.PersistedRun, runCfg *apiserverPlugins.PluginConfig) error {
	if h == nil || run == nil || runCfg == nil {
		return fmt.Errorf("handler, run and runCfg must be non-nil")
	}

	resolvedSettings := ResolvePluginSettings(runCfg.Settings)
	resolvedMLflowCfg, err := ResolveMLflowPluginConfig(runCfg, resolvedSettings)
	if err != nil {
		return fmt.Errorf("failed to resolve MLflow plugin config: %v", err)
	}

	h.syncMLflowRuns(ctx, run, resolvedMLflowCfg, apiserverPlugins.RunSyncModeRetry, "", nil, "retry", run.Namespace)
	return nil
}

// syncMLflowRuns resolves the MLflow request context, syncs the parent and nested runs, and
// updates the plugin output state.
func (h *Handler) syncMLflowRuns(ctx context.Context, run *apiserverPlugins.PersistedRun, config *commonmlflow.MLflowPluginConfig, mode apiserverPlugins.RunSyncMode, terminalStatus string, endTimeRef *int64, label string, namespace string) {
	pluginOutput := run.PluginsOutput[h.Name()]
=======
	return h.syncMLflowRuns(ctx, run, config, RunSyncModeTerminal, terminalStatus, endTimeRef, "terminal")
}

// HandleRetry reopens the MLflow parent run and any failed/killed nested runs.
func (h *Handler) HandleRetry(ctx context.Context, run *apiserverPlugins.PersistedRun, config *ResolvedConfig) {
	h.syncMLflowRuns(ctx, run, config, RunSyncModeRetry, "", nil, "retry")
}

// syncMLflowRuns resolves the MLflow request context, syncs the parent and nested runs, and
// updates the plugin output state. The returned bool reports whether the failure is
// transient and worth retrying; permanent failures (missing parent run id, unavailable
// or invalid config, unresolvable credentials) return false so callers do not retry
// a sync that cannot succeed until an operator fixes the configuration.
func (h *Handler) syncMLflowRuns(ctx context.Context, run *apiserverPlugins.PersistedRun, config *ResolvedConfig, mode RunSyncMode, terminalStatus string, endTimeRef *int64, label string) bool {
	pluginOutput := run.PluginsOutput[PluginName]
>>>>>>> upstream/master
	if pluginOutput == nil {
		return false
	}

	parentRunID := apiserverPlugins.GetParentRunID(pluginOutput)
	experimentID := apiserverPlugins.GetStringEntry(pluginOutput, apiserverPlugins.EntryExperimentID)
	if parentRunID == "" {
		msg := fmt.Sprintf("MLflow %s sync skipped: missing parent root_run_id in plugins_output.mlflow", label)
		glog.Warning(msg)
<<<<<<< HEAD
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return
=======
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return false
>>>>>>> upstream/master
	}

	localConfig := cloneResolvedConfig(config)
	if localConfig == nil || localConfig.Config == nil {
		msg := fmt.Sprintf("MLflow %s sync failed: config unavailable", label)
		glog.Warning(msg)
<<<<<<< HEAD
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return
	}

	// Use the endpoint stored at run-start time so that in-flight runs
	// always talk to the MLflow server where their parent run was created,
	// even if the admin changes the endpoint while the run is in progress.
	storedEndpoint := apiserverPlugins.GetStringEntry(pluginOutput, apiserverPlugins.EntryEndpoint)
	if storedEndpoint != "" {
		config.Endpoint = storedEndpoint
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(namespace, config)
	if err != nil {
		msg := fmt.Sprintf("MLflow %s sync failed: %v", label, err)
		glog.Warning(msg)
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return
=======
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return false
	}
	if localConfig.Config.Settings == nil {
		msg := fmt.Sprintf("MLflow %s sync failed: resolved MLflow settings are missing", label)
		glog.Warning(msg)
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return false
	}

	mlflowRequestCtx, err := BuildMLflowRunRequestContext(ctx, h.namespace, localConfig)
	if err != nil {
		msg := fmt.Sprintf("MLflow %s sync failed: %v", label, err)
		glog.Warning(msg)
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		return false
>>>>>>> upstream/master
	}

	syncErrors := SyncParentAndNestedRuns(ctx, mlflowRequestCtx, parentRunID, experimentID, mode, terminalStatus, endTimeRef)
	if len(syncErrors) > 0 {
		msg := strings.Join(syncErrors, "; ")
		glog.Warningf("MLflow %s sync encountered errors for run %s: %s", label, run.RunID, msg)
<<<<<<< HEAD
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
	} else {
		apiserverPlugins.SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
=======
		SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_FAILED, msg)
		// The MLflow calls themselves failed (network, availability, or
		// server-side errors); a later retry can succeed.
		return true
	}
	SetPluginOutputState(pluginOutput, apiv2beta1.PluginState_PLUGIN_SUCCEEDED, "")
	return false
}

func resolveHandlerConfig(config interface{}) *ResolvedConfig {
	typedConfig, _ := config.(*ResolvedConfig)
	return typedConfig
}

func runtimeCredentialSecretRef(settings *commonmlflow.MLflowPluginSettings) *commonmlflow.CredentialSecretRef {
	if settings == nil || settings.CredentialSecretRef == nil {
		return nil
	}
	switch settings.AuthType {
	case commonmlflow.AuthTypeBearer, commonmlflow.AuthTypeBasicAuth:
		credentialSecretRef := *settings.CredentialSecretRef
		return &credentialSecretRef
	default:
		return nil
>>>>>>> upstream/master
	}
}

func cloneResolvedConfig(config *ResolvedConfig) *ResolvedConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	if config.Config != nil {
		configCopy := *config.Config
		cloned.Config = &configCopy
	}
	if config.Config != nil && config.Config.Settings != nil {
		settingsCopy := *config.Config.Settings
		if settingsCopy.WorkspacesEnabled != nil {
			workspacesEnabled := *settingsCopy.WorkspacesEnabled
			settingsCopy.WorkspacesEnabled = &workspacesEnabled
		}
		cloned.Config.Settings = &settingsCopy
	}
	return &cloned
}
