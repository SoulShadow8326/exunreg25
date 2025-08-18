class HomePage {
    constructor() {
        this.events = [];
        this.init();
    }

    async init() {
        await this.loadEvents();
        this.renderEvents();
        this.setupEventListeners();
    }

    async loadEvents() {
        try {
            const response = await ExunServices.api.apiRequest('/events');
            this.events = response.data || [];
        } catch (error) {
            console.error('Failed to load events from API:', error);
            Utils.showToast('Loading events from backup...', 'info');
            try {
                const localResponse = await fetch('data/events.json');
                const json = await localResponse.json();
                this.events = Object.entries(json.events).map(([name, image]) => ({
                    id: name.replace(/\s+/g, '-').toLowerCase(),
                    name,
                    image: image ? `/illustrations/${image.split('/').pop()}` : '/assets/exun_base.webp',
                    ...json.default,
                    description_short: json.descriptions[name]?.short || '',
                    description_long: json.descriptions[name]?.long || ''
                }));
            } catch (jsonError) {
                console.error('Failed to load events from events.json:', jsonError);
                Utils.showToast('Failed to load events', 'error');
            }
        }
    }

    renderEvents() {
        const eventsGrid = document.querySelector('.events-grid');
        if (!eventsGrid) return;

        eventsGrid.innerHTML = '';

        this.events.forEach(event => {
            const eventCard = this.createEventCard(event);
            eventsGrid.appendChild(eventCard);
        });
    }

    createEventCard(event) {
    const card = document.createElement('a');
    card.href = `/${slugify(event.name)}`;
        card.className = 'event-card';
        card.onclick = (e) => {
            e.preventDefault();
            this.navigateToEvent(event.id);
        };

    const imageUrl = event.image ? `/illustrations/${event.image.split('/').pop()}` : '/assets/exun_base.webp';
        const eligibilityText = formatEligibility(event.eligibility, event.open_to_all);
        const participantsText = formatParticipants(event.participants);
        const modeText = formatEventMode(event.mode);

        card.innerHTML = `
            <img src="${imageUrl}" alt="${event.name}" class="event-card__image" />
            <h3 class="event-card__title">${event.name}</h3>
            <div class="event-card__details">
                <span class="event-card__mode">${modeText}</span>
                <span class="event-card__participants">${participantsText}</span>
                <span class="event-card__eligibility ${event.open_to_all ? 'event-card__eligibility--open' : ''}">${eligibilityText}</span>
            </div>
        `;

        return card;
    }

    setupEventListeners() {
        const exploreBtn = document.querySelector('[data-action="explore-events"]');
        const brochureBtn = document.querySelector('[data-action="view-brochure"]');

        if (exploreBtn) {
            exploreBtn.addEventListener('click', () => {
                window.location.href = '/events.html';
            });
        }

        if (brochureBtn) {
            brochureBtn.addEventListener('click', () => {
                window.location.href = '/brochure.html';
            });
        }

        this.updateHeroImage();
        window.addEventListener('resize', debounce(() => {
            this.updateHeroImage();
        }, 250));
    }

    updateHeroImage() {
        const heroImage = document.querySelector('.hero__image img');
        if (!heroImage) return;

        if (isMobile()) {
            heroImage.style.height = '320px';
            heroImage.style.width = 'auto';
        } else {
            heroImage.style.width = '300px';
            heroImage.style.height = 'auto';
        }
    }

    navigateToEvent(eventId) {
    window.location.href = `/${slugify(eventId)}`;
    }
}

function slugify(text) {
    return String(text)
    .trim()
    .toLowerCase()
    .replace(/[:'/"\?\!\.,]+/g, '') 
    .replace(/\s+/g, '-') 
    .replace(/[^a-z0-9\-]/g, '') 
    .replace(/-+/g, '-');
}

document.addEventListener('DOMContentLoaded', () => {
    if (document.body.dataset.page === 'home') {
        new HomePage();
    }
});
