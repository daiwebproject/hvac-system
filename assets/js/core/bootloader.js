/**
 * Frontend Bootloader - Auto-loads JavaScript modules based on data-module attributes
 * 
 * Usage in HTML:
 *   <body data-module="dashboard">     → loads features/dashboard/index.js
 *   <div data-module="kanban">         → loads features/kanban/index.js
 *   <section data-module="slots,map">  → loads multiple modules
 * 
 * @module core/bootloader
 */

const MODULES_BASE = '/assets/js/features';

// Module registry - maps module names to their init status
const loadedModules = new Map();

/**
 * Dynamically imports a module and calls its init() function if available
 * @param {string} moduleName - Name of the module (e.g., 'dashboard', 'kanban')
 * @returns {Promise<object|null>} The loaded module or null on error
 */
async function loadModule(moduleName) {
    if (loadedModules.has(moduleName)) {
        console.log(`[Bootloader] Module "${moduleName}" already loaded`);
        return loadedModules.get(moduleName);
    }

    const modulePath = `${MODULES_BASE}/${moduleName}/index.js`;

    try {
        console.log(`[Bootloader] Loading module: ${moduleName}`);
        const module = await import(modulePath);

        loadedModules.set(moduleName, module);

        // Call init() if the module exports it
        if (typeof module.init === 'function') {
            console.log(`[Bootloader] Initializing: ${moduleName}`);
            await module.init();
        }

        return module;
    } catch (error) {
        console.error(`[Bootloader] Failed to load "${moduleName}":`, error);
        return null;
    }
}

/**
 * Scans the DOM for data-module attributes and loads corresponding modules
 */
async function scanAndLoad() {
    const elements = document.querySelectorAll('[data-module]');
    const modulesToLoad = new Set();

    elements.forEach(el => {
        const modules = el.dataset.module.split(',').map(m => m.trim());
        modules.forEach(m => modulesToLoad.add(m));
    });

    if (modulesToLoad.size === 0) {
        console.log('[Bootloader] No modules to load');
        return;
    }

    console.log(`[Bootloader] Found modules:`, Array.from(modulesToLoad));

    // Load all modules in parallel
    const loadPromises = Array.from(modulesToLoad).map(loadModule);
    await Promise.all(loadPromises);

    console.log('[Bootloader] All modules loaded');
}

/**
 * Public API - allows manual module loading
 */
export const Bootloader = {
    load: loadModule,
    scan: scanAndLoad,
    getLoaded: () => Array.from(loadedModules.keys()),
};

// Auto-run on DOM ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', scanAndLoad);
} else {
    scanAndLoad();
}

// Expose globally for non-module scripts
window.Bootloader = Bootloader;
