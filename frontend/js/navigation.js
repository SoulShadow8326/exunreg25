class Navigation {
    constructor() {
        this.currentUser = null;
        this.init();
    }

    async init() {
    this.setupEventListeners();
    await this.loadUserState();
    this.updateNavigation();
    }

    async loadUserState() {
        try {
            if (ExunServices.api.isAuthenticated()) {
                const response = await ExunServices.api.apiRequest('/auth/profile');
                if (response && response.status === 'success') {
                    this.currentUser = response.data;
                } else {
                    if (response && response.error === 'Complete signup required') {
                        window.location.href = '/complete';
                        return;
                    }
                    ExunServices.api.clearAuthToken();
                    window.location.href = '/login';
                    return;
                }
            }
        } catch (error) {
            console.error('Failed to load user state:', error);
            ExunServices.api.clearAuthToken();
        }
    }

    updateNavigation() {
        this.updateAuthLinks();
        this.updateActiveLink();
    }

    updateAuthLinks() {
        const authContainer = document.querySelector('[data-nav="auth"]');
        if (!authContainer) return;

        if (this.currentUser) {
            authContainer.innerHTML = `
                <a href="/summary" class="navbar__link">Registration Summary</a>
                <button class="btn btn--primary navbar__link" data-action="logout">Logout</button>
            `;
        } else {
            const trimmed = authContainer.textContent ? authContainer.textContent.trim() : '';
            if (!trimmed) {
                authContainer.innerHTML = `
                    <button class="btn btn--primary navbar__link" data-action="login">Login</button>
                `;
            }
        }
    }

    updateAdminLinks() {
    }

    updateActiveLink() {
        const navLinks = document.querySelectorAll('.navbar__link');
        let currentPath = window.location.pathname;
        if (currentPath === '/' || currentPath === '/index.html') {
            currentPath = '/';
        }
        navLinks.forEach(link => {
            link.classList.remove('navbar__link--active');
            let linkPath = link.getAttribute('href');
            if (linkPath === '/' || linkPath === '/index.html') {
                linkPath = '/';
            }
            if (linkPath === currentPath) {
                link.classList.add('navbar__link--active');
            }
        });
    }

    setupEventListeners() {
        const logoutBtn = document.querySelector('[data-action="logout"]');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', async (e) => {
                e.preventDefault();
                await this.handleLogout();
            });
        }

        const loginBtn = document.querySelector('[data-action="login"]');
        if (loginBtn) {
            loginBtn.addEventListener('click', (e) => {
                e.preventDefault();
                window.location.href = '/login';
            });
        }

        if (!this._delegatedClickAttached) {
            this._delegatedClickAttached = true;
            document.addEventListener('click', (e) => {
                const loginTarget = e.target.closest('[data-action="login"]');
                if (loginTarget) {
                    e.preventDefault();
                    window.location.href = '/login';
                    return;
                }
                const logoutTarget = e.target.closest('[data-action="logout"]');
                if (logoutTarget) {
                    e.preventDefault();
                    try {
                        this.handleLogout();
                    } catch (err) {
                        console.error('Delegated logout failed', err);
                    }
                }
            });
        }

        this.setupMobileMenu();
    }

    setupMobileMenu() {
        if (!Utils.isMobile()) return;

        const navbar = document.querySelector('.navbar');
        const nav = document.querySelector('.navbar__nav');
        
        if (!navbar || !nav) return;

        const toggleBtn = document.createElement('button');
        toggleBtn.className = 'navbar__toggle';
        toggleBtn.setAttribute('aria-label', 'Toggle navigation');
        toggleBtn.setAttribute('aria-expanded', 'false');

        const svgMenu = `
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="24px" height="24px" viewBox="0 0 24 24" version="1.1" aria-hidden="true">
    <title>Menu</title>
    <g stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <g>
            <rect fill-rule="nonzero" x="0" y="0" width="24" height="24"></rect>
            <line x1="5" y1="7" x2="19" y2="7" stroke="#0C0310" stroke-width="2" stroke-linecap="round"></line>
            <line x1="5" y1="12" x2="19" y2="12" stroke="#0C0310" stroke-width="2" stroke-linecap="round"></line>
            <line x1="5" y1="17" x2="19" y2="17" stroke="#0C0310" stroke-width="2" stroke-linecap="round"></line>
        </g>
    </g>
</svg>`;

        let isOpen = false;

        toggleBtn.innerHTML = svgMenu;

        nav.setAttribute('aria-hidden', 'true');
        nav.classList.remove('open');

        const openMenu = () => {
            isOpen = true;
            nav.classList.add('open');
            nav.setAttribute('aria-hidden', 'false');
            toggleBtn.setAttribute('aria-expanded', 'true');
            toggleBtn.innerHTML = '✕';
            document.addEventListener('click', outsideClickListener);
            document.addEventListener('keydown', escapeListener);
        };

        const closeMenu = () => {
            isOpen = false;
            nav.classList.remove('open');
            nav.setAttribute('aria-hidden', 'true');
            toggleBtn.setAttribute('aria-expanded', 'false');
            toggleBtn.innerHTML = svgMenu;
            document.removeEventListener('click', outsideClickListener);
            document.removeEventListener('keydown', escapeListener);
        };

        const outsideClickListener = (e) => {
            if (!navbar.contains(e.target)) {
                closeMenu();
            }
        };

        const escapeListener = (e) => {
            if (e.key === 'Escape' || e.key === 'Esc') {
                closeMenu();
            }
        };

        toggleBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (isOpen) closeMenu(); else openMenu();
        });

        navbar.appendChild(toggleBtn);
        const handleResize = () => {
            if (!Utils.isMobile()) {
                nav.classList.remove('open');
                nav.removeAttribute('aria-hidden');
                toggleBtn.style.display = 'none';
                toggleBtn.setAttribute('aria-expanded', 'false');
            } else {
                nav.classList.remove('open');
                nav.setAttribute('aria-hidden', 'true');
                toggleBtn.style.display = 'block';
                toggleBtn.setAttribute('aria-expanded', 'false');
            }
        };

        handleResize();

        window.addEventListener('resize', () => {
            handleResize();
        });
    }

    async handleLogout() {
        try {
            Utils.setLoading(document.querySelector('[data-action="logout"]'), true);
            await ExunServices.api.apiRequest('/auth/logout', { method: 'POST' });
            ExunServices.api.clearAuthToken();
            Utils.showToast('Logged out successfully', 'success');
            setTimeout(() => window.location.href = '/', 1000);
        } catch (error) {
            console.error('Logout failed:', error);
            Utils.showToast('Logout failed', 'error');
        } finally {
            Utils.setLoading(document.querySelector('[data-action="logout"]'), false);
        }
    }

    getUserState() {
        return {
            isAuthenticated: !!this.currentUser,
            isAdmin: ExunServices.api.isAdmin(),
            user: this.currentUser
        };
    }
}

window.Navigation = Navigation;

document.addEventListener('DOMContentLoaded', () => {
    const tryInit = () => {
        if (document.querySelector('[data-nav="auth"]')) {
            window.nav = new Navigation();
        } else {
            setTimeout(tryInit, 50);
        }
    };
    tryInit();
});
