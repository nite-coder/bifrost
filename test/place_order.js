import http from 'k6/http';


export default function () {
    const url = 'http://localhost:8001/place_order';

    const payload = JSON.stringify({
        "market": "BTC_USDT",
        "base": "BTC",
        "quote": "USDT",
        "type": "limit",
        "price": "25000",
        "size": "0.0001",
        "side": "sell",
        "user_id": 1
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'X-User-ID': '1'
        },
    };

    http.post(url, payload, params);
}