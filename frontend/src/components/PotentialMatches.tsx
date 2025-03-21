import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "./ui/card";
import { Button } from "./ui/button";
import { useToast } from "./ui/use-toast";
import { Heart, X } from "lucide-react";
import { apiRequest } from "@/lib/api";
import { useNavigate } from "react-router-dom";
import { Badge } from "./ui/badge";
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from "./ui/select";

interface PotentialMatch {
  user_id: number;
  name: string;
  bio: string;
  interests: string[];
  location: string;
  age: number;
  looking_for: string;
  profile_picture_url: string | null;
  match_score: number;
  has_liked_me: boolean;
}

interface LikeResponse {
  isMatch: boolean;
}

export const PotentialMatches = () => {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  // Fetch user profile
  const { data: profile, isLoading: profileLoading } = useQuery({
    queryKey: ["profile"],
    queryFn: () => apiRequest("/api/me/profile"),
  });

  // Fetch user match preference
  const { data: matchPreference } = useQuery({
    queryKey: ["match-preference"],
    queryFn: () => apiRequest("/api/matches/preferences"),
  });

  // Fetch potential matches based on preference
  const { data: matches, isLoading } = useQuery({
    queryKey: ["potential-matches", matchPreference?.priority],
    queryFn: () => apiRequest(`/api/matches/potential?priority=${matchPreference?.priority || "none"}`),
    enabled: !!profile?.name,
  });

  const likeMutation = useMutation({
    mutationFn: async (userId: number) => {
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
        queryClient.invalidateQueries({ queryKey: ["potential-matches"] });
        queryClient.invalidateQueries({ queryKey: ["matches"] });
      } else {
        toast({
          title: "Like sent!",
          description: "We'll let you know if they like you back",
        });
        queryClient.invalidateQueries({ queryKey: ["potential-matches"] });
      }
    },
    onError: () => {
      toast({
        title: "Error",
        description: "Failed to like profile. Please try again.",
        variant: "destructive",
      });
    },
  });
  
  const dismissMutation = useMutation({
    mutationFn: (userId: number) => 
      apiRequest(`/api/matches/dismiss/${userId}`, { method: 'POST' }),
    onSuccess: () => {
      toast({
        title: "Profile dismissed",
        description: "You won't see this profile again",
      });
      queryClient.invalidateQueries({ queryKey: ['potential-matches'] });
    },
  });

  const updatePreferenceMutation = useMutation({
    mutationFn: async (priority: string) => {
      return apiRequest("/api/matches/preferences", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ priority }),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["match-preference"] });
      queryClient.invalidateQueries({ queryKey: ["potential-matches"] });
    },
  });

  // Sort matches based on selected priority
  const sortedMatches = matches ? [...matches].sort((a, b) => {
    if (!matchPreference?.priority || matchPreference.priority === "none") return 0;

    if (matchPreference.priority === "age" && profile) {
      return Math.abs(a.age - profile.age) - Math.abs(b.age - profile.age);
    }

    if (matchPreference.priority === "interests" && profile) {
      const commonA = a.interests.filter(i => profile.interests.includes(i)).length;
      const commonB = b.interests.filter(i => profile.interests.includes(i)).length;
      return commonB - commonA; // Sort in descending order
    }

    if (matchPreference.priority === "looking_for" && profile) {
      return a.looking_for === profile.looking_for ? -1 : 1;
    }

    return 0;
  }) : [];

  if (profileLoading || isLoading) {
    return <div>Loading...</div>;
  }

  if (!profile?.name) {
    return (
      <Card className="p-8 text-center">
        <h3 className="text-xl font-semibold mb-4">Welcome to Match Me!</h3>
        <p className="text-gray-600 mb-6">Please complete your profile to start seeing potential matches.</p>
        <Button onClick={() => navigate("/profile")}>Complete Profile</Button>
      </Card>
    );
  }

  if (!sortedMatches.length) {
    return <div>No potential matches found at the moment.</div>;
  }

  return (
    <div className="space-y-4">
      {/* Filter Dropdown */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700">Filter by:</label>
        <Select
          value={matchPreference?.priority || "none"}
          onValueChange={(value) => updatePreferenceMutation.mutate(value)}
        >
          <SelectTrigger className="w-full border p-2 rounded">
          <SelectValue>
  {matchPreference?.priority
    ? {
        none: "Best match",
        interests: "Interests",
        age: "Age",
        looking_for: "Looking for",
      }[matchPreference.priority] || "None"
    : "None"}
</SelectValue>

          </SelectTrigger>
          <SelectContent className="bg-white border rounded shadow-md">
            <SelectItem value="none">Best match</SelectItem>
            <SelectItem value="interests">Interests</SelectItem>
            <SelectItem value="age">Age</SelectItem>
            <SelectItem value="looking_for">Looking for</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {sortedMatches.map((match: PotentialMatch) => (
        <Card key={match.user_id} className="p-4">
          <div className="flex items-start gap-4">
            {match.profile_picture_url && (
              <img
                src={match.profile_picture_url}
                alt={match.name}
                className="w-24 h-24 rounded-full object-cover"
              />
            )}
            <div className="flex-1">
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="text-lg font-semibold">{match.name}, {match.age}</h3>
                  <p className="text-sm text-gray-500">{match.location}</p>
                </div>
                <div className="flex flex-col items-end gap-2">
                  <span className="text-sm text-green-600">
                    {Math.round(match.match_score)}% Match
                  </span>
                  {match.has_liked_me && (
                    <Badge variant="secondary" className="bg-pink-100 text-pink-800">
                      Likes you
                    </Badge>
                  )}
                </div>
              </div>
              <p className="mt-2 text-sm text-gray-700">{match.bio}</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {match.interests.map((interest, index) => (
                  <span
                    key={index}
                    className="px-2 py-1 bg-gray-100 rounded-full text-xs text-gray-700"
                  >
                    {interest}
                  </span>
                ))}
              </div>
              <div className="mt-2">
                <Badge variant="secondary" className="text-xs">
                  Looking for: {match.looking_for}
                </Badge>
              </div>
              <div className="mt-4 flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => navigate(`/users/${match.user_id}`)}
                >
                  View Profile
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => dismissMutation.mutate(match.user_id)}
                >
                  <X className="h-4 w-4" />
                </Button>
                <Button
                  variant="default"
                  size="icon"
                  className="bg-red-500 hover:bg-red-600"
                  onClick={() => likeMutation.mutate(match.user_id)}
                >
                  <Heart className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </div>
        </Card>
      ))}
    </div>
  );
};
