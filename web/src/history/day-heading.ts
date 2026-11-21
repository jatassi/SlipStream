/** A date chip: today, the last 7 days or the last 30, each counting today. */
export type SinceChip = "today" | "7d" | "30d";

const chipDays: Record<SinceChip, number> = { today: 0, "7d": 6, "30d": 29 };

/** The browser's midnight that starts chip's window, as an instant. */
export function sinceOf(chip: SinceChip, now: Date): string {
  const midnight = new Date(now);
  midnight.setHours(0, 0, 0, 0);
  midnight.setDate(midnight.getDate() - chipDays[chip]);
  return midnight.toISOString();
}
