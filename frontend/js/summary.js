console.log('summary.js loaded');

class SummaryPage {
    constructor() {
        this.userProfile = null;
        this.registrations = [];
        this.init();
    }

    async init() {
        console.log('SummaryPage.init: starting');
        try {
            console.log('SummaryPage.init: isAuthenticated=', ExunServices.api.isAuthenticated(), 'cookies=', document.cookie);
        } catch (e) {
            console.log('SummaryPage.init: ExunServices not available yet');
        }

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
        console.log('SummaryPage.loadData: fetching /user/profile and /summary');
        const calls = await Promise.allSettled([
            ExunServices.api.apiRequest('/user/profile'),
            ExunServices.api.apiRequest('/summary')
        ]);

        console.log('SummaryPage.loadData: api calls settled', calls);

        const profileCall = calls[0];
        const summaryCall = calls[1];

        if (profileCall.status === 'fulfilled') {
            console.log('SummaryPage.loadData: /user/profile fulfilled', profileCall.value);
            try {
                this.userProfile = profileCall.value.data || {};
            } catch (e) { console.error('SummaryPage.loadData: error reading profile response', e); }
        } else {
            console.warn('SummaryPage.loadData: /user/profile failed', profileCall.reason);
        }

        if (summaryCall.status === 'fulfilled') {
            console.log('SummaryPage.loadData: /summary fulfilled', summaryCall.value);
            try {
                const sumData = summaryCall.value.data || {};
                this.registrations = Array.isArray(sumData.events) ? sumData.events : (sumData.events || []);
                if (sumData.user_info) {
                    const ui = sumData.user_info;
                    this.userProfile = this.userProfile || {};
                    this.userProfile.name = ui.fullname || this.userProfile.name || ui.username || this.userProfile.fullname;
                    this.userProfile.email = ui.email || this.userProfile.email;
                    this.userProfile.school = ui.institution_name || this.userProfile.institution_name || this.userProfile.school;
                    this.userProfile.class = ui.class || this.userProfile.class;
                    this.userProfile.phone = ui.phone || this.userProfile.phone || ui.phone_number || this.userProfile.phone_number;
                }
            } catch (e) { console.error('SummaryPage.loadData: error reading summary response', e); }
        } else {
            console.warn('SummaryPage.loadData: /summary failed', summaryCall.reason);
        }

        if (profileCall.status === 'rejected' && summaryCall.status === 'rejected') {
            const err = profileCall.reason || summaryCall.reason;
            console.error('Failed to load summary data:', err);
            Utils.showToast('Failed to load summary data', 'error');
            const msg = (err && err.message) ? err.message.toLowerCase() : '';
            if (msg.includes('complete signup')) {
                setTimeout(() => window.location.href = '/complete_signup', 1000);
                return;
            }
            if (msg.includes('user not found')) {
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
            'pending-registrations': stats.pendingRegistrations
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
        return {
            totalRegistrations: this.registrations.length,
            confirmedRegistrations: confirmed,
            pendingRegistrations: pending
        };
    }

    renderProfile() {
        const profileContainer = document.getElementById('profile-section');
        if (!profileContainer) return;
        console.log('SummaryPage.renderProfile: userProfile=', this.userProfile);
        if (!this.userProfile) {
            profileContainer.innerHTML = '<div class="loading-placeholder">Loading profile...</div>';
            return;
        }

        const institution = this.userProfile.institution_name || this.userProfile.InstitutionName || this.userProfile.institution || this.userProfile.school || '';
        const address = this.userProfile.address || this.userProfile.Address || '';
        const schoolCode = this.userProfile.school_code || this.userProfile.SchoolCode || this.userProfile.schoolCode || '';
        const principalName = this.userProfile.principals_name || this.userProfile.PrincipalsName || this.userProfile.principalsName || '';
        const principalEmail = this.userProfile.principals_email || this.userProfile.PrincipalsEmail || this.userProfile.principalsEmail || '';

        profileContainer.innerHTML = `
            <div class="profile-card">
                <h4 class="profile-card__title">School Information</h4>
                <div class="registration-card__details">
                    <div class="registration-detail">
                        <span class="registration-detail__label">Institution:</span>
                        <span class="registration-detail__value">${institution || 'Not provided'}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">Address:</span>
                        <span class="registration-detail__value">${address || 'Not provided'}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">School Code:</span>
                        <span class="registration-detail__value">${schoolCode || 'Not provided'}</span>
                    </div>
                </div>
            </div>
            <div class="profile-card">
                <h4 class="profile-card__title">Principal</h4>
                <div class="registration-card__details">
                    <div class="registration-detail">
                        <span class="registration-detail__label">Principal's Name:</span>
                        <span class="registration-detail__value">${principalName || 'Not provided'}</span>
                    </div>
                    <div class="registration-detail">
                        <span class="registration-detail__label">Principal's Email:</span>
                        <span class="registration-detail__value">${principalEmail || 'Not provided'}</span>
                    </div>
                </div>
            </div>
        `;
    }

    renderRegistrations() {
        const registrationsContainer = document.getElementById('registrations-container');
        if (!registrationsContainer) return;
        console.log('SummaryPage.renderRegistrations: registrations=', this.registrations);
        if (!this.registrations || this.registrations.length === 0) {
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
    console.log('summary.js DOMContentLoaded, data-page=', document.body.dataset.page);
    if (document.body && document.body.dataset.page === 'summary') {
        console.log('summary.js: instantiating SummaryPage');
        new SummaryPage();
    }
});
