import { useQuery } from "@tanstack/react-query";
import { apiRequest } from "@/lib/api";

interface UserStatusProps {
  userId?: number;
}

export const UserStatus = ({ userId }: UserStatusProps) => {
  const { data: status } = useQuery({
    queryKey: ['userStatus', userId],
    queryFn: async () => {
      if (!userId) return { online: false };
      const response = await apiRequest(`/users/${userId}/status`);
      return response;
    },
    enabled: !!userId,
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  return (
    <div className="flex items-center gap-1 text-xs text-muted-foreground">
      <div className={`w-2 h-2 rounded-full ${status?.online ? 'bg-green-500' : 'bg-gray-400'}`} />
      <span>{status?.online ? 'Online' : 'Offline'}</span>
    </div>
  );
};