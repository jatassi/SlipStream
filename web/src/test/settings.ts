/** Settings' groups and their sections, in the order its nav and phone list show them. */
export const settingsGroupLabels = {
  Library: ["Content Types", "Library roots", "Library scan", "Importing", "Naming"],
  "Metadata and subtitles": ["Metadata", "Subtitles"],
  "Targets and Profiles": ["Targets", "Profiles", "Custom Formats", "Editions"],
  "Search and download": ["Indexers", "Automatic search", "Trackers", "Download Clients"],
  Access: ["Address", "Sign-in", "API key"],
  System: ["Nodes", "Email", "Media Servers", "Notifications"],
} as const;
