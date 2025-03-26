import { OccupationOption, InterestOption } from '@/constants/profile-options';

export interface Profile {
  id: number;
  organization_name?: string;
  profile_picture_url: string | null;
  mission_statement?: string;
  state?: string;
  city?: string;
  zip_code?: string;
  ein?: string;
  language?: string;
  applicant_type?: string;
  sectors?: string[];
  target_groups?: string[];
  project_stage?: string;
  website_url?: string;
  contact_email: string;
  chat_opt_in: boolean;
  location?: string;
  website?: string;
  role?: string;
  status?: string;
}

export interface RecipientData {
  needs: string[];
  budget_requested: number;
  team_size: number;
  timeline: string;
  prior_funding: boolean;
}

export interface ProviderData {
  funding_type: string;
  amount_offered: number;
  region_scope: string;
  location_notes: string;
  eligibility_notes: string;
  deadline: string;
  application_link: string;
}

export interface Message {
  id: number;
  senderId: number;
  content: string;
  createdAt: string;
  read: boolean;
}

export interface Connection {
  id: number;
  initiator_id: number;
  target_id: number;
  created_at: string;
  updated_at: string;
  other_user_name: string;
  other_user_picture: string;
  connection_type: 'following' | 'follower';
}