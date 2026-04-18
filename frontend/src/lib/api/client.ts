import axios from 'axios';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api'
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// On 401, clear storage and redirect to login so the user gets a fresh token.
// On 403, attach a human-readable hint to the error so the UI can display it.
api.interceptors.response.use(
  res => res,
  err => {
    const status = err?.response?.status;
    if (status === 401) {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
      window.location.href = '/login';
    }
    if (status === 403) {
      err.uiMessage = 'Permission denied. If you are an admin, your session may be stale — please sign out and sign back in.';
    }
    return Promise.reject(err);
  }
);

