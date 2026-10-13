import { check, sleep } from 'k6';
import ws from 'k6/ws';

export const options = {
    stages: [
        { duration: '30s', target: 1000 },
        { duration: '1m',  target: 5000 },
        { duration: '2m',  target: 5000 },
        { duration: '30s', target: 0 },
    ],
    thresholds: {
        'ws_connecting': ['p(95)<200', 'p(99)<500'],
        'ws_msgs_received': ['count>10000'],
    },
};

export default function () {
    const url = 'ws://localhost:8080/ws?token=' + __ENV.TEST_PASETO_TOKEN;
    const params = { headers: { Origin: 'https://app.wassup.local' } };

    const res = ws.connect(url, params, function (socket) {
        socket.on('open', () => {
            socket.setInterval(() => {
                socket.send(JSON.stringify({
                    type: 'HEARTBEAT',
                    timestamp: Date.now()
                }));
            }, 15000);

            socket.send(JSON.stringify({
                type: 'MESSAGE',
                idempotency_id: 'k6_' + Math.random().toString(36).substring(7),
                conversation_id: 'k6_room_1',
                sender_id: 'user_' + __VU,
                recipient_id: 'user_target',
                payload: { ciphertext: 'aGVsbG8gd29ybGQ=', nonce: '123456789012', key_version: 1 },
                timestamp: Date.now()
            }));
        });

        socket.on('message', (data) => {
            const msg = JSON.parse(data);
            check(msg, { 'received ACK or MSG': (m) => m.type !== undefined });
        });

        socket.on('close', () => {});
        socket.on('error', (e) => console.error('WS Error:', e.error()));

        socket.setTimeout(() => socket.close(), 30000);
    });

    check(res, { 'status is 101': (r) => r && r.status === 101 });
}
