import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Header } from "@/components/Header";
import { apiRequest } from "@/lib/api";
import { Profile } from "@/types";
import { useToast } from "@/hooks/use-toast";
import { Heart, X, MessageCircle } from "lucide-react";

interface Message {
  id: string;
  content: string;
  senderId: number;
  timestamp: string;
}

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

interface LikeResponse {
  isMatch: boolean;
}

const UserProfile = () => {
  const { userId } = useParams();
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  // Check if users are matched
  const { data: matchStatus, isLoading: isMatchLoading } = useQuery<{ isMatched: boolean }>({
    queryKey: ["match-status", userId],
    queryFn: async () => {
      try {
        const matches = await apiRequest('/api/matches');
        const currentUserId = parseInt(localStorage.getItem("user") ? JSON.parse(localStorage.getItem("user") || "{}").id : "0");
        const isMatched = matches?.some(match => 
          ((match.user_id_1 === currentUserId && match.user_id_2 === parseInt(userId!)) ||
           (match.user_id_2 === currentUserId && match.user_id_1 === parseInt(userId!))) &&
          match.status === 'connected'
        );
        return { isMatched: !!isMatched };
      } catch (error) {
        console.error('Error checking match status:', error);
        return { isMatched: false };
      }
    },
    staleTime: 0,
    refetchOnMount: true,
    retry: false
  });

  // Like mutation
  const likeMutation = useMutation({
    mutationFn: async () => {
      const response = await apiRequest(`/api/matches/like/${userId}`, { 
        method: "POST" 
      }) as LikeResponse;
      return response;
    },
    onSuccess: (data) => {
      if (data.isMatch) {
        toast({
          title: "It's a match! 🎉",
          description: "You can now start chatting with this person",
        });
        // Immediately update the cache to show matched state
        queryClient.setQueryData(["is-matched", userId], true);
        // Then invalidate to get fresh data
        queryClient.invalidateQueries({ queryKey: ["is-matched", userId] });
      } else {
        toast({
          title: "Like sent!",
          description: "We'll let you know if they like you back",
        });
      }
      queryClient.invalidateQueries({ queryKey: ["potential-matches"] });
    },
    onError: (error: any) => {
      if (error?.message === "Users are already connected") {
        // Immediately update the cache to show matched state
        queryClient.setQueryData(["is-matched", userId], true);
        toast({
          title: "Already Connected",
          description: "You are already connected with this user",
          variant: "destructive",
        });
      } else {
        toast({
          title: "Error",
          description: error?.message || "Failed to like profile. Please try again.",
          variant: "destructive",
        });
      }
    },
  });
  
  // Dismiss mutation
  const dismissMutation = useMutation({
    mutationFn: () => 
      apiRequest(`/api/matches/dismiss/${userId}`, { method: 'POST' }),
    onSuccess: () => {
      toast({
        title: "Profile dismissed",
        description: "You won't see this profile again",
      });
      queryClient.invalidateQueries({ queryKey: ['potential-matches'] });
      navigate('/matches');
    },
  });

  const { data: profile, isLoading: isProfileLoading } = useQuery<Profile>({
    queryKey: ["profile", userId],
    queryFn: async () => {
      const response = await apiRequest(`/api/users/${userId}/profile`);
      return {
        ...response,
        lookingFor: response.looking_for,
      };
    },
    meta: {
      onError: () => {
        toast({
          variant: "destructive",
          title: "Error",
          description: "Failed to load profile. Please try again later."
        });
      }
    }
  });

  const { data: bioData, isLoading: isBioLoading } = useQuery<{ age: number; location: string }>({
    queryKey: ["bio", userId],
    queryFn: async () => apiRequest(`/api/users/${userId}/bio`),
  });

  const { data: userData, isLoading: isUserLoading } = useQuery<{ name: string; profilePictureUrl: string }>({
    queryKey: ["user", userId],
    queryFn: async () => {
      const response = await apiRequest(`/api/users/${userId}`);
      return {
        name: response.name,  
        profilePictureUrl: response.profile_picture_url,
      };
    },
  });

  if (isProfileLoading || isBioLoading || isUserLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
        <Header />
        <div className="max-w-2xl mx-auto p-8">
          <div>Loading...</div>
        </div>
      </div>
    );
  }

  if (!profile || !bioData || !userData) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
        <Header />
        <div className="max-w-2xl mx-auto p-8">
          <div>User not found</div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
      <Header />
      <div className="max-w-2xl mx-auto p-8">
        <Card className="bg-white">
          <CardHeader>
            <div className="flex items-center gap-4">
              <Avatar className="h-20 w-20">
                <AvatarImage src={userData.profilePictureUrl || "/placeholder.svg"} alt={profile.name} />
                <AvatarFallback>{profile.name?.[0]}</AvatarFallback>
              </Avatar>
              <div>
                <h1 className="text-2xl font-bold">{userData.name}</h1>
                <p className="text-gray-600">Age: {bioData.age ?? "Not provided"}</p>
                <p className="text-gray-600">Location: {bioData.location ?? "Not specified"}</p>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <p className="text-gray-700 mb-4">{profile.bio}</p>
            <h3 className="text-lg font-semibold">Interests</h3>
            <div className="flex flex-wrap gap-2 mt-2">
              {profile.interests?.map((interest) => (
                <span key={interest} className="px-3 py-1 bg-gray-200 rounded-full text-sm">
                  {interest}
                </span>
              ))}
            </div>
            <div className="mt-4">
              <h3 className="text-lg font-semibold">Looking For</h3>
              <p className="text-gray-700">{profile.lookingFor ?? "No preferences specified"}</p>
            </div>
            <div className="mt-4">
              <h3 className="text-lg font-semibold">Occupation</h3>
              <p className="text-gray-700">{profile.occupation}</p>
            </div>
            <div className="flex justify-end gap-2 mt-6">
              {matchStatus?.isMatched && (
                <Button 
                  variant="default"
                  onClick={() => navigate(`/chats?user=${userId}`)}
                  className="bg-purple-500 hover:bg-purple-600"
                >
                  <MessageCircle className="h-4 w-4 mr-2" />
                  Chat History
                </Button>
              )}
              {matchStatus?.isMatched === false && (
                <>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => dismissMutation.mutate()}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="default"
                    size="icon"
                    className="bg-red-500 hover:bg-red-600"
                    onClick={() => likeMutation.mutate()}
                  >
                    <Heart className="h-4 w-4" />
                  </Button>
                </>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default UserProfile;
