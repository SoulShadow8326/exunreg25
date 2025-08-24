async function apiRequest(endpoint, options = {}) {
    let url = endpoint || '';
    if (!url.startsWith('/')) url = '/' + url;
    if (!url.startsWith('/api/')) url = '/api' + url;
    const config = {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        },
        ...options
    };

    if (!config.credentials) {
        config.credentials = 'include'
    }

    const token = localStorage.getItem('authToken');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }

    try {
    const response = await fetch(url, config);
        
        const contentType = response.headers.get('content-type');
        if (!response.ok) {
            if (contentType && contentType.includes('application/json')) {
                const errJson = await response.json();
                const msg = errJson.error || errJson.message || `HTTP error! status: ${response.status}`;
                throw new Error(msg);
            }
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        if (contentType && contentType.includes('application/json')) {
            return await response.json();
        }
        return await response.text();
    } catch (error) {
        console.error('API Request Error:', error);
        throw error;
    }
}

function setAuthToken(token) {
    localStorage.setItem('authToken', token);
}

function clearAuthToken() {
    localStorage.removeItem('authToken');
}

function isAuthenticated() {
    const cookieValue = document.cookie.split(';').map(c=>c.trim()).find(c=>c.startsWith('auth_token='));
    if (cookieValue) return true;
    return !!localStorage.getItem('authToken');
}

function getCurrentUser() {
    const token = localStorage.getItem('authToken');
    if (!token) return null;
    function base64UrlDecode(input) {
        let str = input.replace(/-/g, '+').replace(/_/g, '/');
        while (str.length % 4) {
            str += '=';
        }
        return atob(str);
    }
    if (token.indexOf('.') === -1) {
        return null;
    }

    try {
        const parts = token.split('.');
        if (parts.length < 2) throw new Error('token is not a JWT');
        const payloadBase64 = parts[1];
        const payloadJson = base64UrlDecode(payloadBase64);
        const payload = JSON.parse(payloadJson);
        return payload;
    } catch (error) {
        console.error('Invalid JWT token:', error);
        try { localStorage.removeItem('authToken'); } catch(e) {}
        return null;
    }
}

function isAdmin() {
    const user = getCurrentUser();
    if (!user || !user.email) return false;
    if (!window.__ADMIN_EMAILS) {
        apiRequest('/admin/config').then(resp => {
            if (resp && resp.data && resp.data.admin_emails) {
                window.__ADMIN_EMAILS = resp.data.admin_emails.split(',').map(s => s.trim().toLowerCase());
            } else if (resp && resp.admin_emails) {
                window.__ADMIN_EMAILS = resp.admin_emails.split(',').map(s => s.trim().toLowerCase());
            }
        }).catch(() => {
            window.__ADMIN_EMAILS = ['exun@dpsrkp.net'];
        });
        return false;
    }
    return window.__ADMIN_EMAILS.indexOf(user.email.toLowerCase()) !== -1;
}

window.ExunServices = {
    api: {
        apiRequest,
        setAuthToken,
        clearAuthToken,
        isAuthenticated,
        getCurrentUser,
        isAdmin
    }
};

window.ExunServices.events = {
    getAllEvents: function() {
        return window.ExunServices.api.apiRequest('/events');
    }
};

window.ExunServices.admin = {
    getUserDetails: async function(search) {
        if (search && search.trim() !== '') {
            const resp = await window.ExunServices.api.apiRequest(`/admin/users?email=${encodeURIComponent(search)}`);
            return { users: [resp] };
        }
        const resp = await window.ExunServices.api.apiRequest('/admin/export?type=users');
        return { users: resp };
    },
    getEventRegistrations: async function(eventId) {
        if (!eventId) {
            return { registrations: [] };
        }
        const resp = await window.ExunServices.api.apiRequest(`/admin/event-registrations?event_id=${encodeURIComponent(eventId)}`);
        return { registrations: resp };
    },
    updateEvent: function(eventData) {
        return window.ExunServices.api.apiRequest('/admin/events', { method: 'POST', body: JSON.stringify(eventData) });
    },
};

window.ExunServices.getCookie = function(name) {
    const val = document.cookie.split(';').map(c=>c.trim()).find(c=>c.startsWith(name + '='));
    if (!val) return null;
    return decodeURIComponent(val.split('=')[1] || '');
}
