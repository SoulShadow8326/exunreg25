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
    return !!localStorage.getItem('authToken');
}

function getCurrentUser() {
    const token = localStorage.getItem('authToken');
    if (!token) return null;
    
    try {
        const payload = JSON.parse(atob(token.split('.')[1]));
        return payload;
    } catch (error) {
        console.error('Invalid token:', error);
        localStorage.removeItem('authToken');
        return null;
    }
}

function isAdmin() {
    const user = getCurrentUser();
    return user && user.email === 'exun@dpsrkp.net';
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
