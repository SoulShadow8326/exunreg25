class EventDetailPage {
    constructor() {
        this.event = null;
        this.eventId = this.getEventIdFromUrl();
        this.init();
    }

    getEventIdFromUrl() {
        const urlParams = new URLSearchParams(window.location.search);
    const q = urlParams.get('id');
    if (q) return q;
    const path = window.location.pathname.replace(/^\//, '');
    return path ? decodeURIComponent(path) : null;
    }

    async init() {
        if (!this.eventId) {
            Utils.showToast('Invalid event ID', 'error');
            Utils.redirect('/events', 800);
            return;
        }

        await this.loadEvent();
        this.renderEvent();
        this.setupEventListeners();
    }

    async loadEvent() {
        try {
            const response = await ExunServices.api.apiRequest(`/events/?id=${encodeURIComponent(this.eventId)}`);
            if (response && response.status === 'success') {
                this.event = response.data;
            } else {
                throw new Error(response && response.error ? response.error : 'Event not found');
            }
        } catch (error) {
            console.error('Failed to load event:', error);
            Utils.showToast('Failed to load event details', 'error');
            Utils.redirect('/events', 800);
        }
    }

    renderEvent() {
        if (!this.event) return;

        this.updatePageTitle();
        this.renderEventHeader();
        this.renderEventDetails();
        this.renderEventDescription();
        this.renderRegistrationSection();
    }

    updatePageTitle() {
        document.title = `${this.event.name} - Exun 2025`;
        
        const breadcrumb = document.querySelector('.breadcrumb__current');
        if (breadcrumb) {
            breadcrumb.textContent = this.event.name;
        }
    }

    renderEventHeader() {
        const titleEl = document.getElementById('event-title');
        const subtitleEl = document.getElementById('event-subtitle');
        if (titleEl) titleEl.textContent = this.event.name;
        if (subtitleEl) subtitleEl.textContent = this.event.subtitle || 'Event Details';

    const imageUrl = this.event.image ? `/illustrations/${this.event.image.toString().split('/').pop()}` : '/assets/exun_base.webp';
        const headerContainer = document.querySelector('.event-detail-main');
        if (headerContainer) {
            headerContainer.insertAdjacentHTML('afterbegin', `
                <div class="event-detail__header">
                    <div class="event-detail__image-container">
                        <img src="${imageUrl}" alt="${this.event.name}" class="event-detail__image" />
                    </div>
                    <div class="event-detail__header-content">
                        <h1 class="event-detail__title">${this.event.name}</h1>
                        <div class="event-detail__meta">
                            <span class="event-detail__mode">${Utils.formatEventMode(this.event.mode)}</span>
                            <span class="event-detail__participants">${this.event.participants ? Utils.formatParticipants(this.event.participants) : 'TBA'}</span>
                            <span class="event-detail__points">${this.event.points || 0} Points</span>
                        </div>
                    </div>
                </div>
            `);
        }
    }

    renderEventDetails() {
        const details = document.getElementById('event-info');
        if (!details) return;
    const eligibilityText = Utils.formatEligibility(this.event.eligibility, this.event.open_to_all);
    const registrationType = this.event.individual ? 'Individual' : 'Team';

        details.innerHTML = `
            <div class="event-info-item">
                <span class="event-info-label">Mode:</span>
                <span class="event-info-value">${Utils.formatEventMode(this.event.mode)}</span>
            </div>
            <div class="event-info-item">
                <span class="event-info-label">Participants:</span>
                <span class="event-info-value">${this.event.participants || 'TBA'}</span>
            </div>
            <div class="event-info-item">
                <span class="event-info-label">Eligibility:</span>
                <span class="event-info-value">${eligibilityText}</span>
            </div>
            <div class="event-info-item">
                <span class="event-info-label">Points:</span>
                <span class="event-info-value">${this.event.points || 0}</span>
            </div>
            <div class="event-info-item">
                <span class="event-info-label">Dates:</span>
                <span class="event-info-value">${this.event.dates || 'TBA'}</span>
            </div>
            <div class="event-info-item">
                <span class="event-info-label">Registration:</span>
                <span class="event-info-value">${registrationType}</span>
            </div>
        `;
    }

    renderEventDescription() {
    const description = document.getElementById('event-description');
        if (!description) return;
    const longDesc = this.event.description_long || this.event.description_long === '' ? this.event.description_long : null;
    const shortDesc = this.event.description_short || null;

    let html = '<h2>About This Event</h2>';
    html += '<div class="description-content">';
    if (shortDesc) html += `<p class="description-short">${shortDesc}</p>`;
    if (longDesc) html += `<div class="description-long">${this.formatDescription(longDesc)}</div>`;
    if (!shortDesc && !longDesc) html += '<p>Details coming soon.</p>';
    html += '</div>';

    description.innerHTML = html;
    }

    formatDescription(description) {
        return description
            .split('\n')
            .map(paragraph => paragraph.trim())
            .filter(paragraph => paragraph.length > 0)
            .map(paragraph => `<p>${paragraph}</p>`)
            .join('');
    }

    renderRegistrationSection() {
        const registration = document.getElementById('registration-card');
        if (!registration) return;
        const isAuthenticated = ExunServices.api.isAuthenticated();
        registration.innerHTML = `
            <h3>Ready to Register?</h3>
            <div style="margin-top:0.75rem;">
                <button class="btn btn--primary" id="register-event-btn">Register</button>
            </div>
        `;
        const backButtonHtml = `
            <div style="margin-top:0.5rem; display:flex; justify-content:center;" id="back-to-events-wrapper">
                <button class="btn" data-action="back-to-events" id="back-to-events-btn">← Back to events</button>
            </div>
        `;
        const regEl = document.getElementById('registration-card');
        if (regEl && !document.getElementById('back-to-events-btn')) {
            regEl.insertAdjacentHTML('afterend', backButtonHtml);
        }
    }

    setupEventListeners() {
        const registerBtn = document.getElementById('register-event-btn');
        if (registerBtn) {
            registerBtn.addEventListener('click', () => {
                if (!ExunServices.api.isAuthenticated()) {
                    Utils.redirect('/login', 100);
                    return;
                }
                this.handleRegistration();
            });
        }

        

        const backBtn = document.querySelector('[data-action="back-to-events"]');
        if (backBtn) backBtn.addEventListener('click', () => Utils.redirect('/events', 200));
    }

    async handleRegistration() {
        try {
            const eventId = this.event.id || this.event.name;
            const eventParticipants = this.event.participants || 1;
            const isIndividual = this.event.individual === true || this.event.independent_registration === true || this.event.independent_registration === undefined;

            let participants = [];

            if (isIndividual || eventParticipants === 1) {
                const profileResp = await ExunServices.api.apiRequest('/auth/profile');
                if (!profileResp || profileResp.status !== 'success') {
                    Utils.showToast('Failed to fetch profile. Please complete signup.', 'error');
                    return;
                }
                const user = profileResp.data;
                participants.push({ name: user.fullname || user.username || user.email, email: user.email, class: user.class || 0, phone: user.phone_number || '' });
            } else {
                for (let i = 0; i < eventParticipants; i++) {
                    const name = prompt(`Participant ${i+1} name:`);
                    if (!name) { Utils.showToast('Registration cancelled', 'info'); return; }
                    const email = prompt(`Participant ${i+1} email:`);
                    if (!email) { Utils.showToast('Registration cancelled', 'info'); return; }
                    const classStr = prompt(`Participant ${i+1} class (1-12):`);
                    const cls = parseInt(classStr || '0', 10) || 0;
                    const phone = prompt(`Participant ${i+1} phone (10 digits):`) || '';
                    participants.push({ name, email, class: cls, phone });
                }
            }

            const payload = { id: eventId, data: participants };
            const response = await ExunServices.api.apiRequest('/submit_registrations', { method: 'POST', body: JSON.stringify(payload) });
            if (response && response.status === 'success') {
                Utils.showToast('Registration submitted!', 'success');
                setTimeout(() => Utils.redirect('/summary', 800), 800);
            } else {
                throw new Error(response && response.error ? response.error : 'Registration failed');
            }
        } catch (error) {
            console.error('Registration failed:', error);
            Utils.showToast(error.message || 'Registration failed. Please try again.', 'error');
        }
    }
}

function isAuthenticatedUser() {
    return ExunServices.api.isAuthenticated();
}

document.addEventListener('DOMContentLoaded', async () => {
    if (document.body.dataset.page === 'event-detail') {
        await Utils.loadComponent('components/navbar.html', document.getElementById('navbar-container'));
        await Utils.loadComponent('components/footer.html', document.getElementById('footer-container'));
        new EventDetailPage();
    }
});
