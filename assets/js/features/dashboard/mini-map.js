/**
 * Mini Map Feature for Dashboard
 * @module features/dashboard/mini-map
 */

import { apiClient } from '../../core/api-client.js';

let mapInstance = null;
let mapMarkers = [];
let unmappedJobs = [];

export function initMiniMap() {
    const mapEl = document.getElementById('fleet-map');
    if (!mapEl) return;

    // Check Leaflet
    if (typeof L === 'undefined') {
        console.warn('[MiniMap] Leaflet not loaded');
        return;
    }

    // Cleanup existing
    if (mapInstance) {
        mapInstance.remove();
        mapInstance = null;
        mapMarkers = [];
    }

    // Reset unmapped list
    unmappedJobs = [];
    renderUnmappedInterface(mapEl);

    try {
        // Create Map
        mapInstance = L.map('fleet-map', {
            zoomControl: false,
            attributionControl: false
        }).setView([10.8231, 106.6297], 13); // Default HCM

        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap'
        }).addTo(mapInstance);

        // Add zoom control manually
        L.control.zoom({ position: 'topright' }).addTo(mapInstance);

        // Add markers
        renderMarkers();

        // Auto fit bounds
        setTimeout(fitMapBounds, 1000);

        // Expose fitBounds globally for the button
        window.fitMapBounds = fitMapBounds;

    } catch (e) {
        console.error('[MiniMap] Init error:', e);
    }
}

function renderUnmappedInterface(container) {
    let ui = document.getElementById('map-unmapped-ui');
    if (ui) ui.remove();

    ui = document.createElement('div');
    ui.id = 'map-unmapped-ui';
    ui.className = 'absolute bottom-2 left-2 z-[1000] max-w-xs';
    ui.innerHTML = `
        <div id="unmapped-alert" class="hidden bg-white/95 backdrop-blur shadow-md rounded border border-orange-200 p-2 text-xs">
             <div class="flex items-center gap-2 cursor-pointer" onclick="document.getElementById('unmapped-list').classList.toggle('hidden')">
                <span class="badge badge-warning badge-xs" id="unmapped-count">0</span>
                <span class="font-semibold text-slate-700">Chưa tìm thấy vị trí</span>
                 <i class="fa-solid fa-caret-down ml-auto"></i>
             </div>
             <div id="unmapped-list" class="hidden mt-2 max-h-32 overflow-y-auto space-y-1 pt-1 border-t border-gray-100"></div>
        </div>
    `;

    // Append to parent to avoid Leaflet conflict
    if (container.parentElement) {
        // Ensure parent is relative (it is in dashboard.html)
        container.parentElement.appendChild(ui);
    } else {
        container.appendChild(ui);
    }
}

function updateUnmappedUI() {
    const alertBox = document.getElementById('unmapped-alert');
    const countBadge = document.getElementById('unmapped-count');
    const list = document.getElementById('unmapped-list');

    if (!alertBox || !countBadge || !list) return;

    if (unmappedJobs.length === 0) {
        alertBox.classList.add('hidden');
    } else {
        alertBox.classList.remove('hidden');
        countBadge.textContent = unmappedJobs.length;

        list.innerHTML = unmappedJobs.map(job => `
            <div class="p-1.5 bg-gray-50 rounded border border-gray-100 hover:bg-blue-50 cursor-pointer" 
                 onclick="window.location.href='/admin/bookings?id=${job.id}'">
                <div class="font-bold truncate text-slate-800">${job.customer || 'Khách lẻ'}</div>
                <div class="text-gray-500 truncate text-[10px]">${job.address || 'Không có địa chỉ'}</div>
            </div>
        `).join('');
    }
}

function renderMarkers() {
    const bookings = window.initialBookings || [];
    const techs = window.initialTechs || [];

    // Techs
    techs.forEach(tech => {
        if (tech.active && tech.lat && tech.long) {
            addTechMarker(tech);
        }
    });

    // Jobs
    bookings.forEach((job, index) => {
        if (job.lat && job.long) {
            addJobMarker(job);
        } else if (job.address && job.address.length > 5) {
            // Geocode delay
            setTimeout(() => geocodeAndDraw(job), index * 1200);
        } else {
            // No lat/long and invalid/short address -> Unmapped
            addToUnmapped(job);
        }
    });
}

function addTechMarker(tech) {
    if (!mapInstance) return;
    const marker = L.marker([tech.lat, tech.long], {
        icon: createCustomIcon('#3b82f6', 'fa-user-gear', true)
    }).addTo(mapInstance)
        .bindPopup(`<b>KTV: ${tech.name}</b><br><span class="text-green-600">● Đang hoạt động</span>`);
    mapMarkers.push(marker);
}

function addJobMarker(job) {
    if (!mapInstance) return;
    const color = getJobColor(job.status);
    const marker = L.marker([job.lat, job.long], {
        icon: createCustomIcon(color, 'fa-wrench')
    }).addTo(mapInstance)
        .bindPopup(`
        <div class="text-sm">
            <b>${job.customer || job.customer_name}</b><br>
            <span class="text-gray-500">${job.address}</span><br>
            <span class="badge badge-xs mt-1" style="background:${color};color:white">${job.status_label || job.status}</span>
        </div>
    `);
    mapMarkers.push(marker);
}

function addToUnmapped(job) {
    // Avoid duplicates
    if (!unmappedJobs.find(j => j.id === job.id)) {
        unmappedJobs.push(job);
        updateUnmappedUI();
    }
}

async function geocodeAndDraw(job) {
    if (!job.address || /^\d+$/.test(job.address.replace(/\s|[,\.]/g, ''))) {
        addToUnmapped(job);
        return;
    }

    try {
        const query = `${job.address}, Vietnam`;
        const res = await apiClient.get('/api/public/geocode?q=' + encodeURIComponent(query));

        if (Array.isArray(res) && res.length > 0) {
            const { lat, lon } = res[0];
            job.lat = lat;
            job.long = lon;
            addJobMarker(job);
        } else {
            console.warn(`[MiniMap] Geocode failed for: ${job.address}`);
            addToUnmapped(job);
        }
    } catch (err) {
        console.warn('[MiniMap] Geocode network error:', err);
        addToUnmapped(job);
    }
}

function fitMapBounds() {
    if (!mapInstance || mapMarkers.length === 0) return;
    try {
        const group = new L.featureGroup(mapMarkers);
        mapInstance.fitBounds(group.getBounds(), { padding: [50, 50] });
    } catch (e) {
        console.warn('[MiniMap] Fit bounds error', e);
    }
}

// Helpers
function getJobColor(status) {
    const colors = {
        completed: '#22c55e',
        working: '#a855f7',
        moving: '#a855f7',
        assigned: '#3b82f6',
        cancelled: '#ef4444'
    };
    return colors[status] || '#eab308'; // pending
}

function createCustomIcon(color, iconClass = 'fa-circle', isTech = false) {
    const size = isTech ? 40 : 32;
    const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="${size}" height="${size}" class="drop-shadow-md">
        <path fill="${color}" d="M12 0C7.58 0 4 3.58 4 8c0 5.25 8 16 8 16s8-10.75 8-16c0-4.42-3.58-8-8-8z"/>
        <circle cx="12" cy="8" r="3.5" fill="white"/>
    </svg>`;

    return L.divIcon({
        className: 'custom-map-marker-container',
        html: `
            <div style="position: relative; width: ${size}px; height: ${size}px;">
                ${svg}
                <i class="fa-solid ${iconClass}" style="position: absolute; top: ${isTech ? 8 : 6}px; left: 50%; transform: translateX(-50%); font-size: ${isTech ? 14 : 12}px; color: ${color};"></i>
            </div>
        `,
        iconSize: [size, size],
        iconAnchor: [size / 2, size],
        popupAnchor: [0, -size]
    });
}
