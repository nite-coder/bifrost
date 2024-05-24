k6 run --vus=2048 --iterations=100000 place_order.js

k6 run --vus=2048 --duration 20s place_order.js

curl -i --request POST '<http://localhost:80/place_order>'
