import http from 'k6/http';


export const options = {
    scenarios: {
        constant_request_rate: {
            executor: 'constant-arrival-rate',
            rate: 10000,
            timeUnit: '1s',
            duration: '10s',
            preAllocatedVUs: 100,
            maxVUs: 100,
        },
    },
    summaryTrendStats: ['avg', 'min', 'max', 'p(50)', 'p(75)', 'p(95)', 'p(99)', 'count'],
}

export default function () {
    const url = 'http://localhost:8001/spot/orders?mode=b';

    const payload = JSON.stringify({
        "market": "BTC_USDT",
        "base": "BTC",
        "quote": "USDT",
        "type": "limit",
        "price": "25000",
        "size": "0.0001",
        "side": "sell",
        "user_id": 1,
        "text": "你好世界"
    });

    const params = {
        headers: {
            'Connection': 'Keep-Alive',
            'Content-Type': 'application/json',
            'X-User-ID': '1'
        },
        timeout: '1s'
    };

    http.post(url, payload, params);
}
