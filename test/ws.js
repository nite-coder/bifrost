import ws from 'k6/ws';
import { check } from 'k6';

export const options = {
    scenarios: {
        websocket_test: {
            executor: 'constant-vus',
            vus: 500,
            duration: '10s',
            gracefulStop: '0s',
        },
    },
    insecureSkipTLSVerify: true,
    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(75)', 'p(95)', 'p(99)', 'count'],
};

export default function () {
    const url = 'ws://127.0.0.1:8001/websocket';
    const params = { tags: { my_tag: 'hello' } };

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

    const res = ws.connect(url, params, function (socket) {
        socket.on('open', function () {
            console.log('connected');
            socket.send(payload); // send a message
            socket.setInterval(function () {
                socket.send(payload);
            }, 100);
        });

        socket.on('message', function (data) {
            //console.log(isValidJSON(data));
            //console.log('Message received: ', data);
        });

        socket.on('close', () => console.log('disconnected'));

        socket.on('error', function (e) {
            if (e.error() != 'websocket: close sent') {
                console.log('An unexpected error occured: ', e.error());
            }
        });
    });

    check(res, { 'status is 101': (r) => r && r.status === 101 });
}


function isValidJSON(str) {
    try {
        JSON.parse(str);
        return true;
    } catch (e) {
        return false;
    }
}