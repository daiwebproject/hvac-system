/**
 * SSE Service - Server-Sent Events connection management
 * @module core/sse-service
 */

/**
 * @typedef {Object} SSEHandlers
 * @property {function(Object): void} onMessage - Message handler
 * @property {function(Event): void} [onError] - Error handler
 * @property {function(): void} [onOpen] - Open handler
 */

/**
 * Create and manage an SSE connection with auto-reconnect
 * @param {string} url - SSE endpoint URL
 * @param {SSEHandlers} handlers - Event handlers
 * @returns {{ close: function(): void }} Connection control object
 */
export function createSSEConnection(url, handlers) {
    let eventSource = null;
    let reconnectTimeout = null;
    let reconnectDelay = 1000; // Start with 1 second
    const maxReconnectDelay = 30000; // Max 30 seconds

    function connect() {
        if (eventSource) {
            eventSource.close();
        }

        console.log(`[SSE] Connecting to ${url}...`);
        eventSource = new EventSource(url);

        eventSource.onopen = () => {
            console.log('[SSE] Connected');
            reconnectDelay = 1000; // Reset delay on successful connection
            handlers.onOpen?.();
        };

        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                handlers.onMessage(data);
            } catch (err) {
                console.error('[SSE] Parse error:', err);
            }
        };

        eventSource.onerror = (event) => {
            console.error('[SSE] Connection error');
            handlers.onError?.(event);

            // Auto-reconnect with exponential backoff
            if (eventSource.readyState === EventSource.CLOSED) {
                scheduleReconnect();
            }
        };
    }

    function scheduleReconnect() {
        if (reconnectTimeout) {
            clearTimeout(reconnectTimeout);
        }

        console.log(`[SSE] Reconnecting in ${reconnectDelay / 1000}s...`);
        reconnectTimeout = setTimeout(() => {
            connect();
            reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
        }, reconnectDelay);
    }

    function close() {
        if (reconnectTimeout) {
            clearTimeout(reconnectTimeout);
            reconnectTimeout = null;
        }
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        console.log('[SSE] Connection closed');
    }

    // Initial connection
    connect();

    return { close };
}

export default createSSEConnection;
