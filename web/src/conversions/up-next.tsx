import type { ConversionsConversionList, ConversionsUpNext } from "../api/gen/model";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { useShell } from "../ui/shell-context";
import { VirtualRows } from "../ui/virtual-rows";
import { GroupRow } from "./group-row";
import { lanesOf } from "./load";
import { WaitingRow } from "./waiting-row";
import { laneHeadings } from "./words";

const rowHeights = { desktop: 56, phone: 72 } as const;

const keyOf = (row: ConversionsUpNext) => `${row.kind}:${row.id}`;

export type More = { has: boolean; busy: boolean; next: () => void };

export function UpNext({
  pages,
  more,
}: {
  pages: readonly ConversionsConversionList[];
  more: More;
}) {
  const shell = useShell();
  return (
    <>
      {lanesOf(pages).map(({ lane, rows }) => (
        <Section key={lane} title={laneHeadings[lane]} count={rows.length}>
          <div className="flex max-h-[min(64vh,640px)] flex-col">
            <VirtualRows
              label={laneHeadings[lane]}
              rows={rows}
              rowKey={keyOf}
              estimate={rowHeights[shell]}
              renderRow={(row) => (
                <div className="pb-1.5">
                  {row.kind === "group" ? <GroupRow group={row} /> : <WaitingRow row={row} />}
                </div>
              )}
            />
          </div>
        </Section>
      ))}
      {more.has && (
        <Button look="ghost" busy={more.busy} onClick={more.next}>
          Show more
        </Button>
      )}
    </>
  );
}
