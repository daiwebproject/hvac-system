/**
 * Kanban Module Entry Point
 * @module features/kanban
 */

import { kanbanBoard } from './kanban-board.js';

/**
 * Initialize Kanban board with Alpine.js component registration
 */
export function init() {
    console.log('[Kanban] Initializing...');

    // Register Alpine.js component if Alpine is loaded
    if (typeof Alpine !== 'undefined') {
        Alpine.data('kanbanBoard', kanbanBoard);
        console.log('[Kanban] Alpine component registered');
    } else {
        console.warn('[Kanban] Alpine.js not loaded');
    }

    console.log('[Kanban] Ready');
}

// Re-export for direct access
export { kanbanBoard };
