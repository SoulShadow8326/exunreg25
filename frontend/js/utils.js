function isMobile() {
    return window.innerHeight > window.innerWidth || window.innerWidth <= 768;
}
function isMobile() {
    return window.innerHeight > window.innerWidth || window.innerWidth <= 768;
}

async function loadComponent(componentPath, targetElement) {
    try {
        const response = await fetch(componentPath);
        const html = await response.text();
        if (targetElement) {
            targetElement.innerHTML = html;
        }
        return html;
    } catch (error) {
        console.error('Error loading component:', error);
        return '';
    }
}
    window.Utils = {
        isMobile,
        loadComponent,
        formatDate,
        showToast,
        debounce,
        validateEmail,
        validatePhone,
        sanitizeHTML,
        copyToClipboard,
        generateRandomId,
        formatFileSize,
        throttle,
        handleFormSubmit,
        setLoading,
        redirect,
        formatEventMode,
        formatParticipants,
        formatEligibility
    };

function formatDate(date) {
    return new Date(date).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric'
    });
}

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast toast--${type}`;
    toast.textContent = message;
    
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.classList.add('toast--show');
    }, 100);

    setTimeout(() => {
        toast.classList.remove('toast--show');
        setTimeout(() => {
            document.body.removeChild(toast);
        }, 300);
    }, 3000);
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function validateEmail(email) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
}

function validatePhone(phone) {
    const phoneRegex = /^\+?[\d\s\-\(\)]{10,}$/;
    return phoneRegex.test(phone);
}

function sanitizeHTML(str) {
    const temp = document.createElement('div');
    temp.textContent = str;
    return temp.innerHTML;
}

async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        showToast('Copied to clipboard!', 'success');
    } catch (error) {
        console.error('Failed to copy:', error);
        showToast('Failed to copy to clipboard', 'error');
    }
}

function generateRandomId() {
    return Math.random().toString(36).substr(2, 9);
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function throttle(func, limit) {
    let inThrottle;
    return function() {
        const args = arguments;
        const context = this;
        if (!inThrottle) {
            func.apply(context, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

async function handleFormSubmit(form, submitHandler) {
    const formData = new FormData(form);
    const data = Object.fromEntries(formData.entries());
    
    try {
        await submitHandler(data);
    } catch (error) {
        console.error('Form submission error:', error);
        showToast(error.message || 'An error occurred', 'error');
    }
}

function setLoading(element, isLoading) {
    if (isLoading) {
        element.disabled = true;
        element.classList.add('loading');
        element.dataset.originalText = element.textContent;
        element.textContent = 'Loading...';
    } else {
        element.disabled = false;
        element.classList.remove('loading');
        element.textContent = element.dataset.originalText || element.textContent;
    }
}

function redirect(url, delay = 0) {
    setTimeout(() => {
        window.location.href = url;
    }, delay);
}

function formatEventMode(mode) {
    if (!mode || typeof mode !== 'string') return '';
    return mode.charAt(0).toUpperCase() + mode.slice(1).toLowerCase();
}

function formatParticipants(count) {
    if (count === 1) return '1 participant';
    return `${count} participants`;
}

function formatEligibility(eligibility, openToAll) {
    if (openToAll) return 'Open to All';
    if (eligibility && eligibility.length === 2) {
        return `${eligibility[0]}th - ${eligibility[1]}th`;
    }
    return 'All Classes';
}
