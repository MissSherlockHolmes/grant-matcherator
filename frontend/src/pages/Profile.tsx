import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { Profile } from "@/types";
import { Header } from "@/components/Header";
import { apiRequest } from "@/lib/api";
import { LOCATIONS } from "@/constants/locations";
import { LOOKING_FOR_OPTIONS } from "@/constants/profile";
import { OCCUPATION_OPTIONS, INTEREST_OPTIONS, type InterestOption, type OccupationOption } from "@/constants/profile-options";
import { X } from "lucide-react";

const ProfilePage = () => {
  const navigate = useNavigate();
  const [profile, setProfile] = useState<Profile>({
    id: 0,
    name: "",
    bio: "",
    interests: [],
    location: "",
    lookingFor: "",
    age: 0,
    occupation: "",
    profilePictureUrl: null,
    email: "", // Add this line
  });

  const { data: profileData, isLoading } = useQuery({
    queryKey: ['profile'],
    queryFn: () => apiRequest('/api/me/profile'),
  });

  useEffect(() => {
    if (profileData) {
      setProfile({
        id: profileData.user_id || 0,
        name: profileData.name || "",
        bio: profileData.bio || "",
        interests: (profileData.interests || []) as InterestOption[],
        location: profileData.location || "",
        lookingFor: profileData.looking_for || "",
        age: profileData.age || 0,
        occupation: (profileData.occupation as OccupationOption) || "",
        profilePictureUrl: profileData.profile_picture_url || null,
        email: profileData.email || "", // Add this line
      });
    }
  }, [profileData]);

  const updateProfileMutation = useMutation({
    mutationFn: async (updatedProfile: Profile) => {
      const profileData = {
        user_id: updatedProfile.id,
        name: updatedProfile.name,
        bio: updatedProfile.bio,
        interests: updatedProfile.interests,
        location: updatedProfile.location,
        looking_for: updatedProfile.lookingFor,
        age: updatedProfile.age,
        occupation: updatedProfile.occupation,
        profile_picture_url: updatedProfile.profilePictureUrl,
      };

      return apiRequest('/api/me/profile', {
        method: 'PUT',
        body: JSON.stringify(profileData),
      });
    },
    onSuccess: () => {
      toast.success("Profile updated successfully!");
    },
    onError: (error) => {
      toast.error(`Failed to update profile: ${error.message}`);
    },
  });

  const handleAgeChange = (e) => {
    let age = parseInt(e.target.value) || 0;
    if (age < 1) age = 0;
    if (age > 100) age = 100;
    setProfile((prev) => ({ ...prev, age }));
  };

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const formData = new FormData();
      formData.append("file", file);

      try {
        const token = localStorage.getItem('token');
        const response = await fetch('http://localhost:3000/api/upload/profile-picture', {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
          },
          body: formData,
        });

        if (!response.ok) {
          throw new Error('Failed to upload image');
        }

        const data = await response.json();
        setProfile((prev) => ({
          ...prev,
          profilePictureUrl: `http://localhost:3000${data.url}`,
        }));
        
        toast.success("Profile picture uploaded successfully!");
      } catch (error) {
        console.error('Error uploading image:', error);
        toast.error("Failed to upload profile picture");
      }
    }
  };

  const handleRemoveImage = () => {
    setProfile((prev) => ({
      ...prev,
      profilePictureUrl: null,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    updateProfileMutation.mutate(profile);
  };

  const handleInterestChange = (interest: InterestOption) => {
    setProfile((prev) => ({
      ...prev,
      interests: prev.interests.includes(interest)
        ? prev.interests.filter((i) => i !== interest)
        : [...prev.interests, interest],
    }));
  };

  if (isLoading) return <div>Loading...</div>;

  return (
    <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
      <Header />
      <div className="max-w-2xl mx-auto p-8">
        <Card>
          <CardHeader>
            <h1 className="text-2xl font-bold text-center">Profile Management</h1>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-6">
              <div className="flex flex-col items-center space-y-4">
                <div className="relative">
                  <Avatar className="h-32 w-32">
                    <AvatarImage
                      src={profile.profilePictureUrl || "/placeholder.svg"}
                      alt="Profile"
                    />
                    <AvatarFallback>👤</AvatarFallback>
                  </Avatar>
                  {profile.profilePictureUrl && (
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      className="absolute -top-2 -right-2 rounded-full h-6 w-6"
                      onClick={handleRemoveImage}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
                <Input
                  type="file"
                  accept="image/*"
                  onChange={handleImageUpload}
                  className="max-w-xs"
                />
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Email</label>
                  <Input
                    value={profile.email}
                    readOnly
                    placeholder="Your email"
                    className="bg-gray-100 cursor-not-allowed"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Name</label>
                  <Input
                    value={profile.name}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, name: e.target.value }))
                    }
                    placeholder="Your name"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Age</label>
                  <Input
                    type="number"
                    value={profile.age || ""}
                    onChange={handleAgeChange}
                    placeholder="Your age"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Location</label>
                  <Select
                    value={profile.location || undefined}
                    onValueChange={(value) =>
                      setProfile((prev) => ({ ...prev, location: value }))
                    }
                  >
                    <SelectTrigger className="w-full bg-white">
                      <SelectValue placeholder="Select your location" />
                    </SelectTrigger>
                    <SelectContent className="bg-white">
                      {LOCATIONS.map((location) => (
                        <SelectItem key={location} value={location}>
                          {location}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Occupation</label>
                  <Select
                    value={profile.occupation || undefined}
                    onValueChange={(value: OccupationOption) =>
                      setProfile((prev) => ({ ...prev, occupation: value }))
                    }
                  >
                    <SelectTrigger className="w-full bg-white">
                      <SelectValue placeholder="Select your occupation" />
                    </SelectTrigger>
                    <SelectContent className="bg-white">
                      {OCCUPATION_OPTIONS.map((occupation) => (
                        <SelectItem key={occupation} value={occupation}>
                          {occupation}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Bio</label>
                  <Textarea
                    value={profile.bio || ""}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, bio: e.target.value }))
                    }
                    placeholder="Tell us about yourself"
                    className="min-h-[100px]"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Interests</label>
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                    {INTEREST_OPTIONS.map((interest) => (
                      <Button
                        key={interest}
                        type="button"
                        variant={profile.interests.includes(interest) ? "default" : "outline"}
                        className="text-sm"
                        onClick={() => handleInterestChange(interest)}
                      >
                        {interest}
                      </Button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Looking For</label>
                  <Select
                    value={profile.lookingFor || undefined}
                    onValueChange={(value) =>
                      setProfile((prev) => ({ ...prev, lookingFor: value }))
                    }
                  >
                    <SelectTrigger className="w-full bg-white">
                      <SelectValue placeholder="What are you looking for?" />
                    </SelectTrigger>
                    <SelectContent className="bg-white">
                      {LOOKING_FOR_OPTIONS.map((option) => (
                        <SelectItem key={option} value={option}>
                          {option}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex justify-end space-x-4">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => navigate("/dashboard")}
                  >
                    Cancel
                  </Button>
                  <Button type="submit">Save Profile</Button>
                </div>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default ProfilePage;