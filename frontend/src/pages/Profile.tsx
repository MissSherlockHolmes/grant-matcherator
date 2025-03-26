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
import { Profile, RecipientData, ProviderData } from "@/types";
import { Header } from "@/components/Header";
import { apiRequest } from "@/lib/api";
import { X } from "lucide-react";

// Constants for select options
const SECTORS = [
  "Education", "Healthcare", "Environment", "Arts & Culture",
  "Social Services", "Technology", "Economic Development",
  "Youth Development", "Community Development", "Research"
];

const TARGET_GROUPS = [
  "Children", "Youth", "Elderly", "Veterans", "Immigrants",
  "Low-income", "Disabilities", "Women", "Minorities",
  "LGBTQ+", "Students", "Unemployed"
];

const APPLICANT_TYPES = [
  "Non-profit", "For-profit", "Government", "Educational Institution",
  "Research Organization", "Community Group", "Individual", "Startup",
  "Social Enterprise", "Cooperative", "Foundation", "Association"
];

const PROJECT_STAGES = [
  "Idea Stage", "Pre-seed", "Seed", "Early Stage",
  "Growth Stage", "Scale-up", "Mature", "Expansion",
  "Pilot", "MVP", "Product Development", "Market Testing"
];

const LANGUAGES = ["English", "Spanish"];

const ProfilePage = () => {
  const navigate = useNavigate();
  const [profile, setProfile] = useState<Profile>({
    id: 0,
    organization_name: "",
    profile_picture_url: null,
    mission_statement: "",
    state: "",
    city: "",
    zip_code: "",
    ein: "",
    language: "",
    applicant_type: "",
    sectors: [],
    target_groups: [],
    project_stage: "",
    website_url: "",
    contact_email: "",
    chat_opt_in: false,
    location: "",
    website: "",
    status: "active",
    role: "provider"
  });

  const { data: profileData, isLoading, error } = useQuery({
    queryKey: ['profile'],
    queryFn: async () => {
      console.log('Fetching profile data...');
      const response = await apiRequest('/me/profile');
      console.log('Raw API response:', response);
      if (response) {
        setProfile(response);
      }
      return response;
    },
    retry: false,
    staleTime: 0,
    refetchOnMount: true
  });

  // Show error toast if query fails
  useEffect(() => {
    if (error) {
      console.error('Profile fetch error:', {
        error,
        message: error instanceof Error ? error.message : 'Unknown error',
        stack: error instanceof Error ? error.stack : undefined
      });
      toast.error("Failed to load profile data. Please try again.");
    }
  }, [error]);

  const updateProfileMutation = useMutation({
    mutationFn: async (updatedProfile: Profile) => {
      console.log('Updating profile with data:', updatedProfile);
      const response = await apiRequest('/me/profile', {
        method: 'PUT',
        body: JSON.stringify(updatedProfile),
      });
      console.log('Raw API response from update:', response);
      return response;
    },
    onSuccess: (data) => {
      console.log('Profile update successful:', data);
      toast.success("Profile updated successfully!");
    },
    onError: (error) => {
      console.error('Profile update failed:', error);
      toast.error(`Failed to update profile: ${error.message}`);
    },
  });

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const formData = new FormData();
      formData.append("file", file);

      try {
        const response = await fetch('/api/upload/profile-picture', {
          method: 'POST',
          body: formData,
        });

        if (!response.ok) {
          throw new Error('Failed to upload image');
        }

        const data = await response.json();
        setProfile((prev) => ({
          ...prev,
          profile_picture_url: data.url,
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
      profile_picture_url: null,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    updateProfileMutation.mutate(profile);
  };

  const handleArrayChange = (field: 'sectors' | 'target_groups', value: string) => {
    setProfile((prev) => {
      const currentArray = prev[field] || [];
      return {
        ...prev,
        [field]: currentArray.includes(value)
          ? currentArray.filter((item) => item !== value)
          : [...currentArray, value],
      };
    });
  };

  if (isLoading) return <div>Loading...</div>;

  return (
    <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
      <Header />
      <div className="max-w-2xl mx-auto p-8">
        <Card>
          <CardHeader>
            <h1 className="text-2xl font-bold text-center">Organization Profile</h1>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-6">
              <div className="flex flex-col items-center space-y-4">
                <div className="relative">
                  <Avatar className="h-32 w-32">
                    <AvatarImage
                      src={profile.profile_picture_url || "/placeholder.svg"}
                      alt="Profile"
                    />
                    <AvatarFallback>👤</AvatarFallback>
                  </Avatar>
                  {profile.profile_picture_url && (
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
                  <label className="block text-sm font-medium mb-1">Organization Name</label>
                  <Input
                    value={profile.organization_name || ""}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, organization_name: e.target.value }))
                    }
                    placeholder="Your organization's name"
                    required
                  />
                  <div className="flex items-center space-x-2 mt-2">
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

                <div>
                  <label className="block text-sm font-medium mb-1">Mission Statement</label>
                  <Textarea
                    value={profile.mission_statement || ""}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, mission_statement: e.target.value }))
                    }
                    placeholder="Your organization's mission statement"
                    className="min-h-[100px]"
                    required
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">State</label>
                    <Input
                      value={profile.state || ""}
                      onChange={(e) =>
                        setProfile((prev) => ({ ...prev, state: e.target.value }))
                      }
                      placeholder="State"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">City</label>
                    <Input
                      value={profile.city || ""}
                      onChange={(e) =>
                        setProfile((prev) => ({ ...prev, city: e.target.value }))
                      }
                      placeholder="City"
                      required
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">ZIP Code</label>
                  <Input
                    value={profile.zip_code || ""}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, zip_code: e.target.value }))
                    }
                    placeholder="ZIP Code"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">EIN</label>
                  <Input
                    value={profile.ein || ""}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, ein: e.target.value }))
                    }
                    placeholder="Employer Identification Number"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Language</label>
                  <Select
                    value={profile.language || ""}
                    onValueChange={(value) =>
                      setProfile((prev) => ({ ...prev, language: value }))
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Select primary language" />
                    </SelectTrigger>
                    <SelectContent>
                      {LANGUAGES.map((lang) => (
                        <SelectItem key={lang} value={lang}>
                          {lang}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Applicant Type</label>
                  <Select
                    value={profile.applicant_type || ""}
                    onValueChange={(value) =>
                      setProfile((prev) => ({ ...prev, applicant_type: value }))
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Select applicant type" />
                    </SelectTrigger>
                    <SelectContent>
                      {APPLICANT_TYPES.map((type) => (
                        <SelectItem key={type} value={type}>
                          {type}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Project Stage</label>
                  <Select
                    value={profile.project_stage || ""}
                    onValueChange={(value) =>
                      setProfile((prev) => ({ ...prev, project_stage: value }))
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Select project stage" />
                    </SelectTrigger>
                    <SelectContent>
                      {PROJECT_STAGES.map((stage) => (
                        <SelectItem key={stage} value={stage}>
                          {stage}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Sectors</label>
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                    {SECTORS.map((sector) => (
                      <Button
                        key={sector}
                        type="button"
                        variant={(profile.sectors || []).includes(sector) ? "default" : "outline"}
                        className="text-sm"
                        onClick={() => handleArrayChange('sectors', sector)}
                      >
                        {sector}
                      </Button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Target Groups</label>
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                    {TARGET_GROUPS.map((group) => (
                      <Button
                        key={group}
                        type="button"
                        variant={(profile.target_groups || []).includes(group) ? "default" : "outline"}
                        className="text-sm"
                        onClick={() => handleArrayChange('target_groups', group)}
                      >
                        {group}
                      </Button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Website URL</label>
                  <Input
                    value={profile.website_url || ""}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, website_url: e.target.value }))
                    }
                    placeholder="https://your-organization.org"
                    type="url"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Contact Email</label>
                  <Input
                    value={profile.contact_email}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, contact_email: e.target.value }))
                    }
                    placeholder="contact@your-organization.org"
                    type="email"
                    required
                  />
                </div>

                <div className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    id="chat_opt_in"
                    checked={profile.chat_opt_in}
                    onChange={(e) =>
                      setProfile((prev) => ({ ...prev, chat_opt_in: e.target.checked }))
                    }
                  />
                  <label htmlFor="chat_opt_in" className="text-sm font-medium">
                    Enable chat with potential matches
                  </label>
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