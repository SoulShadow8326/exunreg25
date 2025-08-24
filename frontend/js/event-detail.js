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

        const imgEl = document.querySelector('.event-detail__image');
        if (imgEl) {
            imgEl.src = this.event.image ? (this.event.image.toString().startsWith('/') ? this.event.image : `/illustrations/${this.event.image.toString().split('/').pop()}`) : '/assets/exun_base.webp';
            imgEl.alt = this.event.name;
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
                const slug = this.event && this.event.name ? String(this.event.name).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') : (this.event && (this.event.id || this.event.ID || this.eventId) || '');
                const url = `/summary?open=${encodeURIComponent(slug)}`;
                window.location.href = url;
            });
        }

        

        const backBtn = document.querySelector('[data-action="back-to-events"]');
        if (backBtn) backBtn.addEventListener('click', () => Utils.redirect('/events', 200));
    }

    async handleRegistration() {
        try {
            const urlParams = new URLSearchParams(window.location.search);
            const eventId = urlParams.get('id') || this.event.id || this.event.name;
            const eventParticipants = this.event.participants || 1;

            const profileResp = await ExunServices.api.apiRequest('/auth/profile');
            if (!profileResp || profileResp.status !== 'success') {
                Utils.redirect('/login', 100);
                return;
            }
            const user = profileResp.data;
            if (!user) {
                Utils.redirect('/complete_signup', 100);
                return;
            }

            const userIsIndividual = user.individual === true || user.Individual === true;
            if (userIsIndividual && !this.event.independent_registration) {
                Utils.showToast('Individual registration not allowed for this event', 'error');
                return;
            }

            return;
        } catch (error) {
            console.error('Registration failed:', error);
            Utils.showToast(error.message || 'Registration failed. Please try again.', 'error');
            return;
        }

    }

    openRegistrationModal() {
    Utils.showToast('Please register from the Summary page', 'info');
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
