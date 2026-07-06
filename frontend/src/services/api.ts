import axios from 'axios';
import { useAuthStore } from '../stores/auth';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_URL,
});

api.interceptors.request.use((config) => {
  const { token, activeOrgId } = useAuthStore.getState();
  
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  
  if (activeOrgId) {
    config.headers['X-Organization-ID'] = activeOrgId;
  }
  
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      useAuthStore.getState().logout();
      // Only redirect if we are not already on login or register page
      const path = window.location.pathname;
      if (path !== '/login' && path !== '/cadastro' && path !== '/' && path !== '/precos') {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);
