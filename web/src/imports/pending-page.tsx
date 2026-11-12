import type { ReactNode } from "react";

import type { ImportsLead, WorksContentType } from "../api/gen/model";
import { type RootEntry, useRootImports } from "./root-entry";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Empty } from "../shell/empty";
import { Page } from "../shell/page";
import { Button } from "../ui/button";
import { useAdding } from "./adding";
import { type Group, RootGroup } from "./root-group";
import { isAddable } from "./words";

export type PendingView = "needs-you" | "ignored";

export const pendingParams = (view: PendingView) =>
  view === "ignored" ? { source: "root" as const, ignored: true } : { source: "root" as const };

function groupsOf(
  view: PendingView,
  leads: readonly ImportsLead[],
  entries: readonly RootEntry[],
  types: readonly WorksContentType[],
): Group[] {
  const shown = view === "ignored" ? [] : leads;
  const roots = [
    ...new Map(
      [
        ...shown.map(({ root, contentType }) => ({ root, contentType })),
        ...entries.map(({ root, contentType }) => ({ root, contentType })),
      ].map((found) => [found.root.id, found]),
    ).values(),
  ];
  return roots.flatMap(({ root, contentType }) =>
    types
      .filter(({ id }) => id === contentType)
      .map((type) => ({
        root,
        type,
        lead: shown.find((lead) => lead.root.id === root.id),
        entries: entries.filter((entry) => entry.root.id === root.id),
      })),
  );
}

function AddAll({ entries }: { entries: readonly RootEntry[] }) {
  const { add, busy } = useAdding();
  const addable = entries.filter((entry) => isAddable(entry));
  if (addable.length < 2) {
    return null;
  }
  return (
    <Button
      busy={busy}
      onClick={() => {
        add(addable);
      }}
    >
      Add all {addable.length}
    </Button>
  );
}

const empties: { readonly [V in PendingView]: string } = {
  "needs-you":
    "No folders need you. A library scan lists here each folder it can't match to a Work with confidence.",
  ignored: "No folders are ignored.",
};

/**
 * Pending imports: what each root's last library scan added and how many
 * folders need the Admin, then each waiting folder with its reason and
 * suggestion. The Ignored view lists the folders the Admin set aside.
 */
export function PendingPage({ view, beside }: { view: PendingView; beside: ReactNode }) {
  const { leads, entries } = useRootImports(pendingParams(view));
  const { contentTypes } = useListContentTypesSuspense().data;
  const groups = groupsOf(view, leads, entries, contentTypes);
  return (
    <Page beside={beside} action={view === "ignored" ? undefined : <AddAll entries={entries} />}>
      {groups.length === 0 ? (
        <Empty>{empties[view]}</Empty>
      ) : (
        groups.map((group) => <RootGroup key={group.root.id} group={group} />)
      )}
    </Page>
  );
}
