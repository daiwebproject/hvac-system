/**
 * Tech Manager Component - Alpine.js data component
 * @module features/techs/tech-manager
 */

/**
 * Define the Tech Manager Alpine.js component
 * @returns {Object} Alpine.js component
 */
export function techManager() {
    return {
        refreshList() {
            if (window.htmx) {
                htmx.trigger('#tech-list-container', 'load');
            } else {
                console.warn('HTMX not found');
            }
        }
    };
}
