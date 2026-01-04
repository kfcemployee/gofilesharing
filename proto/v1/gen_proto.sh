#bin!bash

protoc -I . --go_out=./gen/registry --go_opt=paths=source_relative --go-grpc_out=./gen/registry --go-grpc_opt=paths=source_relative proto/v1/registry.proto