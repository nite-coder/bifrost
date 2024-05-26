k6 run --vus=2048 --iterations=100000 place_order.js

k6 run --vus=2048 --duration 60s place_order.js

curl -i --request POST '<http://localhost:80/place_order>'

curl -o default.pgo 'http://localhost:8001/debug/pprof/profile?seconds=30'


k6 run --vus=1 --iterations=1 place_order.js