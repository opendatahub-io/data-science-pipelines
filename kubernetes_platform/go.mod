module github.com/kubeflow/pipelines/kubernetes_platform

go 1.25.7

require (
	github.com/kubeflow/pipelines/api v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.11
)

require google.golang.org/genproto/googleapis/rpc v0.0.0-20260401001100-f93e5f3e9f0f // indirect

replace github.com/kubeflow/pipelines/api => ../api
