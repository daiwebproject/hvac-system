/**
 * Dashboard Module Entry Point
 * @module features/dashboard
 */

import { initMiniMap } from './mini-map.js';

/**
 * Initialize all dashboard features
 */
export function init() {
    console.log('[Dashboard] Initializing...');

    // Initialize mini map for fleet tracking
    initMiniMap();

    console.log('[Dashboard] Ready');
}

// Re-export for direct access
export { initMiniMap };
