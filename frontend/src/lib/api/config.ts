import axios from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

// Create axios instance with default config
export const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add request interceptor to add auth token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Add response interceptor to handle errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/';
    }
    return Promise.reject(error);
  }
);

// Auth service
export const auth = {
  login: async (data: { email: string; password: string }) => {
    const response = await api.post('/auth/login', data);
    return response.data;
  },
  signup: async (data: { email: string; password: string; role: 'provider' | 'recipient' }) => {
    const response = await api.post('/auth/signup', data);
    return response.data;
  },
};

// User service
export const user = {
  getProfile: async () => {
    const response = await api.get('/me/profile');
    return response.data;
  },
  updateProfile: async (data: any) => {
    const response = await api.put('/me/profile', data);
    return response.data;
  },
  getBio: async () => {
    const response = await api.get('/me/bio');
    return response.data;
  },
  updateBio: async (data: { bio: string }) => {
    const response = await api.put('/me/bio', data);
    return response.data;
  },
  getUser: async (id: number) => {
    const response = await api.get(`/users/${id}`);
    return response.data;
  },
  getUserProfile: async (id: number) => {
    const response = await api.get(`/users/${id}/profile`);
    return response.data;
  },
  getUserBio: async (id: number) => {
    const response = await api.get(`/users/${id}/bio`);
    return response.data;
  },
};

// Chat service
export const chat = {
  getChats: async () => {
    const response = await api.get('/chat');
    return response.data;
  },
  getMessages: async (chatId: number) => {
    const response = await api.get(`/chat/${chatId}/messages`);
    return response.data;
  },
  markMessagesRead: async (chatId: number) => {
    const response = await api.post(`/chat/${chatId}/messages/read`);
    return response.data;
  },
};

// Connection service
export const connection = {
  getConnections: async () => {
    const response = await api.get('/connections');
    return response.data;
  },
  getPotentialMatches: async () => {
    const response = await api.get('/potential-matches');
    return response.data;
  },
  recalculateMatches: async () => {
    const response = await api.post('/potential-matches/recalculate');
    return response.data;
  },
  requestConnection: async (userId: number) => {
    const response = await api.post(`/connections/${userId}/request`);
    return response.data;
  },
  acceptConnection: async (userId: number) => {
    const response = await api.post(`/connections/${userId}/accept`);
    return response.data;
  },
  rejectConnection: async (userId: number) => {
    const response = await api.post(`/connections/${userId}/reject`);
    return response.data;
  },
};

// Notification service
export const notification = {
  getNotifications: async () => {
    const response = await api.get('/notifications');
    return response.data;
  },
  markAsRead: async (notificationId: number) => {
    const response = await api.post(`/notifications/${notificationId}/read`);
    return response.data;
  },
}; 