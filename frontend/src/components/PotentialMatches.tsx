import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "./ui/card";
import { Button } from "./ui/button";
import { useToast } from "./ui/use-toast";
import { Heart, X } from "lucide-react";
import { apiRequest } from "@/lib/api";
import { useNavigate } from "react-router-dom";
import { Badge } from "./ui/badge";
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from "./ui/select";
import { useEffect } from "react";
import { connection } from "@/lib/api/config";

interface PotentialMatch {
  id: number;
  score: number;
  email: string;
  organization_name: string;
  profile_picture_url: string | null;
}

interface LikeResponse {
  isMatch: boolean;
}

export const PotentialMatches = () => {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  // Fetch user profile
  const { data: profile, isLoading: profileLoading, error: profileError } = useQuery({
    queryKey: ["profile"],
    queryFn: async () => {
      console.log('=== Profile Fetch Start ===');
      console.log('Making request to /me/profile');
      try {
        const response = await apiRequest("/me/profile");
        console.log('Profile API Response:', {
          status: 'success',
          data: response
        });
        return response;
      } catch (error) {
        console.error('Profile API Error:', {
          error,
          message: error instanceof Error ? error.message : 'Unknown error',
          stack: error instanceof Error ? error.stack : undefined
        });
        throw error;
      }
    },
  });

  // Recalculate matches mutation
  const recalculateMatchesMutation = useMutation({
    mutationFn: async () => {
      console.log('=== Recalculate Matches Start ===');
      try {
        await connection.recalculateMatches();
        console.log('Recalculate Matches Success');
      } catch (error) {
        console.error('Recalculate Matches Error:', {
          error,
          message: error instanceof Error ? error.message : 'Unknown error'
        });
        throw error;
      }
    },
    onSuccess: () => {
      console.log('Invalidating potential-matches query');
      queryClient.invalidateQueries({ queryKey: ["potential-matches"] });
    },
  });

  // Fetch potential matches based on preference
  const { data: matches, isLoading, error: matchesError } = useQuery({
    queryKey: ["potential-matches"],
    queryFn: async () => {
      console.log('=== Potential Matches Fetch Start ===');
      console.log('Making request to /potential-matches');
      try {
        const response = await apiRequest("/potential-matches");
        console.log('Potential Matches API Response:', {
          status: 'success',
          data: response,
          type: typeof response,
          isArray: Array.isArray(response),
          length: Array.isArray(response) ? response.length : 'not an array'
        });
        
        if (!response) {
          console.warn('No matches returned from API');
          return [];
        }
        
        if (!Array.isArray(response)) {
          console.warn('Response is not an array:', {
            type: typeof response,
            value: response
          });
          return [];
        }
        
        console.log(`Found ${response.length} potential matches`);
        console.log('First match:', response[0]);
        return response;
      } catch (error) {
        console.error('Potential Matches API Error:', {
          error,
          message: error instanceof Error ? error.message : 'Unknown error',
          stack: error instanceof Error ? error.stack : undefined
        });
        throw error;
      }
    },
    enabled: !!profile?.organization_name,
  });

  // Recalculate matches when component mounts
  useEffect(() => {
    if (profile?.organization_name) {
      console.log('Recalculating matches on mount');
      recalculateMatchesMutation.mutate();
    }
  }, [profile?.organization_name]);

  // Log any errors that occur
  useEffect(() => {
    console.log('=== Component State Update ===');
    if (profileError) {
      console.error('Profile Query Error:', {
        error: profileError,
        message: profileError instanceof Error ? profileError.message : 'Unknown error'
      });
    }
    if (matchesError) {
      console.error('Potential Matches Query Error:', {
        error: matchesError,
        message: matchesError instanceof Error ? matchesError.message : 'Unknown error'
      });
    }
  }, [profileError, matchesError]);

  const connectMutation = useMutation({
    mutationFn: async (userId: number) => {
      console.log('=== Connect Action Start ===');
      console.log(`Attempting to connect with user ${userId}`);
      try {
        const response = await apiRequest('/connections', { 
          method: "POST",
          body: JSON.stringify({ target_id: userId })
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
    onSuccess: (_, userId) => {
      console.log('Connect Action Success');
      toast({
        title: "Connected!",
        description: "You are now connected with this organization",
      });
      // Optimistically update the UI by removing the connected match
      queryClient.setQueryData(['potential-matches'], (oldData: any) => {
        if (!oldData) return oldData;
        return oldData.filter((match: PotentialMatch) => match.id !== userId);
      });
      // Then invalidate to ensure we have the latest data
      queryClient.invalidateQueries({ queryKey: ["potential-matches"] });
      queryClient.invalidateQueries({ queryKey: ["connections"] });
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
    mutationFn: async (userId: number) => {
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
    onSuccess: (_, userId) => {
      console.log('Dismiss Action Success');
      toast({
        title: "Profile dismissed",
        description: "You won't see this profile again",
      });
      // Optimistically update the UI by removing the dismissed match
      queryClient.setQueryData(['potential-matches'], (oldData: any) => {
        if (!oldData) return oldData;
        return oldData.filter((match: PotentialMatch) => match.id !== userId);
      });
      // Then invalidate to ensure we have the latest data
      queryClient.invalidateQueries({ queryKey: ['potential-matches'] });
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

  // Log component state changes
  useEffect(() => {
    console.log('=== Component State ===', {
      profileLoading,
      isLoading,
      hasProfile: !!profile,
      matchesCount: matches?.length || 0
    });
  }, [profileLoading, isLoading, profile, matches]);

  if (profileLoading || isLoading) {
    console.log('=== Loading State ===');
    return <div>Loading...</div>;
  }

  if (!profile?.organization_name) {
    console.log('=== No Profile State ===');
    return (
      <Card className="p-8 text-center">
        <h3 className="text-xl font-semibold mb-4">Welcome to Grant Matcherator!</h3>
        <p className="text-gray-600 mb-6">Please complete your profile to start seeing potential matches.</p>
        <Button onClick={() => navigate("/profile")}>Complete Profile</Button>
      </Card>
    );
  }

  if (!matches?.length) {
    console.log('=== No Matches State ===');
    return <div>No potential matches found at the moment.</div>;
  }

  console.log('=== Rendering Matches ===', {
    count: matches.length,
    matches: matches.map(m => ({
      id: m.id,
      name: m.organization_name,
      score: m.score
    }))
  });

  return (
    <div className="space-y-4">
      {matches.map((match: PotentialMatch) => (
        <Card key={match.id} className="p-4">
          <div className="flex items-start gap-4">
            {match.profile_picture_url && (
              <img
                src={match.profile_picture_url}
                alt={match.organization_name}
                className="w-24 h-24 rounded-full object-cover"
              />
            )}
            <div className="flex-1">
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="text-lg font-semibold">{match.organization_name}</h3>
                  <p className="text-sm text-gray-500">{match.email}</p>
                </div>
                <div className="flex flex-col items-end gap-2">
                  <span className="text-sm text-green-600">
                    {Math.round(match.score)}% Match
                  </span>
                </div>
              </div>
              <div className="mt-4 flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => navigate(`/users/${match.id}`)}
                >
                  View Profile
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => dismissMutation.mutate(match.id)}
                >
                  <X className="h-4 w-4" />
                </Button>
                <Button
                  variant="default"
                  onClick={() => connectMutation.mutate(match.id)}
                >
                  Connect
                </Button>
              </div>
            </div>
          </div>
        </Card>
      ))}
    </div>
  );
};
