import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';
import exec from 'k6/execution';

const client = new grpc.Client();
client.load(['definitions'], './hello_world.proto');


export const options = {
    scenarios: {
        grpc_test: {
            executor: 'constant-vus',
            vus: 100,
            duration: '10s',
            gracefulStop: '0s',
        },
    },
    insecureSkipTLSVerify: true,
    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(75)', 'p(95)', 'p(99)', 'count'],
};


export default () => {

    if (exec.vu.iterationInScenario == 0) {
        client.connect('127.0.0.1:8001', {
            plaintext: true
        });
    }

    const data = { name: 'Bert' };

    const response = client.invoke('/helloworld.Greeter/SayHello', data);

    check(response, {
        'status is OK': (r) => r && r.status === grpc.StatusOK,
    });


    //client.close();
    //console.log(response.message);
};