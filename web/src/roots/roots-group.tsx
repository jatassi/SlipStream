import { Add01Icon } from "@hugeicons/core-free-icons";

import type { NodesNode, RootsNodeVolumes, RootsRoot, WorksContentType } from "../api/gen/model";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { Table } from "../ui/table";
import { rootColumns } from "./root-columns";
import type { Acting } from "./acting-dialogs";
import { stateWords } from "./words";

export type Group = { readonly type: WorksContentType; readonly roots: readonly RootsRoot[] };

/** Every enabled Content Type, and any other that has a root, with its roots oldest first. */
export const groupsOf = (
  roots: readonly RootsRoot[],
  types: readonly WorksContentType[],
): Group[] =>
  types
    .map((type) => ({ type, roots: roots.filter((root) => root.contentType === type.id) }))
    .filter((group) => group.type.enabled || group.roots.length > 0);

/** One Content Type's roots, or the sentence that asks for its first. */
export function RootsGroup({
  group: { type, roots },
  nodes,
  volumes,
  onAct,
}: {
  group: Group;
  nodes: readonly NodesNode[];
  volumes: readonly RootsNodeVolumes[];
  onAct: (acting: Acting) => void;
}) {
  const label = type.work.many;
  return (
    <Section title={label} count={roots.length}>
      {roots.length === 0 ? (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-card bg-glass px-4 py-3.5 ring-1 ring-hairline ring-inset">
          <p className="text-ink-2">Add a library root to hold {label}.</p>
          <Button
            icon={Add01Icon}
            onClick={() => {
              onAct({ kind: "add", contentType: type.id });
            }}
          >
            {`Add a root for ${label}`}
          </Button>
        </div>
      ) : (
        <Table
          label={`${label} roots`}
          columns={rootColumns(type, nodes, volumes, onAct)}
          rows={roots}
          rowKey={(root) => root.id}
          lines={roots.some((root) => root.state !== "ready" || root.rehome !== undefined) ? 2 : 1}
          tone={(root) => (stateWords[root.state].tone === "warn" ? "warn" : undefined)}
          stackBelow={640}
        />
      )}
    </Section>
  );
}
