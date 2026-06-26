module github.com/kubeflow/pipelines/kubernetes_platform

go 1.26

toolchain go1.26.3

require (
	github.com/kubeflow/pipelines/api v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.11
)

require google.golang.org/genproto/googleapis/rpc v0.0.0-20260618152121-87f3d3e198d3 // indirect

replace github.com/kubeflow/pipelines/api => ../api
