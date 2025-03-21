export const OCCUPATION_OPTIONS = [
  "Software Developer",
  "Teacher",
  "Healthcare Professional",
  "Business Professional",
  "Student",
  "Artist/Creative",
  "Engineer",
  "Sales/Marketing",
  "Service Industry",
  "Entrepreneur",
  "Other"
] as const;

export const INTEREST_OPTIONS = [
  "Reading",
  "Travel",
  "Music",
  "Sports",
  "Gaming",
  "Cooking",
  "Photography",
  "Art",
  "Technology",
  "Movies/TV",
  "Fitness",
  "Nature/Outdoors",
  "Cars/Automotive",
  "Fashion",
  "Science",
  "Animals/Pets",
  "Food/Dining",
  "Dancing",
  "Writing",
  "Volunteering"
] as const;

export type OccupationOption = typeof OCCUPATION_OPTIONS[number];
export type InterestOption = typeof INTEREST_OPTIONS[number];