/**
 * Slots Module Entry Point
 * @module features/slots
 */

import { slotManager } from './slot-manager.js';

/**
 * Initialize Time Slot management features
 */
export function init() {
    console.log('[Slots] Initializing...');

    // Register Alpine.js component if Alpine is loaded
    if (typeof Alpine !== 'undefined') {
        Alpine.data('slotManager', slotManager);
        console.log('[Slots] Alpine component registered');
    }

    console.log('[Slots] Ready');
}

export { slotManager };
