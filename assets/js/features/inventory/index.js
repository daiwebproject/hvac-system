/**
 * Inventory Module Entry Point
 * @module features/inventory
 */

import { inventoryManager } from './inventory-manager.js';

/**
 * Initialize Inventory management features
 */
export function init() {
    console.log('[Inventory] Initializing...');

    // Register Alpine.js component if Alpine is loaded
    if (typeof Alpine !== 'undefined') {
        Alpine.data('inventoryManager', inventoryManager);
        console.log('[Inventory] Alpine component registered');
    }

    console.log('[Inventory] Ready');
}

export { inventoryManager };
