/**
 * Admin Module Entry Point
 * @module admin
 * 
 * This is the main entry point for admin-side ES Modules.
 * It imports feature modules and registers them as Alpine.js components.
 */

import { kanbanBoard } from '../features/kanban/kanban-board.js';
import { slotManager } from '../features/slots/slot-manager.js';
import { inventoryManager } from '../features/inventory/inventory-manager.js';
import { initMiniMap } from '../features/dashboard/mini-map.js';

// Export for direct usage
export { kanbanBoard, slotManager, inventoryManager, initMiniMap };

// Register components
function registerComponents() {
    if (!window.Alpine) {
        // Alpine not loaded yet, wait for event
        return;
    }

    // Register components as Alpine data
    window.Alpine.data('kanbanBoard', kanbanBoard);
    window.Alpine.data('slotManager', slotManager);
    window.Alpine.data('inventoryManager', inventoryManager);

    // Also expose globally for compatibility with existing templates
    window.kanbanBoard = kanbanBoard;
    window.slotManager = slotManager;
    window.inventoryManager = inventoryManager;

    // Initialize Mini Map (if element exists)
    initMiniMap();

    console.log('[Admin] ES Modules initialized');
}

// 1. If Alpine is already ready, register immediately
if (window.Alpine) {
    registerComponents();
} else {
    // 2. Otherwise wait for Alpine to initialize
    // This is the preferred way as it ensures Alpine is ready
    document.addEventListener('alpine:init', registerComponents);

    // 3. Fallback: Listen for window load (in case alpine:init missed or non-standard load)
    window.addEventListener('load', () => {
        if (window.Alpine) registerComponents();
    });
}
