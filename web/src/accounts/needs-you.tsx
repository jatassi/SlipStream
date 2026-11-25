import { Link } from "@tanstack/react-router";

import type { MediaserversAccessView as Access } from "../api/gen/model";
import { useListMediaServerAccessSuspense } from "../api/gen/query/mediaservers/mediaservers";
import { Chip } from "../ui/chip";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { AccessActions, accessLine } from "./access-actions";

function Member({ member }: { member: Access["member"] }) {
  return (
    <span className="flex items-center gap-2">
      {member.accountId === undefined ? (
        member.name
      ) : (
        <Link
          to="/accounts/$accountId"
          params={{ accountId: member.accountId }}
          className="hover:underline"
        >
          {member.name}
        </Link>
      )}
      {member.disabled && <Chip>Turned off</Chip>}
    </span>
  );
}

/** Members' access waiting on an Admin: a share to make or accept, or a removal to try again. */
export function NeedsYou() {
  const rows = useListMediaServerAccessSuspense().data.rows.filter((row) => row.needsYou);
  if (rows.length === 0) {
    return null;
  }
  return (
    <Section title="Needs you" count={rows.length}>
      <List label="Needs you">
        {rows.map((row) => (
          <ListRow
            key={row.id}
            title={<Member member={row.member} />}
            meta={row.server.name}
            detail={
              <div className="mt-1 flex flex-col gap-2">
                <p className="text-pretty text-ink-2">{accessLine(row)}</p>
                <AccessActions row={row} />
              </div>
            }
          />
        ))}
      </List>
    </Section>
  );
}
