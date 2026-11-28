import { z } from "zod";

import type { ListHistoryParams } from "../api/gen/model";
import { type SinceChip, sinceOf } from "./day-heading";

export const groups = [
  "grabs",
  "imports",
  "library",
  "subtitles",
  "health",
  "backups",
  "moves",
  "conversions",
] as const;

/** What History lists, kept in the address. The Content Type is Activity's own `type`. */
export const historySearch = z.object({
  groups: z.array(z.enum(groups)).optional().catch(undefined),
  kinds: z.array(z.string()).optional().catch(undefined),
  since: z.enum(["today", "7d", "30d"]).optional().catch(undefined),
  work: z.uuid().optional().catch(undefined),
});

export type HistorySearch = z.infer<typeof historySearch>;

export type HistoryFilter = HistorySearch & { readonly type?: string | undefined };

/** The filter as the API reads it, the date chip as the browser's midnight that starts it. */
export const paramsOf = (
  { type, groups: chosen, kinds, since, work }: HistoryFilter,
  now: Date,
): ListHistoryParams => ({
  ...(type !== undefined && { contentType: type }),
  ...(chosen !== undefined && chosen.length > 0 && { groups: chosen }),
  ...(kinds !== undefined && kinds.length > 0 && { kinds }),
  ...(since !== undefined && { since: sinceOf(since satisfies SinceChip, now) }),
  ...(work !== undefined && { work }),
});

export const filtered = ({ type, groups: chosen, kinds, since, work }: HistoryFilter): boolean =>
  type !== undefined ||
  (chosen?.length ?? 0) > 0 ||
  (kinds?.length ?? 0) > 0 ||
  since !== undefined ||
  work !== undefined;
