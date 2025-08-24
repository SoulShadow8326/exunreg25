document.addEventListener('DOMContentLoaded', () => {
  const form = document.getElementById('complete-form');
  const message = document.getElementById('message');
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    message.textContent = '';
    const username = document.getElementById('username').value.trim();
    if (!username) {
      message.textContent = 'Username is required';
      return;
    }
    try {
  const resp = await ExunServices.api.apiRequest('/auth/complete', {
        method: 'POST',
        body: JSON.stringify({ username })
      });
      if (resp.status === 'success') {
        message.textContent = 'Signup complete. Redirecting to profile completion...';
        setTimeout(() => window.location.href = '/signup', 800);
      } else {
        message.textContent = resp.error || 'Failed to complete signup';
      }
    } catch (err) {
      message.textContent = err.message || 'Error completing signup';
    }
  });
});
