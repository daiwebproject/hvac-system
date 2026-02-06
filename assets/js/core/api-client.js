/**
 * API Client - Centralized HTTP request handler
 * @module core/api-client
 */

/**
 * Make a fetch request with common error handling
 * @param {string} url - Request URL
 * @param {Object} options - Fetch options
 * @returns {Promise<Response>}
 */
async function request(url, options = {}) {
    const defaultOptions = {
        headers: {
            'Accept': 'application/json',
        },
    };

    const mergedOptions = { ...defaultOptions, ...options };

    try {
        const response = await fetch(url, mergedOptions);
        return response;
    } catch (error) {
        console.error(`[API] Network error: ${url}`, error);
        throw error;
    }
}

/**
 * API Client with common HTTP methods
 */
export const apiClient = {
    /**
     * GET request
     * @param {string} url
     * @returns {Promise<any>}
     */
    async get(url) {
        const response = await request(url, { method: 'GET' });
        if (!response.ok) {
            throw new Error(`GET ${url} failed: ${response.status}`);
        }
        return response.json();
    },

    /**
     * POST request with FormData
     * @param {string} url
     * @param {FormData|Object} data
     * @returns {Promise<Response>}
     */
    async post(url, data) {
        const body = data instanceof FormData ? data : (() => {
            const fd = new FormData();
            Object.entries(data).forEach(([key, value]) => fd.append(key, value));
            return fd;
        })();

        return request(url, {
            method: 'POST',
            body,
        });
    },

    /**
     * POST request expecting JSON response
     * @param {string} url
     * @param {Object} data
     * @returns {Promise<any>}
     */
    async postJSON(url, data) {
        const response = await request(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json',
            },
            body: JSON.stringify(data),
        });

        if (!response.ok) {
            throw new Error(`POST ${url} failed: ${response.status}`);
        }
        return response.json();
    },

    /**
     * DELETE request
     * @param {string} url
     * @returns {Promise<Response>}
     */
    async delete(url) {
        return request(url, { method: 'DELETE' });
    },
};

export default apiClient;
