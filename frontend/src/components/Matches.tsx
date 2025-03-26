import { useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ScrollArea } from "./ui/scroll-area";
import { Card } from "./ui/card";
import { Button } from "./ui/button";
import { useToast } from "./ui/use-toast";
import { apiRequest } from "@/lib/api";
import { Link, useNavigate } from "react-router-dom";
import { UserMinus } from "lucide-react";
import { UserStatus } from './UserStatus';
import { Connection } from "@/types";

export const Matches = () => {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { data: matches, isLoading } = useQuery({
    queryKey: ['connections'],
    queryFn: async () => {
      console.log('Fetching connections...');
      const userStr = localStorage.getItem("user");
      console.log('Current user data from localStorage:', userStr);
      
      if (!userStr) {
        console.error('No user data found in localStorage. User may not be logged in.');
        throw new Error('User not logged in');
      }

      let currentUser = null;
      try {
        currentUser = JSON.parse(userStr);
        console.log('Successfully parsed user data:', currentUser);
      } catch (parseError) {
        console.error('Error parsing user data:', parseError);
        localStorage.removeItem("user");
        throw new Error('Invalid user data');
      }

      if (!currentUser || !currentUser.id) {
        console.error('Invalid user data structure:', currentUser);
        localStorage.removeItem("user");
        throw new Error('Invalid user data structure');
      }

      const response = await apiRequest('/connections');
      console.log('Raw API response:', response);
      return response as Connection[];
    },
    retry: false,
    staleTime: 0,
    refetchOnMount: true
  });

  const markNotificationsAsRead = useMutation({
    mutationFn: () => apiRequest('/notifications/mark-matches-read', { method: 'POST' }),
    onSuccess: () => {
      console.log('Successfully marked matches notifications as read');
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  useEffect(() => {
    console.log('Marking matches notifications as read on component mount');
    apiRequest('/notifications/mark-matches-read', { method: 'POST' });
  }, []);

  const disconnectMutation = useMutation({
    mutationFn: (targetId: number) => 
      apiRequest(`/connections/${targetId}`, { method: 'DELETE' }),
    onSuccess: () => {
      console.log('Successfully disconnected from match');
      toast({
        title: "Connection removed",
        description: "You've disconnected from this match",
      });
      queryClient.invalidateQueries({ queryKey: ['connections'] });
    },
    onError: (error) => {
      console.error('Error disconnecting from match:', error);
      toast({
        title: "Error",
        description: "Failed to disconnect. Please try again.",
        variant: "destructive",
      });
    },
  });

  if (isLoading) {
    console.log('Loading connections...');
    return <div>Loading connections...</div>;
  }

  console.log('Rendering connections:', matches);

  return (
    <ScrollArea className="h-[600px] w-full rounded-md border p-4">
      <div>
        <h3 className="text-lg font-semibold mb-4">Connected Matches</h3>
        <div className="space-y-4">
        {matches && matches.length > 0 ? (
          matches.map((match) => {
            const userId = parseInt(JSON.parse(localStorage.getItem("user") || "{}").id);
            const otherUserId = match.initiator_id === userId ? match.target_id : match.initiator_id;

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
                    <Button
                      variant="default"
                      onClick={async () => {
                        try {
                          // Check chat preferences
                          const preferences = await apiRequest('/chat/preferences');
                          if (!preferences.opt_in) {
                            // Enable chat
                            await apiRequest('/chat/preferences', {
                              method: 'PUT',
                              body: JSON.stringify({ opt_in: true })
                            });
                          }
                          navigate(`/chats/${match.id}`);
                        } catch (error) {
                          console.error('Error enabling chat:', error);
                          toast({
                            title: "Error",
                            description: "Failed to enable chat. Please try again.",
                            variant: "destructive",
                          });
                        }
                      }}
                      className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                    >
                      Chat
                    </Button>
                  </div>
                </div>
              </Card>
            );
          })
        ) : (
          <div className="text-center text-muted-foreground">
            No connections yet. Keep looking!
          </div>
        )}
        </div>
      </div>
    </ScrollArea>
  );
};
