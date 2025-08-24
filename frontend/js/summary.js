console.log('summary.js loaded');

function clientSlugify(s) {
    if (!s) return '';
    return String(s).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
}

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

        try {
            const urlParams = new URLSearchParams(window.location.search);
            const openSlug = urlParams.get('open');
            if (openSlug) {
                const match = (this.registrations || []).find(r => {
                    const name = r.eventName || r.event_name || r.EventName || r.event || '';
                    const id = r.eventId || r.eventID || r.EventID || r.event_id || r.event || '';
                    const slug = clientSlugify(name || id);
                    return slug === openSlug || String(id) === openSlug;
                });
                if (match) {
                    setTimeout(() => {
                        const container = document.getElementById('registrations-container');
                        if (!container) return;
                        const card = container.querySelector(`.registration-card[data-event-id="${match.eventId || match.eventID || match.event_id || match.event}"]`);
                        if (card) this.toggleInlineEditor(card, match);
                    }, 120);
                }
            }
        } catch (e) {}
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
        const total = Array.isArray(this.registrations) ? this.registrations.length : 0;
        const confirmed = (this.registrations || []).filter(reg => ((reg.status || reg.Status || '').toString().toLowerCase() === 'confirmed')).length;
        const pending = (this.registrations || []).filter(reg => ((reg.status || reg.Status || '').toString().toLowerCase() === 'pending')).length;
        return {
            totalRegistrations: total,
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

        setTimeout(() => {
            const viewBtns = registrationsContainer.querySelectorAll('.btn-view-details');
            viewBtns.forEach(btn => {
                btn.addEventListener('click', (ev) => {
                    ev.stopPropagation();
                    const eid = btn.dataset.eventId;
                    if (!eid) return;
                    const regObj = (Array.isArray(this.registrations) ? this.registrations.find(r => (r.eventId||r.eventID||r.event_id||r.event||'').toString()===eid.toString()) : null);
                    const nameForSlug = (regObj && (regObj.eventName || regObj.event_name || regObj.EventName)) ? (regObj.eventName || regObj.event_name || regObj.EventName) : eid;
                    const slug = clientSlugify(nameForSlug);
                    window.location.href = `/${encodeURIComponent(slug)}`;
                });
            });

            const regBtns = registrationsContainer.querySelectorAll('.btn-register');
            regBtns.forEach(btn => {
                btn.addEventListener('click', async (ev) => {
                    ev.stopPropagation();
                    const eid = btn.dataset.eventId;
                    if (!eid) return;
                    const card = registrationsContainer.querySelector(`.registration-card[data-event-id="${eid}"]`);
                    const reg = this.registrations.find(r => (r.eventId || r.eventID || r.event_id || r.event || '').toString() === eid.toString());
                    if (card) await this.toggleInlineEditor(card, reg);
                });
            });

            const cards = registrationsContainer.querySelectorAll('.registration-card[data-event-id]');
            cards.forEach(card => {
                card.style.cursor = 'pointer';
                card.addEventListener('click', async (ev) => {
                    if (ev.target && ev.target.closest && ev.target.closest('button')) return;
                    const eid = card.dataset.eventId;
                    if (!eid) return;
                    const reg = this.registrations.find(r => (r.eventId || r.eventID || r.event_id || r.event || '').toString() === eid.toString());
                    await this.toggleInlineEditor(card, reg);
                });
            });
        }, 20);
    }

    async toggleInlineEditor(cardEl, registration) {
        const existing = cardEl.querySelector('.inline-registration-editor');
        const checkEditorDirty = (editorEl) => {
            try {
                const rawInitial = editorEl.__initialData || '[]';
                let initial = [];
                try { initial = JSON.parse(rawInitial); } catch (e) { initial = []; }
                const inputs = Array.from(editorEl.querySelectorAll('.inline-member-row')).map(r => ({
                    name: ((r.querySelector('[data-name="name"]')||{value:''}).value || '').trim(),
                    email: ((r.querySelector('[data-name="email"]')||{value:''}).value || '').trim(),
                    class: ((r.querySelector('[data-name="class"]')||{value:''}).value || '').trim(),
                    phone: ((r.querySelector('[data-name="phone"]')||{value:''}).value || '').trim()
                }));
                if (initial.length !== inputs.length) return true;
                for (let i = 0; i < inputs.length; i++) {
                    const a = initial[i] || {};
                    const b = inputs[i];
                    if (String((a.name||'')).trim() !== b.name) return true;
                    if (String((a.email||'')).trim() !== b.email) return true;
                    if (String((a.class||'')).trim() !== b.class) return true;
                    if (String((a.phone||'')).trim() !== b.phone) return true;
                }
                return false;
            } catch (e) { return false; }
        };

        const showDiscardModal = (message) => {
            return new Promise((resolve) => {
                const overlay = document.createElement('div');
                overlay.style.position = 'fixed';
                overlay.style.inset = '0';
                overlay.style.background = 'rgba(0,0,0,0.35)';
                overlay.style.display = 'flex';
                overlay.style.alignItems = 'center';
                overlay.style.justifyContent = 'center';
                overlay.style.zIndex = 9999;
                overlay.tabIndex = 0;

                const box = document.createElement('div');
                box.style.background = '#fff';
                box.style.borderRadius = '10px';
                box.style.padding = '20px';
                box.style.maxWidth = '420px';
                box.style.width = '92%';
                box.style.boxShadow = '0 20px 60px rgba(2,6,23,0.2)';
                box.style.fontFamily = "'Raleway', sans-serif";

                const title = document.createElement('div');
                title.style.fontSize = '16px';
                title.style.fontWeight = '700';
                title.style.marginBottom = '8px';
                title.textContent = 'Discard changes?';

                const desc = document.createElement('div');
                desc.style.fontSize = '14px';
                desc.style.color = '#374151';
                desc.style.marginBottom = '16px';
                desc.textContent = message || 'You have unsaved changes. Are you sure you want to discard them?';

                const actions = document.createElement('div');
                actions.style.display = 'flex';
                actions.style.justifyContent = 'flex-end';
                actions.style.gap = '10px';

                const cancel = document.createElement('button');
                cancel.className = 'btn btn--secondary';
                cancel.textContent = 'Keep editing';
                const discard = document.createElement('button');
                discard.className = 'btn btn--primary';
                discard.textContent = 'Discard';

                actions.appendChild(cancel);
                actions.appendChild(discard);

                box.appendChild(title);
                box.appendChild(desc);
                box.appendChild(actions);
                overlay.appendChild(box);
                document.body.appendChild(overlay);

                const cleanup = () => {
                    overlay.removeEventListener('keydown', onKeyDown);
                    overlay.remove();
                };

                const onKeyDown = (ev) => {
                    if (ev.key === 'Escape') { cleanup(); resolve(false); }
                    if (ev.key === 'Enter') { cleanup(); resolve(true); }
                };

                cancel.addEventListener('click', () => { cleanup(); resolve(false); });
                discard.addEventListener('click', () => { cleanup(); resolve(true); });
                overlay.addEventListener('click', (ev) => { if (ev.target === overlay) { cleanup(); resolve(false); } });

                overlay.addEventListener('keydown', onKeyDown);
                setTimeout(() => { overlay.focus(); cancel.focus(); }, 10);
            });
        };

        if (existing) {
            if (checkEditorDirty(existing)) {
                const confirmed = await showDiscardModal();
                if (!confirmed) return;
            }
            existing.remove(); cardEl.classList.remove('registration-card--open');
            return;
        }

        const eventId = cardEl.dataset.eventId;
        const container = document.createElement('div');
        container.className = 'inline-registration-editor summary-inline-animate';
        container.setAttribute('role', 'region');
        container.setAttribute('aria-label', `Edit registration for ${registration && (registration.eventName||registration.event_name||registration.EventName) || eventId}`);
        container.style.marginTop = '12px';
        container.style.padding = '14px 8px 0 8px';
        container.style.borderTop = '1px solid rgba(0,0,0,0.06)';

        const capacity = parseInt(registration && (registration.capacity || registration.Capacity) || 1, 10) || 1;
        const members = (registration && (registration.participants || registration.teamMembers)) ? (registration.participants || registration.teamMembers) : [];

        const rows = [];
        const initialData = [];

        const closeEditorSafely = () => { try { container.remove(); cardEl.classList.remove('registration-card--open'); } catch (e) {} };

        const createRow = (p) => {
            const row = document.createElement('div');
            row.className = 'inline-member-row';
            row.style.display = 'grid';
            row.style.gridTemplateColumns = '1fr 1fr 90px 130px auto';
            row.style.gap = '12px';
            row.style.marginBottom = '10px';
            const nameVal = String(p && (p.name||p.Name) || '');
            const emailVal = String(p && (p.email||p.Email) || '');
            const classVal = String(p && (p.class||p.Class) || '');
            const phoneVal = String(p && (p.phone||p.Phone) || '');
            row.innerHTML = `
                <input class="form-input" data-name="name" placeholder="Full name" value="${nameVal.replace(/\"/g,'&quot;')}" />
                <input class="form-input" data-name="email" placeholder="Email" value="${emailVal.replace(/\"/g,'&quot;')}" />
                <input class="form-input" data-name="class" placeholder="Class" value="${classVal.replace(/\"/g,'&quot;')}" />
                <input class="form-input" data-name="phone" placeholder="Phone" value="${phoneVal.replace(/\"/g,'&quot;')}" />
                <button class="btn btn--tertiary btn-inline-clear"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-delete-icon lucide-delete"><path d="M10 5a2 2 0 0 0-1.344.519l-6.328 5.74a1 1 0 0 0 0 1.481l6.328 5.741A2 2 0 0 0 10 19h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2z"/><path d="m12 9 6 6"/><path d="m18 9-6 6"/></svg></button>
            `;
            const clearBtn = row.querySelector('.btn-inline-clear');
            clearBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                if (rows.length <= 1) {
                    closeEditorSafely();
                    return;
                }
                const idx = rows.indexOf(row);
                if (idx !== -1) rows.splice(idx, 1);
                row.remove();
                if (addBtn) {
                    if (rows.length >= capacity) {
                        addBtn.classList.add('is-disabled');
                        addBtn.setAttribute('aria-disabled', 'true');
                    } else {
                        addBtn.classList.remove('is-disabled');
                        addBtn.removeAttribute('aria-disabled');
                    }
                }
            });
            return row;
        };

        const initialCount = Math.max(1, Math.min(members.length, capacity));
        for (let i = 0; i < initialCount; i++) {
            const p = members[i] || {};
            const r = createRow(p);
            rows.push(r);
            container.appendChild(r);
            initialData.push({ name: r.querySelector('[data-name="name"]').value || '', email: r.querySelector('[data-name="email"]').value || '', class: r.querySelector('[data-name="class"]').value || '', phone: r.querySelector('[data-name="phone"]').value || '' });
        }

        container.__initialData = JSON.stringify(initialData);

        const addBtn = document.createElement('button');
        addBtn.className = 'btn btn--tertiary';
        addBtn.textContent = 'Add participant';
        addBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (rows.length >= capacity) {
                Utils.showToast('Max participants reached', 'error');
                return;
            }
            const existingIndex = rows.length;
            const p = members[existingIndex] || {};
            const nr = createRow(p);
            rows.push(nr);
            if (addContainer) container.insertBefore(nr, addContainer);
            if (rows.length >= capacity) {
                addBtn.classList.add('is-disabled');
                addBtn.setAttribute('aria-disabled', 'true');
            } else {
                addBtn.classList.remove('is-disabled');
                addBtn.removeAttribute('aria-disabled');
            }
            setTimeout(() => { const first = nr.querySelector('input[data-name="name"]'); if (first) first.focus(); }, 20);
        });
        const addContainer = document.createElement('div');
        addContainer.style.marginTop = '8px';
        addContainer.style.display = 'flex';
        addContainer.style.justifyContent = 'flex-start';
        addContainer.appendChild(addBtn);
        container.appendChild(addContainer);
        if (addBtn) {
            if (rows.length >= capacity) {
                addBtn.classList.add('is-disabled');
                addBtn.setAttribute('aria-disabled', 'true');
            } else {
                addBtn.classList.remove('is-disabled');
                addBtn.removeAttribute('aria-disabled');
            }
        }

        const actions = document.createElement('div');
        actions.style.marginTop = '14px';
        actions.style.display = 'flex';
        actions.style.gap = '12px';
        actions.style.justifyContent = 'flex-end';

        const saveBtn = document.createElement('button');
        saveBtn.className = 'btn btn--primary';
        saveBtn.textContent = 'Save';
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'btn btn--secondary';
        cancelBtn.textContent = 'Cancel';
        actions.appendChild(saveBtn);
        actions.appendChild(cancelBtn);
        container.appendChild(actions);

        cancelBtn.addEventListener('click', async (e) => {
            e.stopPropagation();
            if (checkEditorDirty(container)) {
                const confirmed = await showDiscardModal();
                if (!confirmed) return;
            }
            container.remove(); cardEl.classList.remove('registration-card--open');
        });
        saveBtn.addEventListener('click', async (e) => {
            e.stopPropagation();
            const data = [];
            for (const r of rows) {
                const name = r.querySelector('[data-name="name"]').value.trim();
                const email = r.querySelector('[data-name="email"]').value.trim();
                const cls = r.querySelector('[data-name="class"]').value.trim();
                const phone = r.querySelector('[data-name="phone"]').value.trim();
                if (!name) continue;
                data.push({ name, email, class: parseInt(cls||0,10) || cls, phone });
            }
            try {
                const resp = await fetch('/api/submit_registrations', { method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include', body: JSON.stringify({ id: eventId, data }) });
                const json = await resp.json();
                if (json === true) {
                    Utils.showToast('Saved', 'success');
                    await this.refreshData();
                } else {
                    Utils.showToast('Save failed', 'error');
                }
            } catch (err) { Utils.showToast('Save failed', 'error'); }
            container.remove(); cardEl.classList.remove('registration-card--open');
        });

        container.addEventListener('click', (e) => { e.stopPropagation(); });

        cardEl.appendChild(container);
        cardEl.classList.add('registration-card--open');
        const firstInput = container.querySelector('input[data-name="name"]');
        if (firstInput) { setTimeout(() => firstInput.focus(), 60); }
    }


    renderRegistrationCard(registration) {
        const createdRaw = registration.createdAt || registration.created_at || registration.CreatedAt || null;
        const eventName = registration.eventName || registration.event_name || registration.EventName || registration.Event || 'Unnamed Event';
        const statusRaw = (registration.status || registration.Status || '').toString().toLowerCase();
        let status = statusRaw;
        if (!status) {
            const members = (registration.teamMembers && registration.teamMembers.length > 0) ? registration.teamMembers : (registration.participants && registration.participants.length > 0 ? registration.participants : []);
            status = (members && members.length > 0) ? 'confirmed' : 'pending';
        }
        const statusClass = (status === 'confirmed') ? 'confirmed' : (status === 'cancelled' ? 'cancelled' : 'pending');
        const wrapperClass = `registration-card registration-card--${statusClass}`;

        let detailsHtml = '';
        if (createdRaw) {
            detailsHtml += `
                    <div class="registration-detail">
                        <span class="registration-detail__label">Registration Date:</span>
                        <span class="registration-detail__value">${Utils.formatDate(createdRaw)}</span>
                    </div>`;
        }

        if (registration.registrationId) {
            detailsHtml += `
                    <div class="registration-detail">
                        <span class="registration-detail__label">Registration ID:</span>
                        <span class="registration-detail__value">${registration.registrationId}</span>
                    </div>`;
        }

        const eventId = registration.eventId || registration.eventID || registration.EventID || registration.event_id || registration.event || '';

        return `
            <div class="${wrapperClass}" data-event-id="${escapeHtml(eventId)}">
                <div class="registration-card__header">
                    <h4 class="registration-card__title">${eventName}</h4>
                    <div class="registration-card__status registration-card__status--${statusClass}">${status.toString().toUpperCase()}</div>
                </div>
                <div class="registration-card__details">
                    ${detailsHtml}
                </div>
                ${
                    (registration.teamMembers && registration.teamMembers.length > 0) || (registration.participants && registration.participants.length > 0) ? `
                    <div class="team-members">
                        <h5 class="team-members__title">Team Members:</h5>
                        <div class="team-members__list">
                                ${(() => {
                        const members = (registration.teamMembers && registration.teamMembers.length > 0) ? registration.teamMembers : (registration.participants && registration.participants.length > 0 ? registration.participants : []);
                        const capacity = registration.capacity || registration.Capacity || 1;
                        const rendered = [];
                        for (let i = 0; i < capacity; i++) {
                            const member = members[i];
                            if (member) {
                                rendered.push(`<div class="team-member">${member.name || member.Name || ''} (${member.email || member.Email || ''})</div>`);
                            } else {
                                rendered.push(`<div class="team-member">TBD</div>`);
                            }
                        }
                        return rendered.join('');
                    })()}
                            </div>
                    </div>
                ` : ''}
                <div class="registration-card__actions" style="margin-top:12px; display:flex; gap:8px; justify-content:flex-end;">
                    <button class="btn btn--secondary btn-view-details" data-event-id="${escapeHtml(eventId)}">View Details</button>
                    <button class="btn btn--primary btn-register" data-event-id="${escapeHtml(eventId)}">${status === 'confirmed' ? 'Edit Registration' : 'Register'}</button>
                </div>
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
            try { await ExunServices.api.apiRequest('/auth/logout', { method: 'POST' }); } catch(e) {}
            ExunServices.api.clearAuthToken();
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
