/**
 * Whether a torrent that owes is at risk of a hit and run: yes while no copy of it announces as a seeder, unknown while every copy's Node is offline
 */
export type DownloadclientsSeedingViewAtRisk = "no" | "yes" | "unknown";
