package mlflow

import (
	"encoding/json"
	"fmt"
	"strings"

	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/spf13/viper"
)

const (
	mlflowRunID     = "MLFLOW_RUN_ID"
	kfpMLflowConfig = "KFP_MLFLOW_CONFIG"
)

func GetStringConfig(configName string) string {
	return viper.GetString(configName)
}

func GetMLflowRunID() string {
	return GetStringConfig(mlflowRunID)
}

// ParseKfpMLflowRuntimeConfig parses the KFP_MLFLOW_CONFIG environment variable into an MLflowRuntimeConfig struct.
// Returns an error if the variable is not set, malformed, or contains an unsupported auth type.
func ParseKfpMLflowRuntimeConfig() (*commonmlflow.MLflowRuntimeConfig, error) {
	var cfg commonmlflow.MLflowRuntimeConfig
	runtimeCfg := GetStringConfig(kfpMLflowConfig)
	if runtimeCfg == "" {
		return nil, fmt.Errorf("KFP_MLFLOW_CONFIG env var not set")
	}
	if err := json.Unmarshal([]byte(runtimeCfg), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KFP_MLFLOW_CONFIG: %v", err)
	}
	if cfg.Workspace != "" {
		cfg.WorkspacesEnabled = true
	}
	var missingFields []string
	if cfg.Endpoint == "" {
		missingFields = append(missingFields, "Endpoint")
	}
	if cfg.ParentRunID == "" {
		missingFields = append(missingFields, "ParentRunID")
	}
	if cfg.ExperimentID == "" {
		missingFields = append(missingFields, "ExperimentID")
	}
	if cfg.AuthType == "" {
		missingFields = append(missingFields, "AuthType")
	}
	if cfg.Timeout == "" {
		missingFields = append(missingFields, "Timeout")
	}
	if len(missingFields) > 0 {
		return nil, fmt.Errorf("missing one or more of the following required fields in KFP_MLFLOW_CONFIG: %s", strings.Join(missingFields, ", "))
	}
	if cfg.AuthType != "kubernetes" {
		return nil, fmt.Errorf("unsupported auth type: %s", cfg.AuthType)
	}
	// Disabling TLS verification is not supported in the driver/launcher
	// to prevent CWE-295 (improper certificate validation).
	if cfg.InsecureSkipVerify {
		return nil, fmt.Errorf("insecureSkipVerify is not supported")
	}
	// Preserve the CABundlePath from the runtime config (propagated by the
	// API server) so the driver/launcher can verify certificates signed by
	// an internal CA (e.g., cert-manager). InsecureSkipVerify is always
	// forced to false.
	caBundlePath := ""
	if cfg.TLS != nil {
		caBundlePath = cfg.TLS.CABundlePath
	}
	cfg.TLS = &commonplugins.TLSConfig{
		InsecureSkipVerify: false,
		CABundlePath:       caBundlePath,
	}
	return &cfg, nil
}

// IsEnabled reports whether the env var for the MLflow runtime config is present,
// indicating the driver/launcher has opted in to MLflow integration.
func IsEnabled() bool {
	return viper.IsSet(commonmlflow.EnvMLflowConfig)
}

// BuildMLflowTaskRequestContext constructs a fully initialized RequestContext
// by delegating to the common BuildMLflowRequestContext with task-specific parameters.
func BuildMLflowTaskRequestContext(runtimeCfg commonmlflow.MLflowRuntimeConfig) (*commonmlflow.RequestContext, error) {
	mlflowPluginSettings := &commonmlflow.MLflowPluginSettings{
		WorkspacesEnabled: &runtimeCfg.WorkspacesEnabled,
		KFPBaseURL:        runtimeCfg.Endpoint,
		InjectUserEnvVars: &runtimeCfg.InjectUserEnvVars,
	}

	pluginCfg := commonmlflow.MLflowPluginConfig{
		Endpoint: runtimeCfg.Endpoint,
		Timeout:  runtimeCfg.Timeout,
		TLS:      runtimeCfg.TLS,
		Settings: mlflowPluginSettings,
	}
	return commonmlflow.BuildMLflowRequestContext(pluginCfg, runtimeCfg.Workspace, runtimeCfg.WorkspacesEnabled)
}

// ExecutionStateToMLflowTerminalStatus converts a string representing an MLMD Execution_State to an MLflow
// terminal status.
func ExecutionStateToMLflowTerminalStatus(state string) string {
	switch state {
	case "COMPLETE", "CACHED":
		return "FINISHED"
	case "CANCELED":
		return "KILLED"
	default:
		return "FAILED"
	}
}
