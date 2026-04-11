module github.com/kubeflow/pipelines/api

go 1.25.7

require (
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d
	google.golang.org/protobuf v1.36.11
)

replace (
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.18
	golang.org/x/net => golang.org/x/net v0.33.0
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
)
