class AdminPage {
    constructor() {
        this.currentTab = 'overview';
        this.stats = {};
        this.events = [];
        this.users = [];
    this._tabListenersAttached = false;
    this._modalListenersAttached = false;
    this._createEventBtnAttached = false;
    this._importBtnAttached = false;
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
            let statsObj = {};
            if (response) {
                if (response.data && typeof response.data === 'object') {
                    statsObj = Object.assign({}, response.data, response);
                    delete statsObj.data;
                } else {
                    statsObj = response;
                }
            }
            this.stats = statsObj || {};
            this.renderStats();
            return;
        } catch (error) {
            console.error('Failed to load admin data:', error);
            throw error;
        }
    }

    setupEventListeners() {
        if (!this._tabListenersAttached) {
            const tabs = document.querySelectorAll('.admin-tab');
            tabs.forEach(tab => {
                tab.addEventListener('click', () => {
                    this.switchTab(tab.dataset.tab);
                });
            });
            this._tabListenersAttached = true;
        }

        

        const importBtn = document.getElementById('import-events-btn');
        if (importBtn && !this._importBtnAttached) {
            importBtn.addEventListener('click', async () => {
                importBtn.disabled = true;
                try {
                    const resp = await ExunServices.api.apiRequest('/admin/import_events', { method: 'POST' });
                    const data = resp && (resp.data || resp) || {};
                    if (typeof data.created !== 'undefined' || typeof data.updated !== 'undefined') {
                        Utils.showToast('Imported events: created=' + (data.created || 0) + ' updated=' + (data.updated || 0));
                    } else {
                        Utils.showToast('Import completed');
                    }
                } catch (err) {
                    console.error('Import events failed', err);
                    Utils.showToast('Import failed', 'error');
                } finally {
                    importBtn.disabled = false;
                }
            });
            this._importBtnAttached = true;
        }

        this.setupModalEventListeners();
    }

    setupModalEventListeners() {
        if (this._modalListenersAttached) return;

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

        this._modalListenersAttached = true;
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
            'total-events': this.stats.totalEvents || this.stats.total_events || 0,
            'total-users': this.stats.totalUsers || this.stats.total_users || 0,
            'total-registrations': this.stats.totalRegistrations || this.stats.total_registrations || 0,
            'active-events': this.stats.activeEvents || this.stats.active_events || this.stats.totalEvents || this.stats.total_events || 0
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
            this.events = response.data || [];
            
            const content = document.getElementById('admin-content');
            content.innerHTML = `
                <div class="admin-events">
                    <div class="flex justify-between items-center mb-6">
                        <h3 class="text-xl font-semibold">Event Management</h3>
                    </div>
                    <div class="admin-table-container">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th>Event Name</th>
                                    <th>Mode</th>
                                    <th>Participants</th>
                                    <th>Registrations</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${this.events.map(event => `
                                    <tr>
                                        <td>${event.name}</td>
                                        <td>${Utils.formatEventMode(event.mode)}</td>
                                        <td>${Utils.formatParticipants(event.participants)}</td>
                                        <td>${event.registrations || 0}</td>
                                        
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
                <div id="users-table-container">
                    <div class="loading-placeholder">Loading users...</div>
                </div>
            </div>
        `;

        await this.loadUsers();
    }

    async renderRegistrations() {
        const content = document.getElementById('admin-content');
        content.innerHTML = `
            <div class="admin-registrations">
                <div class="flex justify-between items-center mb-6">
                    <h3 class="text-xl font-semibold">Registration Management</h3>
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
            await this.renderUsersTable();
        } catch (error) {
            console.error('Failed to load users:', error);
            Utils.showToast('Failed to load users', 'error');
        }
    }
    async _ensureAdminEmails() {
        if (window.__ADMIN_EMAILS && Array.isArray(window.__ADMIN_EMAILS)) return;
        try {
            const resp = await ExunServices.api.apiRequest('/admin/config');
            const data = resp && (resp.data || resp) || {};
            const admins = data.admin_emails || data.admin_emails || '';
            window.__ADMIN_EMAILS = (admins || '').split(',').map(s => s.trim().toLowerCase());
        } catch (e) {
            window.__ADMIN_EMAILS = ['exun@dpsrkp.net'];
        }
    }

    async renderUsersTable() {
        await this._ensureAdminEmails();
        const container = document.getElementById('users-table-container');
        if (!container) return;

        const rows = this.users.map(u => {
            const user = (u && u.email) ? u : (u && u[0]) ? u[0] : u;
            const name = user.fullname || user.Fullname || user.name || 'N/A';
            const email = user.email || user.Email || '';
            const school = user.institution_name || user.InstitutionName || user.school || 'N/A';
            let regCount = 0;
            if (user.registrations) {
                if (typeof user.registrations === 'string') {
                    try { const parsed = JSON.parse(user.registrations); regCount = Object.values(parsed).reduce((acc, arr) => acc + (Array.isArray(arr) ? arr.length : 0), 0); } catch(e) { regCount = 0 }
                } else if (typeof user.registrations === 'object') {
                    regCount = Object.values(user.registrations).reduce((acc, arr) => acc + (Array.isArray(arr) ? arr.length : 0), 0);
                }
            }
            let created = '';
            if (user.created_at) created = Utils.formatDate(new Date(user.created_at));
            else if (user.createdAt) created = Utils.formatDate(new Date(user.createdAt));
            else created = 'N/A';

            const id = user.id || user.ID || user.email || email;

            let displayName = Utils.escapeHtml(name);
            try {
                const adminList = window.__ADMIN_EMAILS || [];
                if (email && adminList.indexOf(email.toLowerCase()) !== -1) {
                    displayName = `<span style="color:#2977f5">${displayName}</span>`;
                }
            } catch (e) {}

            return `
                <tr>
                    <td>${displayName}</td>
                    <td>${Utils.escapeHtml(email)}</td>
                    <td>${Utils.escapeHtml(school)}</td>
                    <td>${regCount}</td>
                    <td>${created}</td>
                    <td>
                        <button class="btn btn--secondary" data-user-id="${Utils.escapeHtml(id)}" onclick="adminPage.viewUserDetails('${Utils.escapeHtml(id)}')">View Details</button>
                    </td>
                </tr>
            `;
        }).join('');

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
                    ${rows}
                </tbody>
            </table>
        `;
    }

    async loadRegistrations() {
        try {
            const response = await ExunServices.admin.getEventRegistrations();
            let registrations = [];
            if (Array.isArray(response)) registrations = response;
            else if (response && Array.isArray(response.data)) registrations = response.data;
            else if (response && Array.isArray(response.registrations)) registrations = response.registrations;
            else if (response && response.data && Array.isArray(response.data)) registrations = response.data;
            
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
                        ${registrations.map(reg => {
                            const members = Array.isArray(reg.members) ? reg.members : [];
                            const memberNames = members.length ? members.map(m => (m.name || m.Name || m.Name)).join(', ') : '';
                            const createdRaw = reg.createdAt ? new Date(reg.createdAt) : (reg.createdAt instanceof Date ? reg.createdAt : null);
                            const createdStr = createdRaw ? Utils.formatDate(createdRaw) : (reg.createdAt ? Utils.formatDate(reg.createdAt) : 'N/A');
                            const team = reg.teamName || reg.TeamName || (members.length ? 'Team' : 'Individual');
                            const status = (reg.status || reg.Status || 'pending').toString();
                            return `
                            <tr>
                                <td>${Utils.escapeHtml(reg.eventName || reg.eventName || '')}</td>
                                <td>${Utils.escapeHtml(reg.userEmail || reg.userEmail || '')}</td>
                                <td>${Utils.escapeHtml(team)}</td>
                                <td>${members.length || (reg.memberCount || 1)}</td>
                                <td>${Utils.escapeHtml(createdStr)}</td>
                                <td>
                                    <span class="badge badge--${Utils.escapeHtml(status.toLowerCase())}">
                                        ${Utils.escapeHtml(status)}
                                    </span>
                                </td>
                            </tr>
                        `}).join('')}
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


    editEvent(eventId) {
        Utils.showToast('Edit event functionality coming soon!', 'info');
    }

    viewEventRegistrations(eventId) {
        Utils.showToast('View registrations functionality coming soon!', 'info');
    }
}
