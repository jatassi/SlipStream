import { HttpResponse, http } from "msw";

import type { TargetsImpact } from "../api/gen/model";
import { problem } from "./handlers";

export const noImpact: TargetsImpact = {
  works: 0,
  ownList: 0,
  unwanted: { files: 0, bytes: 0 },
  renamed: 0,
  newlyWanted: 0,
  downloading: 0,
};

export type Sent = { readonly query: Record<string, string>; readonly body: unknown };

/**
 * A change op that answers each call from `answer`, given the query it was
 * sent and how many calls came before; every call is recorded in `sent`.
 * `answer` returning "changed" refuses the save with impact_changed.
 */
export function changeOp(
  method: "put" | "delete",
  path: string,
  answer: (query: URLSearchParams, calls: number) => TargetsImpact | "changed",
  record: Record<string, unknown>,
) {
  const sent: Sent[] = [];
  const handler = http[method](path, async ({ request }) => {
    const query = new URL(request.url).searchParams;
    sent.push({
      query: Object.fromEntries(query),
      body: method === "put" ? await request.json() : undefined,
    });
    const impact = answer(query, sent.length - 1);
    if (impact === "changed") {
      return problem(409, "impact_changed", "The change now does more than its preview showed");
    }
    return HttpResponse.json({ impact, ...record });
  });
  return { sent, handler };
}
