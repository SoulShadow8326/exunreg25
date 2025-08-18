class AdminPage {
    constructor() {
        this.currentTab = 'overview';
        this.stats = {};
        this.events = [];
        this.users = [];
        this.init();
    }

    async init() {
        try {
            await this.loadData();
        } catch (err) {
            const msg = (err && err.message) ? err.message.toLowerCase() : '';
            if (msg.includes('401') || msg.includes('unauthorized')) {
                Utils.showToast('Access denied. Admin privileges required.', 'error');
                setTimeout(() => window.location.href = '/login', 2000);
                return;
            }
            Utils.showToast('Failed to load admin data', 'error');
            return;
        }
        this.setupEventListeners();
        this.renderCurrentTab();
    }

    async loadData() {
        try {
            const response = await ExunServices.api.apiRequest('/admin/stats');
            this.stats = response.data || {};
            this.renderStats();
            return;
        } catch (error) {
            console.error('Failed to load admin data:', error);
            throw error;
        }
    }

    setupEventListeners() {
        const tabs = document.querySelectorAll('.admin-tab');
        tabs.forEach(tab => {
            tab.addEventListener('click', () => {
                this.switchTab(tab.dataset.tab);
            });
        });

        const createEventBtn = document.getElementById('create-event-btn');
        if (createEventBtn) {
            createEventBtn.addEventListener('click', () => {
                this.showCreateEventModal();
            });
        }

        const exportBtn = document.getElementById('export-btn');
        if (exportBtn) {
            exportBtn.addEventListener('click', () => {
                this.exportData();
            });
        }

        this.setupModalEventListeners();
    }

    setupModalEventListeners() {
        const modal = document.getElementById('admin-modal');
        const closeBtn = document.getElementById('modal-close');
        
        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                this.closeModal();
            });
        }

        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.closeModal();
                }
            });
        }

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeModal();
            }
        });
    }

    switchTab(tabName) {
        this.currentTab = tabName;
        
        const tabs = document.querySelectorAll('.admin-tab');
        tabs.forEach(tab => {
            tab.classList.remove('admin-tab--active');
            if (tab.dataset.tab === tabName) {
                tab.classList.add('admin-tab--active');
            }
        });
        
        this.renderCurrentTab();
    }

    async renderCurrentTab() {
        const content = document.getElementById('admin-content');
        if (!content) return;

        switch (this.currentTab) {
            case 'overview':
                content.innerHTML = this.renderOverview();
                break;
            case 'events':
                await this.renderEvents();
                break;
            case 'users':
                await this.renderUsers();
                break;
            case 'registrations':
                await this.renderRegistrations();
                break;
            default:
                content.innerHTML = '<p>Tab not found</p>';
        }
    }

    renderStats() {
        const statsElements = {
            'total-events': this.stats.totalEvents || 0,
            'total-users': this.stats.totalUsers || 0,
            'total-registrations': this.stats.totalRegistrations || 0,
            'active-events': this.stats.activeEvents || 0
        };

        Object.entries(statsElements).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value;
            }
        });
    }

    renderOverview() {
        return `
            <div class="admin-overview">
                <h3 class="mb-4 text-xl font-semibold">System Overview</h3>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div class="admin-card">
                        <h4 class="font-semibold mb-3">Recent Activity</h4>
                        <div class="space-y-2">
                            <div class="flex justify-between">
                                <span class="stat-label">New registrations today:</span>
                                <span class="stat-value">${this.stats.todayRegistrations || 0}</span>
                            </div>
                            <div class="flex justify-between">
                                <span class="stat-label">Active sessions:</span>
                                <span class="stat-value">${this.stats.activeSessions || 0}</span>
                            </div>
                        </div>
                    </div>
                    <div class="admin-card">
                        <h4 class="font-semibold mb-3">System Health</h4>
                        <div class="space-y-2">
                            <div class="flex justify-between">
                                <span class="stat-label">Server status:</span>
                                <span class="stat-value text-green-600">Healthy</span>
                            </div>
                            <div class="flex justify-between">
                                <span class="stat-label">Database status:</span>
                                <span class="stat-value text-green-600">Connected</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    async renderEvents() {
        try {
            const response = await ExunServices.events.getAllEvents();
            this.events = response.events || [];
            
            const content = document.getElementById('admin-content');
            content.innerHTML = `
                <div class="admin-events">
                    <div class="flex justify-between items-center mb-6">
                        <h3 class="text-xl font-semibold">Event Management</h3>
                        <button id="create-event-btn" class="btn btn--primary">
                            Create New Event
                        </button>
                    </div>
                    <div class="admin-table-container">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Event Name</th>
                                    <th>Mode</th>
                                    <th>Participants</th>
                                    <th>Registrations</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${this.events.map(event => `
                                    <tr>
                                        <td>${event.name}</td>
                                        <td>${Utils.formatEventMode(event.mode)}</td>
                                        <td>${Utils.formatParticipants(event.participants)}</td>
                                        <td>${event.registrations || 0}</td>
                                        <td>
                                            <div class="flex gap-2">
                                                <button class="btn btn--secondary" onclick="adminPage.editEvent('${event.id}')">
                                                    Edit
                                                </button>
                                                <button class="btn btn--secondary" onclick="adminPage.viewEventRegistrations('${event.id}')">
                                                    View Registrations
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                `).join('')}
                            </tbody>
                        </table>
                    </div>
                </div>
            `;
            
            this.setupEventListeners();
        } catch (error) {
            console.error('Failed to load events:', error);
            Utils.showToast('Failed to load events', 'error');
        }
    }

    async renderUsers() {
        const content = document.getElementById('admin-content');
        content.innerHTML = `
            <div class="admin-users">
                <h3 class="text-xl font-semibold mb-6">User Management</h3>
                <div class="mb-4">
                    <input 
                        type="text" 
                        id="user-search" 
                        class="form-input" 
                        placeholder="Search users by email or name..."
                    >
                </div>
                <div id="users-table-container">
                    <div class="loading-placeholder">Loading users...</div>
                </div>
            </div>
        `;

        const searchInput = document.getElementById('user-search');
        if (searchInput) {
            searchInput.addEventListener('input', Utils.debounce((e) => {
                this.searchUsers(e.target.value);
            }, 300));
        }

        await this.loadUsers();
    }

    async renderRegistrations() {
        const content = document.getElementById('admin-content');
        content.innerHTML = `
            <div class="admin-registrations">
                <div class="flex justify-between items-center mb-6">
                    <h3 class="text-xl font-semibold">Registration Management</h3>
                    <button id="export-btn" class="btn btn--primary">
                        Export Data
                    </button>
                </div>
                <div id="registrations-content">
                    <div class="loading-placeholder">Loading registrations...</div>
                </div>
            </div>
        `;

        this.setupEventListeners();
        await this.loadRegistrations();
    }

    async loadUsers(search = '') {
        try {
            const response = await ExunServices.admin.getUserDetails(search);
            this.users = response.users || [];
            this.renderUsersTable();
        } catch (error) {
            console.error('Failed to load users:', error);
            Utils.showToast('Failed to load users', 'error');
        }
    }

    renderUsersTable() {
        const container = document.getElementById('users-table-container');
        if (!container) return;

        container.innerHTML = `
            <table class="admin-table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Email</th>
                        <th>School</th>
                        <th>Registrations</th>
                        <th>Created</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${this.users.map(user => `
                        <tr>
                            <td>${user.name || 'N/A'}</td>
                            <td>${user.email}</td>
                            <td>${user.school || 'N/A'}</td>
                            <td>${user.registrationCount || 0}</td>
                            <td>${Utils.formatDate(user.createdAt)}</td>
                            <td>
                                <button class="btn btn--secondary" onclick="adminPage.viewUserDetails('${user.id}')">
                                    View Details
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    }

    async loadRegistrations() {
        try {
            const response = await ExunServices.admin.getEventRegistrations();
            const registrations = response.registrations || [];
            
            const container = document.getElementById('registrations-content');
            container.innerHTML = `
                <table class="admin-table">
                    <thead>
                        <tr>
                            <th>Event</th>
                            <th>User</th>
                            <th>Team Name</th>
                            <th>Members</th>
                            <th>Registration Date</th>
                            <th>Status</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${registrations.map(reg => `
                            <tr>
                                <td>${reg.eventName}</td>
                                <td>${reg.userEmail}</td>
                                <td>${reg.teamName || 'Individual'}</td>
                                <td>${reg.memberCount || 1}</td>
                                <td>${Utils.formatDate(reg.createdAt)}</td>
                                <td>
                                    <span class="badge badge--${reg.status.toLowerCase()}">
                                        ${reg.status}
                                    </span>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            `;
        } catch (error) {
            console.error('Failed to load registrations:', error);
            Utils.showToast('Failed to load registrations', 'error');
        }
    }

    showCreateEventModal() {
        const modal = document.getElementById('admin-modal');
        const modalContent = document.getElementById('modal-content');
        
        modalContent.innerHTML = `
            <div class="admin-modal__header">
                <h3 class="admin-modal__title">Create New Event</h3>
                <button id="modal-close" class="admin-modal__close">&times;</button>
            </div>
            <form class="admin-form" id="create-event-form">
                <div class="admin-form__row">
                    <div class="admin-form__group">
                        <label class="admin-form__label">Event Name</label>
                        <input type="text" name="name" class="admin-form__input" required>
                    </div>
                    <div class="admin-form__group">
                        <label class="admin-form__label">Mode</label>
                        <select name="mode" class="admin-form__select" required>
                            <option value="">Select mode</option>
                            <option value="online">Online</option>
                            <option value="offline">Offline</option>
                            <option value="hybrid">Hybrid</option>
                        </select>
                    </div>
                </div>
                <div class="admin-form__row">
                    <div class="admin-form__group">
                        <label class="admin-form__label">Max Participants per Team</label>
                        <input type="number" name="participants" class="admin-form__input" min="1" required>
                    </div>
                    <div class="admin-form__group">
                        <label class="admin-form__label">Eligibility (Class Range)</label>
                        <div class="flex gap-2">
                            <input type="number" name="minClass" class="admin-form__input" placeholder="Min" min="6" max="12">
                            <input type="number" name="maxClass" class="admin-form__input" placeholder="Max" min="6" max="12">
                        </div>
                    </div>
                </div>
                <div class="admin-form__group">
                    <label class="admin-form__label">Description</label>
                    <textarea name="description" class="admin-form__textarea" required></textarea>
                </div>
                <div class="admin-actions">
                    <button type="submit" class="btn btn--primary">Create Event</button>
                    <button type="button" class="btn btn--secondary" id="cancel-create">Cancel</button>
                </div>
            </form>
        `;
        
        modal.classList.add('admin-modal--open');
        this.setupModalEventListeners();
        
        const form = document.getElementById('create-event-form');
        form.addEventListener('submit', (e) => this.handleCreateEvent(e));
        
        const cancelBtn = document.getElementById('cancel-create');
        cancelBtn.addEventListener('click', () => this.closeModal());
    }

    async handleCreateEvent(e) {
        e.preventDefault();
        
        const formData = new FormData(e.target);
        const eventData = {
            name: formData.get('name'),
            mode: formData.get('mode'),
            participants: parseInt(formData.get('participants')),
            description: formData.get('description'),
            eligibility: [
                parseInt(formData.get('minClass')) || 6,
                parseInt(formData.get('maxClass')) || 12
            ],
            open_to_all: !formData.get('minClass') || !formData.get('maxClass')
        };

        try {
            const submitBtn = e.target.querySelector('button[type="submit"]');
            Utils.setLoading(submitBtn, true);
            
            await ExunServices.admin.updateEvent(eventData);
            Utils.showToast('Event created successfully!', 'success');
            this.closeModal();
            this.renderCurrentTab();
        } catch (error) {
            console.error('Failed to create event:', error);
            Utils.showToast('Failed to create event', 'error');
        }
    }

    async exportData() {
        try {
            const response = await ExunServices.admin.exportData('csv');
            
            const blob = new Blob([response], { type: 'text/csv' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `exun-2025-data-${new Date().toISOString().split('T')[0]}.csv`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            window.URL.revokeObjectURL(url);
            
            Utils.showToast('Data exported successfully!', 'success');
        } catch (error) {
            console.error('Failed to export data:', error);
            Utils.showToast('Failed to export data', 'error');
        }
    }

    searchUsers(query) {
        this.loadUsers(query);
    }

    editEvent(eventId) {
        Utils.showToast('Edit event functionality coming soon!', 'info');
    }

    viewEventRegistrations(eventId) {
        Utils.showToast('View registrations functionality coming soon!', 'info');
    }

    viewUserDetails(userId) {
        Utils.showToast('User details functionality coming soon!', 'info');
    }

    closeModal() {
        const modal = document.getElementById('admin-modal');
        modal.classList.remove('admin-modal--open');
    }
}

let adminPage;

document.addEventListener('DOMContentLoaded', () => {
    if (document.body.dataset.page === 'admin') {
        adminPage = new AdminPage();
    }
});
