import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiRequest } from '@/lib/api';

interface UserStatusProps {
  userId?: number;
  className?: string;
}

export const UserStatus = ({ userId, className = '' }: UserStatusProps) => {
  const { data: status, refetch } = useQuery({
    queryKey: ['user-status', userId],
    queryFn: async () => {
      if (!userId) return { status: 'offline' };
      const response = await apiRequest(`/api/status/${userId}`);
      return response;
    },
    refetchInterval: 5000,
    staleTime: 2000,
    enabled: !!userId,
  });

  useEffect(() => {
    const currentUserId = parseInt(localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')!).id : '0');
    
    if (userId === currentUserId) {
      const updateStatus = async () => {
        try {
          await apiRequest('/api/status/update', { 
            method: 'POST',
            body: JSON.stringify({ status: 'online' }),
            headers: {
              'Content-Type': 'application/json'
            }
          });
          await refetch();
        } catch (error) {
          console.error('Error updating status:', error);
        }
      };

      updateStatus();
      const intervalId = setInterval(updateStatus, 30000);

      const activityEvents = ['mousedown', 'keydown', 'touchstart', 'mousemove', 'scroll', 'click'];
      let activityTimeout: NodeJS.Timeout;

      const handleActivity = () => {
        clearTimeout(activityTimeout);
        updateStatus();
        activityTimeout = setTimeout(updateStatus, 30000);
      };

      activityEvents.forEach(event => {
        window.addEventListener(event, handleActivity);
      });

      const handleBeforeUnload = async () => {
        try {
          await apiRequest('/api/status/update', { 
            method: 'POST',
            body: JSON.stringify({ status: 'offline' }),
            headers: {
              'Content-Type': 'application/json'
            }
          });
        } catch (error) {
          console.error('Error setting offline status:', error);
        }
      };

      window.addEventListener('beforeunload', handleBeforeUnload);

      return () => {
        clearInterval(intervalId);
        clearTimeout(activityTimeout);
        activityEvents.forEach(event => {
          window.removeEventListener(event, handleActivity);
        });
        window.removeEventListener('beforeunload', handleBeforeUnload);
        handleBeforeUnload();
      };
    }
  }, [userId, refetch]);

  return (
    <div className={`inline-flex items-center gap-2 ${className}`}>
      <div 
        className={`w-2 h-2 rounded-full ${
          status?.status === 'online' ? 'bg-green-500' : 'bg-gray-400'
        }`}
      />
      <span className="text-sm text-gray-600">
        {status?.status === 'online' ? 'Online' : 'Offline'}
      </span>
    </div>
  );
};