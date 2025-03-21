import { useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ScrollArea } from "./ui/scroll-area";
import { Card } from "./ui/card";
import { Button } from "./ui/button";
import { useToast } from "./ui/use-toast";
import { apiRequest } from "@/lib/api";
import { Link } from "react-router-dom";
import { UserMinus } from "lucide-react";
import { UserStatus } from './UserStatus';

interface Match {
  id: number;
  user_id_1: number;
  user_id_2: number;
  status: string;
  created_at: string;
  updated_at: string;
  other_user_name: string;
  other_user_picture: string;
}

export const Matches = () => {
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const { data: matches, isLoading } = useQuery({
    queryKey: ['matches'],
    queryFn: async () => {
      const response = await apiRequest('/api/matches');
      return response as Match[];
    },
  });

  const markNotificationsAsRead = useMutation({
    mutationFn: () => apiRequest('/api/notifications/mark-matches-read', { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  useEffect(() => {
    apiRequest('/api/notifications/mark-matches-read', { method: 'POST' });
  }, []);

  const disconnectMutation = useMutation({
    mutationFn: (matchId: number) => 
      apiRequest(`/api/matches/disconnect/${matchId}`, { method: 'POST' }),
    onSuccess: () => {
      toast({
        title: "Connection removed",
        description: "You've disconnected from this match",
      });
      queryClient.invalidateQueries({ queryKey: ['matches'] });
    },
    onError: () => {
      toast({
        title: "Error",
        description: "Failed to disconnect. Please try again.",
        variant: "destructive",
      });
    },
  });

  if (isLoading) {
    return <div>Loading matches...</div>;
  }

  return (
    <ScrollArea className="h-[600px] w-full rounded-md border p-4">
      <div>
        <h3 className="text-lg font-semibold mb-4">Connected Matches</h3>
        <div className="space-y-4">
        {matches && matches.length > 0 ? (
  matches.map((match) => {
    const userId = parseInt(JSON.parse(localStorage.getItem("user") || "{}").id);
    const otherUserId = match.user_id_1 === userId ? match.user_id_2 : match.user_id_1;

    return (
      <Card key={match.id} className="p-4 hover:bg-gray-100 transition">
        <div className="flex items-center justify-between">
          <Link to={`/users/${otherUserId}`} className="flex items-center gap-4">
            {match.other_user_picture && (
              <img
                src={match.other_user_picture}
                alt={match.other_user_name}
                className="h-12 w-12 rounded-full object-cover"
              />
            )}
            <div className="flex flex-col">
              <span className="font-medium">{match.other_user_name}</span>
              <UserStatus userId={otherUserId} />
            </div>
          </Link>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => disconnectMutation.mutate(match.id)}
            >
              <UserMinus className="h-4 w-4" />
            </Button>
            <Link 
              to={`/chats?user=${otherUserId}`}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
            >
              Chat
            </Link>
          </div>
        </div>
      </Card>
    );
  })
) : (
  <div className="text-center text-muted-foreground">
    No matches yet. Keep looking!
  </div>
)}
        </div>
      </div>
    </ScrollArea>
  );
};
