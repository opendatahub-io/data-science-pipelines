module github.com/kubeflow/pipelines/kubernetes_platform

go 1.26.0

toolchain go1.26.3

require (
	github.com/kubeflow/pipelines/api v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.12
)

require google.golang.org/genproto/googleapis/rpc v0.0.0-20261019180522-ce908d29299b // indirect

replace github.com/kubeflow/pipelines/api => ../api
