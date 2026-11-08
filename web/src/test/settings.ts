/** Settings' groups and their sections, in the order its nav and phone list show them. */
export const settingsGroupLabels = {
  Library: ["Content Types", "Library roots", "Library scan", "Metadata"],
  "Files to keep": ["Targets", "Profiles", "Editions"],
  "Search and download": ["Indexers", "Download Clients"],
  Access: ["Address", "Sign-in", "API key"],
  System: ["Nodes", "Email"],
} as const;
