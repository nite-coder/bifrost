protoc -I=. --go_out=plugins=grpc:. --go_opt=paths=source_relative ./server/testserver/grpc/proto/*.proto

curl http://localhost:8999/metrics

curl --http2 --insecure -v https://localhost:8443/spot/orders

grpcurl -plaintext -vv localhost:8003 helloworld.Greeter/SayHello

grpcurl -plaintext localhost:8003 list

/grpc.reflection.v1.ServerReflection/ServerReflectionInfo

grpcurl -v -proto hello_world.proto -d '{"name": "jason"}' -plaintext localhost:8001 helloworld.Greeter/SayHello

timeout 30 tcpdump -i any host localhost and port 8001 -w ./bifrost.pcap

ss -tulpn | grep :8001

ss -plnt

-- 查找多少用戶端 tcp 連線到 server port 8001
ss -tn state established '( dport = :8001 )' | wc -l
