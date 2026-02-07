/**
 * Techs Module Entry Point
 * @module features/techs
 */

import { techManager } from './tech-manager.js?v=3';

/**
 * Initialize Technician management features
 */
export function init() {
    console.log('[Techs] Initializing...');

    // Register Alpine.js component if Alpine is loaded
    if (typeof Alpine !== 'undefined') {
        Alpine.data('techManager', techManager);
        console.log('[Techs] Alpine component registered');
    }

    console.log('[Techs] Ready');
}

export { techManager };
