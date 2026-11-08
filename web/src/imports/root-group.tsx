import { Link } from "@tanstack/react-router";

import type {
  ImportsEntry,
  ImportsLead,
  ImportsRootLabel,
  WorksContentType,
} from "../api/gen/model";
import { Section } from "../ui/section";
import { EntryRow } from "./entry-row";
import { leadWords } from "./words";

/** One root's lead and the entries in it. */
export type Group = {
  readonly root: ImportsRootLabel;
  readonly type: WorksContentType;
  readonly lead: ImportsLead | undefined;
  readonly entries: readonly ImportsEntry[];
};

function Lead({ lead, type }: { lead: ImportsLead; type: WorksContentType }) {
  return (
    <div className="mb-3 flex flex-col items-start gap-1">
      <p>{leadWords(lead, type)}</p>
      {lead.added > 0 && type.enabled && (
        <Link
          to="/library/$contentType"
          params={{ contentType: type.id }}
          search={{ scanned: 1, root: lead.root.id }}
          className="font-medium text-ink-2 underline hover:text-ink-1"
        >
          Show the {type.work.many.toLowerCase()} it added
        </Link>
      )}
    </div>
  );
}

export function RootGroup({ group }: { group: Group }) {
  return (
    <Section title={group.root.label}>
      {group.lead !== undefined && <Lead lead={group.lead} type={group.type} />}
      {group.entries.length > 0 && (
        <ul
          aria-label={`Pending imports in the ${group.root.label}`}
          className="flex flex-col gap-2"
        >
          {group.entries.map((entry) => (
            <EntryRow key={entry.id} entry={entry} type={group.type} />
          ))}
        </ul>
      )}
    </Section>
  );
}
