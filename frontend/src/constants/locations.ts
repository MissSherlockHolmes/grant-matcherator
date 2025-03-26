export const LOCATIONS = [
  "Tallinn",
  "Tartu",
  "Narva",
  "Pärnu",
  "Kohtla-Järve",
  "Viljandi",
  "Rakvere",
  "Maardu",
  "Kuressaare",
  "Sillamäe",
  "Võru",
  "Valga",
  "Haapsalu",
  "Jõhvi",
  "Paide"
] as const;

export type Location = typeof LOCATIONS[number];