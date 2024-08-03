k6 run --vus=100 --iterations=100000 place_order.js

k6 run --vus=500 --duration 10s vus.js

k6 run --vus=1 --iterations=1 vus.js

curl -i --request POST '<http://localhost:80/place_order>'

curl -o default.pgo 'http://localhost:8001/debug/pprof/profile?seconds=60'


k6 run qps.js
k6 run vus.js


go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkStringBuilder$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkBytesBuffer$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkByteBufferPool$ http-benchmark/pkg/gateway -v



go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkEscapeJSONStringBuilder$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkEscapeJSONBytePool$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkEscapeJSON1$ http-benchmark/pkg/gateway -v

go test -benchmem -run=^$ -bench ^BenchmarkStrHasPrefix$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -bench ^BenchmarkBytHasPrefix$ http-benchmark/pkg/gateway -v


netstat -ant | grep 8001 | grep ESTABLISHED| wc -l


go tool pprof -http=0.0.0.0:4231 cpu.prof


curl 'http://localhost:9091/metrics'


curl --insecure --http2 --request POST 'https://localhost:8001/spot/orders'
curl --insecure -I --http1.1 --request POST 'https://bifrost.io:443/spot/orders'

curl --insecure --request POST 'https://bifrost.io:443/spot/orders'

curl -v --http2 --request POST 'http://localhost:8001/spot/orders'
curl -v --http2-prior-knowledge --request POST 'https://localhost:8001/spot/orders'
curl -v --http1.1 --request POST 'http://localhost:8001/spot/orders'
