import { type QueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useId } from "react";

import type { MediaserversMyAccessRow as MyRow } from "../../api/gen/model";
import { getListMyMediaServerAccessSuspenseQueryOptions } from "../../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../../api/query-client";
import { sessionQuery } from "../../session/session";
import { Section } from "../../ui/section";
import { HowToWatch } from "./how-to-watch";

export const myAccess = getListMyMediaServerAccessSuspenseQueryOptions;

/** Loads the Member's access for the portal. An Admin has none to show. */
export async function loadMyAccess(queryClient: QueryClient): Promise<unknown> {
  const session = await queryClient.query({ ...sessionQuery, staleTime: "static" });
  return session?.account.admin === false ? ensureQuery(queryClient, myAccess()) : null;
}

export function WhereToWatch() {
  const mine = useSuspenseQuery(myAccess()).data;
  const ready = mine.rows.filter((row) => !row.pending);
  if (ready.length === 0) {
    return null;
  }
  return (
    <Section title="Where to watch">
      <div className="flex flex-col gap-6">
        {ready.map((row) => (
          <ServerHowTo key={row.id} row={row} plexPassLine={mine.plexPassLine} />
        ))}
      </div>
    </Section>
  );
}

function ServerHowTo({ row, plexPassLine }: { row: MyRow; plexPassLine: boolean }) {
  const titleId = useId();
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-3">
      <h3 id={titleId} className="font-semibold text-ink-1">
        {row.server.name}
      </h3>
      <HowToWatch row={row} plexPassLine={plexPassLine} />
    </section>
  );
}
