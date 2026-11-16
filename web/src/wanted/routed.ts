import { z } from "zod";

import { type Reason, type Sort, defaultReasons, reasons, sorts } from "./wanted-words";
import type { WantedFilter } from "./wanted-query";

/** Wanted's filter in its address, such as `reason=failed,gone` and `sort=last`; anything left out is the default. */
export const wantedSearch = z.object({
  reason: z.string().optional().catch(undefined),
  sort: z.enum(sorts).optional().catch(undefined),
});

export type WantedSearch = z.infer<typeof wantedSearch>;

const known: ReadonlySet<string> = new Set(reasons);

function reasonsOf(text: string | undefined): readonly Reason[] {
  const listed = new Set((text ?? "").split(",").filter((part) => known.has(part)));
  const chosen = reasons.filter((reason) => listed.has(reason));
  return chosen.length === 0 ? defaultReasons : chosen;
}

export const filterOf = (search: WantedSearch, contentType: string | undefined): WantedFilter => ({
  contentType,
  reasons: reasonsOf(search.reason),
  sort: search.sort ?? "next",
});

const sameReasons = (a: readonly Reason[], b: readonly Reason[]) =>
  a.length === b.length && a.every((reason) => b.includes(reason));

export const searchOf = ({ reasons: chosen, sort }: WantedFilter): WantedSearch => ({
  ...(!sameReasons(chosen, defaultReasons) && { reason: chosen.join(",") }),
  ...(sort !== "next" && { sort: sort satisfies Sort }),
});
