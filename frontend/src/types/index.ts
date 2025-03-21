import { OccupationOption, InterestOption } from '@/constants/profile-options';

export interface Profile {
  id: number;
  name: string;
  bio: string;
  interests: InterestOption[];
  location: string;
  lookingFor: string;
  age: number;
  occupation: OccupationOption | "";
  profilePictureUrl: string | null;
  email: string;
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
  status: 'pending' | 'connected' | 'rejected';
}