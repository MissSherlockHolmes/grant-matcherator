// Helper function to get the stored token
export const getAuthToken = () => {
  const token = localStorage.getItem('token');
  console.log('=== Auth Token Check ===');
  console.log('Token present:', !!token);
  return token;
};

// Helper function to check if user is authenticated
export const isAuthenticated = () => {
  console.log('=== Authentication Check ===');
  const isAuth = !!getAuthToken();
  console.log('Is authenticated:', isAuth);
  return isAuth;
};

// Get the API base URL from environment variable or use default
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

// Base API request function that includes auth header
export const apiRequest = async (endpoint: string, options: RequestInit = {}) => {
  console.log('=== API Request ===');
  console.log('Endpoint:', endpoint);
  console.log('Options:', options);

  const token = getAuthToken();
  
  if (!token) {
    console.error('No authentication token found');
    throw new Error('No authentication token found');
  }

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
    ...options.headers,
  };
  console.log('Request headers:', headers);

  // Remove /api prefix if it exists since we're adding it here
  const cleanEndpoint = endpoint.startsWith('/api') ? endpoint.slice(4) : endpoint;
  const fullUrl = `${API_BASE_URL}/api${cleanEndpoint}`;
  console.log('Full URL:', fullUrl);

  try {
    console.log('Making request to:', fullUrl);
    const response = await fetch(fullUrl, {
      ...options,
      headers,
    });
    console.log('Response status:', response.status);
    console.log('Response status text:', response.statusText);
    console.log('Response headers:', Object.fromEntries(response.headers.entries()));

    if (!response.ok) {
      console.error('API request failed:', {
        status: response.status,
        statusText: response.statusText,
        headers: Object.fromEntries(response.headers.entries())
      });
      throw new Error(`API request failed: ${response.statusText}`);
    }

    // Check if response is empty
    const text = await response.text();
    console.log('Raw response text:', text);

    if (!text) {
      console.log('Empty response received');
      return null;
    }

    try {
      const parsedResponse = JSON.parse(text);
      console.log('Parsed response:', parsedResponse);
      return parsedResponse;
    } catch (error) {
      console.error('Error parsing JSON response:', {
        error,
        text,
        message: error instanceof Error ? error.message : 'Unknown error'
      });
      throw new Error('Invalid JSON response from server');
    }
  } catch (error) {
    console.error('API request error:', {
      error,
      message: error instanceof Error ? error.message : 'Unknown error',
      stack: error instanceof Error ? error.stack : undefined
    });
    throw error;
  }
};