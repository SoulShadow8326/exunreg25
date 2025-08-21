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
                        window.location.href = '/complete_signup';
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
        this.updateAdminLinks();
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
            authContainer.innerHTML = `
                <button class="btn btn--primary navbar__link" data-action="login">Login</button>
            `;
        }
    }

    updateAdminLinks() {
        const adminContainer = document.querySelector('[data-nav="admin"]');
        if (!adminContainer) return;

        if (this.currentUser && ExunServices.api.isAdmin()) {
            adminContainer.innerHTML = `
                <a href="/admin" class="navbar__link">Admin</a>
            `;
        } else {
            adminContainer.innerHTML = '';
        }
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
        toggleBtn.innerHTML = '☰';
        toggleBtn.setAttribute('aria-label', 'Toggle navigation');
        
        let isOpen = false;
        
        toggleBtn.addEventListener('click', () => {
            isOpen = !isOpen;
            nav.style.display = isOpen ? 'flex' : 'none';
            toggleBtn.innerHTML = isOpen ? '✕' : '☰';
        });

        navbar.insertBefore(toggleBtn, nav);
        nav.style.display = 'none';

        window.addEventListener('resize', () => {
            if (!Utils.isMobile()) {
                nav.style.display = 'flex';
                toggleBtn.style.display = 'none';
            } else {
                toggleBtn.style.display = 'block';
                nav.style.display = isOpen ? 'flex' : 'none';
            }
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
