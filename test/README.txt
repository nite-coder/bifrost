k6 run --vus=100 --iterations=100000 place_order.js

k6 run --vus=500 --duration 10s create_order.js

curl -i --request POST '<http://localhost:80/place_order>'

curl -o default.pgo 'http://localhost:8001/debug/pprof/profile?seconds=30'


k6 run place_order.js
k6 run create_order.js


go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkStringBuilder$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkBytesBuffer$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkByteBufferPool$ http-benchmark/pkg/gateway -v



go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkEscapeJSONStringBuilder$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkEscapeJSONBytePool$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -coverprofile=/tmp/vscode-go0xlfMt/go-code-cover -bench ^BenchmarkEscapeJSON1$ http-benchmark/pkg/gateway -v

go test -benchmem -run=^$ -bench ^BenchmarkEscapeJSON1$ http-benchmark/pkg/gateway -v
go test -benchmem -run=^$ -bench ^BenchmarkEscapeJSON4$ http-benchmark/pkg/gateway -v


netstat -ant | grep 8001 | grep ESTABLISHED| wc -l


go tool pprof -http=0.0.0.0:4231 cpu.prof