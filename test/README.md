k6 run --vus=100 --iterations=100000 place_order.js

curl -i --request POST '<http://localhost:80/upstream>'
