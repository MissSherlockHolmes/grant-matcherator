import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Header } from "@/components/Header";
import { apiRequest } from "@/lib/api";
import { Profile, RecipientData, ProviderData } from "@/types";
import { useToast } from "@/hooks/use-toast";
import { Handshake, X, MessageCircle } from "lucide-react";

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
        console.log('Fetching match status for user ID:', userId);
        const connections = await apiRequest('/connections');
        console.log('Raw API response for connections:', connections);
        const currentUserId = parseInt(localStorage.getItem("user") ? JSON.parse(localStorage.getItem("user") || "{}").id : "0");
        console.log('Current user ID:', currentUserId);
        console.log('Target user ID:', userId);
        
        const isMatched = connections?.some(conn => {
          const isMatch = ((conn.initiator_id === currentUserId && conn.target_id === parseInt(userId!)) ||
                         (conn.target_id === currentUserId && conn.initiator_id === parseInt(userId!)));
          console.log('Checking connection:', {
            connection: conn,
            isMatch,
            initiator: conn.initiator_id,
            target: conn.target_id
          });
          return isMatch;
        });
        
        console.log('Final match status:', isMatched);
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

  const connectMutation = useMutation({
    mutationFn: async () => {
      console.log('=== Connect Action Start ===');
      console.log(`Attempting to connect with user ${userId}`);
      try {
        const response = await apiRequest('/connections', { 
          method: "POST",
          body: JSON.stringify({ target_id: parseInt(userId!) })
        });
        console.log('Connect API Response:', {
          status: 'success',
          data: response
        });
        return response;
      } catch (error) {
        console.error('Connect API Error:', {
          error,
          message: error instanceof Error ? error.message : 'Unknown error',
          stack: error instanceof Error ? error.stack : undefined
        });
        throw error;
      }
    },
    onSuccess: () => {
      console.log('Connect Action Success');
      toast({
        title: "Connected!",
        description: "You are now connected with this organization",
      });
      // Optimistically update the UI by removing the connected match
      queryClient.setQueryData(['potential-matches'], (oldData: any) => {
        if (!oldData) return oldData;
        return oldData.filter((match: any) => match.id !== parseInt(userId!));
      });
      // Update the match status optimistically
      queryClient.setQueryData(['match-status', userId], { isMatched: true });
      // Then invalidate to ensure we have the latest data
      queryClient.invalidateQueries({ queryKey: ["potential-matches"] });
      queryClient.invalidateQueries({ queryKey: ["connections"] });
      queryClient.invalidateQueries({ queryKey: ["match-status", userId] });
      console.log('Queries invalidated, waiting for refetch');
    },
    onError: (error) => {
      console.error('Connect Action Error:', {
        error,
        message: error instanceof Error ? error.message : 'Unknown error'
      });
      toast({
        title: "Error",
        description: "Failed to connect. Please try again.",
        variant: "destructive",
      });
    },
  });
  
  const dismissMutation = useMutation({
    mutationFn: async () => {
      console.log('=== Dismiss Action Start ===');
      console.log(`Attempting to dismiss user ${userId}`);
      try {
        const response = await apiRequest(`/matches/dismiss/${userId}`, { 
          method: 'DELETE' 
        });
        console.log('Dismiss API Response:', {
          status: 'success',
          data: response
        });
        return response;
      } catch (error) {
        console.error('Dismiss API Error:', {
          error,
          message: error instanceof Error ? error.message : 'Unknown error',
          stack: error instanceof Error ? error.stack : undefined
        });
        throw error;
      }
    },
    onSuccess: () => {
      console.log('Dismiss Action Success');
      toast({
        title: "Profile dismissed",
        description: "You won't see this profile again",
      });
      // Optimistically update the UI by removing the dismissed match
      queryClient.setQueryData(['potential-matches'], (oldData: any) => {
        if (!oldData) return oldData;
        return oldData.filter((match: any) => match.id !== parseInt(userId!));
      });
      // Then invalidate to ensure we have the latest data
      queryClient.invalidateQueries({ queryKey: ['potential-matches'] });
      navigate('/matches');
    },
    onError: (error) => {
      console.error('Dismiss Action Error:', {
        error,
        message: error instanceof Error ? error.message : 'Unknown error'
      });
      toast({
        title: "Error",
        description: "Failed to dismiss profile. Please try again.",
        variant: "destructive",
      });
    },
  });

  const { data: profile, isLoading: isProfileLoading } = useQuery<Profile>({
    queryKey: ["profile", userId],
    queryFn: async () => {
      console.log('Fetching user profile for ID:', userId);
      const response = await apiRequest(`/api/users/${userId}/profile`);
      console.log('Raw API response for profile:', response);
      return response;
    },
    meta: {
      onError: (error) => {
        console.error('Error fetching user profile:', error);
        toast({
          variant: "destructive",
          title: "Error",
          description: "Failed to load profile. Please try again later."
        });
      }
    }
  });

  const { data: recipientData, isLoading: isRecipientLoading } = useQuery<RecipientData>({
    queryKey: ["recipient-data", userId],
    queryFn: async () => {
      console.log('Fetching recipient data for ID:', userId);
      const response = await apiRequest(`/api/users/${userId}/recipient-data`);
      console.log('Raw API response for recipient data:', response);
      return response;
    },
    enabled: !!userId,
  });

  const { data: providerData, isLoading: isProviderLoading } = useQuery<ProviderData>({
    queryKey: ["provider-data", userId],
    queryFn: async () => {
      console.log('Fetching provider data for ID:', userId);
      const response = await apiRequest(`/api/users/${userId}/provider-data`);
      console.log('Raw API response for provider data:', response);
      return response;
    },
    enabled: !!userId,
  });

  if (isProfileLoading || isRecipientLoading || isProviderLoading || isMatchLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
        <Header />
        <div className="max-w-2xl mx-auto p-8">
          <div>Loading...</div>
        </div>
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
        <Header />
        <div className="max-w-2xl mx-auto p-8">
          <div>Organization not found</div>
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
            <div className="flex flex-col items-center space-y-4">
              <div className="relative">
                <Avatar className="h-32 w-32">
                  <AvatarImage
                    src={profile.profile_picture_url || "/placeholder.svg"}
                    alt="Profile"
                  />
                  <AvatarFallback>👤</AvatarFallback>
                </Avatar>
              </div>
              <div className="text-center">
                <h1 className="text-2xl font-bold">{profile.organization_name}</h1>
                <div className="flex items-center justify-center space-x-2 mt-2">
                  <span className={`px-2 py-1 rounded-full text-sm ${
                    profile.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                  }`}>
                    {profile.status === 'active' ? 'Active' : 'Inactive'}
                  </span>
                  <span className="px-2 py-1 rounded-full text-sm bg-blue-100 text-blue-800">
                    {profile.role === 'provider' ? 'Provider' : 'Recipient'}
                  </span>
                </div>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-semibold">Mission Statement</h3>
                <p className="text-gray-700">{profile.mission_statement}</p>
              </div>

              <div>
                <h3 className="text-lg font-semibold">Sectors</h3>
                <div className="flex flex-wrap gap-2 mt-2">
                  {profile.sectors?.map((sector) => (
                    <span key={sector} className="px-3 py-1 bg-gray-200 rounded-full text-sm">
                      {sector}
                    </span>
                  ))}
                </div>
              </div>

              <div>
                <h3 className="text-lg font-semibold">Target Groups</h3>
                <div className="flex flex-wrap gap-2 mt-2">
                  {profile.target_groups?.map((group) => (
                    <span key={group} className="px-3 py-1 bg-gray-200 rounded-full text-sm">
                      {group}
                    </span>
                  ))}
                </div>
              </div>

              <div>
                <h3 className="text-lg font-semibold">Project Stage</h3>
                <p className="text-gray-700">{profile.project_stage}</p>
              </div>

              {recipientData && (
                <div>
                  <h3 className="text-lg font-semibold">Recipient Information</h3>
                  <div className="mt-2 space-y-2">
                    <p><span className="font-medium">Needs:</span> {recipientData.needs.join(", ")}</p>
                    <p><span className="font-medium">Budget Requested:</span> ${recipientData.budget_requested.toLocaleString()}</p>
                    <p><span className="font-medium">Team Size:</span> {recipientData.team_size}</p>
                    <p><span className="font-medium">Timeline:</span> {recipientData.timeline}</p>
                    <p><span className="font-medium">Prior Funding:</span> {recipientData.prior_funding ? "Yes" : "No"}</p>
                  </div>
                </div>
              )}

              {providerData && (
                <div>
                  <h3 className="text-lg font-semibold">Provider Information</h3>
                  <div className="mt-2 space-y-2">
                    <p><span className="font-medium">Funding Type:</span> {providerData.funding_type}</p>
                    <p><span className="font-medium">Amount Offered:</span> ${providerData.amount_offered.toLocaleString()}</p>
                    <p><span className="font-medium">Region Scope:</span> {providerData.region_scope}</p>
                    <p><span className="font-medium">Deadline:</span> {new Date(providerData.deadline).toLocaleDateString()}</p>
                    <p><span className="font-medium">Application Link:</span> <a href={providerData.application_link} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">Apply Now</a></p>
                  </div>
                </div>
              )}

              <div className="flex justify-end gap-2 mt-6">
                {matchStatus?.isMatched ? (
                  <Button 
                    variant="default"
                    onClick={() => navigate(`/chats?user=${userId}`)}
                    className="bg-purple-500 hover:bg-purple-600"
                  >
                    <MessageCircle className="h-4 w-4 mr-2" />
                    Chat History
                  </Button>
                ) : (
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
                      onClick={() => connectMutation.mutate()}
                    >
                      <Handshake className="h-4 w-4" />
                    </Button>
                  </>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default UserProfile;
