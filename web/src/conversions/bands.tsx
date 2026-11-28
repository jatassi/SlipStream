import type { ConversionsConversionList } from "../api/gen/model";
import { Empty } from "../ui/empty";
import { Section } from "../ui/section";
import { AttentionRow } from "./attention-row";
import { ConvertingRow } from "./converting-row";
import { type More, UpNext } from "./up-next";
import { attentionAnchor } from "./words";

/**
 * Needs attention, then Converting, then Up next by lane, each only when it
 * has rows, or what to do when there is nothing to show.
 */
export function Bands({
  first,
  pages,
  more,
}: {
  first: ConversionsConversionList;
  pages: readonly ConversionsConversionList[];
  more: More;
}) {
  const { converting, paused, waiting, failed } = first.counts;
  if (converting + paused + waiting + failed === 0) {
    return (
      <Empty look="sheet">
        Conversions appear here once a Profile has a Target made from another Target&apos;s file.
      </Empty>
    );
  }
  if (first.attention.length + first.converting.length + first.next.length === 0) {
    return <Empty look="sheet">No Conversion matches these filters.</Empty>;
  }
  return (
    <>
      {first.attention.length > 0 && (
        <Section id={attentionAnchor} title="Needs attention" count={first.attention.length}>
          <ul className="flex flex-col gap-1.5">
            {first.attention.map((row) => (
              <li key={row.id}>
                <AttentionRow row={row} />
              </li>
            ))}
          </ul>
        </Section>
      )}
      {first.converting.length > 0 && (
        <Section title="Converting" count={first.converting.length}>
          <ul className="flex flex-col gap-1.5">
            {first.converting.map((row) => (
              <li key={row.id}>
                <ConvertingRow row={row} />
              </li>
            ))}
          </ul>
        </Section>
      )}
      <UpNext pages={pages} more={more} />
    </>
  );
}
