class SummaryPage {
    constructor() {
        this.userProfile = null;
        this.registrations = [];
        this.init();
    }

    async init() {
        if (!ExunServices.api.isAuthenticated()) {
            Utils.showToast('Please log in to view your summary', 'error');
            setTimeout(() => window.location.href = '/login', 2000);
            return;
        }

        await this.loadData();
        this.renderSummary();
        this.setupEventListeners();
    }

    async loadData() {
        try {
            const [profileResponse, summaryResponse] = await Promise.all([
                ExunServices.api.apiRequest('/user/profile'),
                ExunServices.api.apiRequest('/summary')
            ]);
            
            this.userProfile = profileResponse.data || {};
            this.registrations = summaryResponse.data?.events || [];
            
        } catch (error) {
            console.error('Failed to load summary data:', error);
            Utils.showToast('Failed to load summary data', 'error');
            if (error && error.message && error.message.toLowerCase().includes('complete signup')) {
                setTimeout(() => window.location.href = '/complete_signup', 1000);
                return;
            }
            if (error && error.message && error.message.toLowerCase().includes('user not found')) {
                setTimeout(() => window.location.href = '/login', 1000);
                return;
            }
        }
    }

    renderSummary() {
        this.renderStats();
        this.renderProfile();
        this.renderRegistrations();
    }

    renderStats() {
        const stats = this.calculateStats();
        
        const statsElements = {
            'total-registrations': stats.totalRegistrations,
            'confirmed-registrations': stats.confirmedRegistrations,
            'pending-registrations': stats.pendingRegistrations,
            'team-events': stats.teamEvents
        };

        Object.entries(statsElements).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value;
            }
        });
    }

    calculateStats() {
        const confirmed = this.registrations.filter(reg => reg.status === 'confirmed').length;
        const pending = this.registrations.filter(reg => reg.status === 'pending').length;
        const teamEvents = this.registrations.filter(reg => reg.teamName).length;
        
        return {
            totalRegistrations: this.registrations.length,
            confirmedRegistrations: confirmed,
            pendingRegistrations: pending,
            teamEvents
        };
    }

    renderProfile() {
        const profileContainer = document.getElementById('profile-section');
        if (!profileContainer || !this.userProfile) return;

        profileContainer.innerHTML = `
            <div class="profile-card">
                <h4 class="profile-card__title">Personal Information</h4>
                <div class="registration-card__details">
                    <div class="registration-detail">
                        <span class="registration-detail__label">Name:</span>
                        <span class="registration-detail__value">${this.userProfile.name || 'Not provided'}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">Email:</span>
                        <span class="registration-detail__value">${this.userProfile.email || 'Not provided'}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">Phone:</span>
                        <span class="registration-detail__value">${this.userProfile.phone || 'Not provided'}</span>
                    </div>
                </div>
            </div>
            <div class="profile-card">
                <h4 class="profile-card__title">Academic Information</h4>
                <div class="registration-card__details">
                    <div class="registration-detail">
                        <span class="registration-detail__label">School:</span>
                        <span class="registration-detail__value">${this.userProfile.school || 'Not provided'}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">Class:</span>
                        <span class="registration-detail__value">${this.userProfile.class || 'Not provided'}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">City:</span>
                        <span class="registration-detail__value">${this.userProfile.city || 'Not provided'}</span>
                    </div>
                </div>
            </div>
        `;
    }

    renderRegistrations() {
        const registrationsContainer = document.getElementById('registrations-container');
        if (!registrationsContainer) return;

        if (this.registrations.length === 0) {
            registrationsContainer.innerHTML = this.renderEmptyState();
            return;
        }

        registrationsContainer.innerHTML = `
            <div class="registrations-grid">
                ${this.registrations.map(registration => this.renderRegistrationCard(registration)).join('')}
            </div>
        `;
    }

    renderRegistrationCard(registration) {
        const statusClass = `registration-card__status--${registration.status.toLowerCase()}`;
        const registrationDate = Utils.formatDate(registration.createdAt);
        
        return `
            <div class="registration-card">
                <div class="registration-card__header">
                    <h4 class="registration-card__title">${registration.eventName}</h4>
                    <span class="registration-card__status ${statusClass}">${registration.status}</span>
                </div>
                <div class="registration-card__details">
                    <div class="registration-detail">
                        <span class="registration-detail__label">Registration Date:</span>
                        <span class="registration-detail__value">${registrationDate}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">Team Name:</span>
                        <span class="registration-detail__value">${registration.teamName || 'Individual'}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">Event Mode:</span>
                        <span class="registration-detail__value">${Utils.formatEventMode(registration.eventMode)}</span>
                    </div>
                    ${registration.registrationId ? `
                        <div class="registration-detail">
                            <span class="registration-detail__label">Registration ID:</span>
                            <span class="registration-detail__value">${registration.registrationId}</span>
                        </div>
                    ` : ''}
                </div>
                ${registration.teamMembers && registration.teamMembers.length > 0 ? `
                    <div class="team-members">
                        <h5 class="team-members__title">Team Members:</h5>
                        <div class="team-members__list">
                            ${registration.teamMembers.map(member => `
                                <div class="team-member">${member.name} (${member.email})</div>
                            `).join('')}
                        </div>
                    </div>
                ` : ''}
            </div>
        `;
    }

    renderEmptyState() {
        return `
            <div class="empty-state">
                <div class="empty-state__icon">☉</div>
                <h3 class="empty-state__title">No Registrations Yet</h3>
                <p class="empty-state__description">
                    You haven't registered for any events yet. Explore our events and register for your favorites!
                </p>
                <button class="btn btn--primary" onclick="window.location.href='/events'">
                    Browse Events
                </button>
            </div>
        `;
    }

    setupEventListeners() {
        const editProfileBtn = document.getElementById('edit-profile-btn');
        if (editProfileBtn) {
            editProfileBtn.addEventListener('click', () => {
                this.editProfile();
            });
        }



        const logoutBtn = document.getElementById('logout-btn');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', async () => {
                await this.handleLogout();
            });
        }

        this.setupRefreshButton();
    }

    setupRefreshButton() {
        const refreshBtn = document.getElementById('refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', async () => {
                Utils.setLoading(refreshBtn, true);
                await this.loadData();
                this.renderSummary();
                Utils.setLoading(refreshBtn, false);
                Utils.showToast('Summary refreshed!', 'success');
            });
        }
    }

    editProfile() {
    window.location.href = '/signup';
    }

    async handleLogout() {
        try {
            await ExunServices.auth.logout();
            Utils.showToast('Logged out successfully', 'success');
            Utils.redirect('/login', 1000);
        } catch (error) {
            console.error('Logout failed:', error);
            Utils.showToast('Logout failed', 'error');
        }
    }

    async refreshData() {
        await this.loadData();
        this.renderSummary();
        Utils.showToast('Data refreshed successfully', 'success');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    if (document.body.dataset.page === 'summary') {
        new SummaryPage();
    }
});
