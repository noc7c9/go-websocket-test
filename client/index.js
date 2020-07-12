(async () => {
    const WS_URL = 'ws://localhost:3000/ws/connect';

    const $id = document.getElementById.bind(document);

    // Setup event logging
    const initialiseTemplate = (id) => {
        const template = $id(id);
        return (content) => {
            const instance = template.content.cloneNode(true);

            Object.entries(content).forEach(([selector, content]) => {
                const elem = instance.querySelector(selector);
                elem.textContent = content;
            });

            return instance;
        };
    };

    const eventStream = $id('event-stream');
    const logger = (logFn, consolePrefix, template) => (event) => {
        logFn(consolePrefix, event);
        const node = template({
            '.event-date': new Date().toISOString(),
            '.event-content':
                typeof event === 'string'
                    ? event
                    : JSON.stringify(event, null, 2),
        });
        eventStream.prepend(node);
    };
    const log = {
        info: logger(
            console.log,
            'INFO',
            initialiseTemplate('template-event-info'),
        ),
        warn: logger(
            console.warn,
            'WARN',
            initialiseTemplate('template-event-warn'),
        ),
        error: logger(
            console.error,
            'ERROR',
            initialiseTemplate('template-event-error'),
        ),
        server: logger(
            console.log,
            'SERVER',
            initialiseTemplate('template-event-server'),
        ),
        client: logger(
            console.log,
            'CLIENT',
            initialiseTemplate('template-event-client'),
        ),
    };

    $id('btn-clear-event-stream').addEventListener('click', () => {
        eventStream.innerHTML = '';
    });

    // Connect
    let ws = null;

    const connect = () => {
        if (ws && ws.readyState === WebSocket.OPEN) {
            log.error('Already connected!');
            return;
        }

        log.info('Connecting...');
        const id = Math.floor(Math.random() * 1000000000);
        ws = new WebSocket(`${WS_URL}?id=${id}`);

        ws.addEventListener('error', () => {
            log.error('Failed to connect to server');
        });

        ws.addEventListener('open', () => {
            log.info('Connected to server');
        });

        ws.addEventListener('close', () => {
            log.warn('Disconnected from server');
        });

        ws.addEventListener('message', (event) => {
            log.server(JSON.parse(event.data));
        });
    };

    const disconnect = () => {
        if (ws == null) {
            log.error('Attempted to disconnect without a connection');
            return;
        }
        if (ws.readyState === WebSocket.CLOSED) {
            log.error('Already disonnected');
            return;
        }
        log.info('Disconnecting...');
        ws.close();
    };

    const sendMessage = (event) => {
        if (ws == null) {
            log.error('Attempted to send message without a connection');
            return;
        }
        if (ws.readyState !== WebSocket.OPEN) {
            log.error('Attempted to send message on non-open connection');
            return;
        }
        ws.send(JSON.stringify(event));
        log.client(event);
    };

    // Setup message buttons
    $id('btn-connect').addEventListener('click', () => connect());
    $id('btn-disconnect').addEventListener('click', () => disconnect());
    $id('btn-send-ping').addEventListener('click', () =>
        sendMessage({ type: 'PING' }),
    );

    // Setup text message input/button
    const textInput = $id('input-send-text');
    const sendText = () => {
        const text = textInput.value;
        if (text.length > 0) {
            sendMessage({ type: 'TEXT', text });
            textInput.value = '';
        }
    };
    textInput.addEventListener('keydown', (event) => {
        if (event.key === 'Enter') sendText();
    });
    $id('btn-send-text').addEventListener('click', () => sendText());

    textInput.focus();

    // Connect to the server
    connect();
})();
