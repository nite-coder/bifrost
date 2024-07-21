protoc -I=. --go_out=plugins=grpc:. --go_opt=paths=source_relative ./server/testserver/grpc/proto/*.proto

grpcurl -plaintext -vv localhost:8003 helloworld.Greeter/SayHello

grpcurl -plaintext localhost:8003 list

/grpc.reflection.v1.ServerReflection/ServerReflectionInfo

grpcurl -v -proto hello_world.proto -d '{"name": "jason"}' -plaintext localhost:8001 helloworld.Greeter/SayHello
