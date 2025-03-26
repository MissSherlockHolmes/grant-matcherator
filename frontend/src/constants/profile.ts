export const LOOKING_FOR_OPTIONS = [
  "Friendship",
  "Relationship",
  "Casual Dating",
  "Networking",
  "Chat Buddies",
  "Activity Partners"
] as const;

export type LookingForOption = typeof LOOKING_FOR_OPTIONS[number];