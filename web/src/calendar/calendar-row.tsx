import { Link } from "@tanstack/react-router";

import type { CalendarRow as Row, WorksContentType } from "../api/gen/model";
import { Details } from "../ui/details";
import { ListRow } from "../ui/list";
import { Artwork } from "../ui/artwork";
import { StateMark } from "../ui/state-mark";
import { typeOf } from "../works/fields";

const stretched = "after:absolute after:inset-0 after:rounded-[inherit] after:content-['']";

/** A row's Work, kind and Items, which no other row of its day shares. */
export const rowKey = ({ work, kind, items }: Row): string =>
  `${work.id}:${kind}:${items.map(({ id }) => id).join(",")}`;

/**
 * The row's words: the Items and their level, then the kind, such as
 * "Season 2 · Episodes 1 to 10 · Air date". `withKind` is false when the page
 * shows one kind, which every row would repeat.
 */
export const rowWords = ({ itemsWords, kindLabel }: Row, withKind: boolean): string =>
  [itemsWords, withKind ? kindLabel : ""].filter((part) => part !== "").join(" · ");

/** A row's Items, listed when it opens. */
function RowItems({ row, types }: { row: Row; types: readonly WorksContentType[] }) {
  const items = typeOf([...types], row.work.contentType).item.many;
  return (
    <div className="relative z-1 pl-11">
      <Details label={items}>
        <ul aria-label={items} className="flex flex-col gap-1 pb-1 text-meta">
          {row.items.map((item) => (
            <li key={item.id}>{item.label}</li>
          ))}
        </ul>
      </Details>
    </div>
  );
}

/**
 * One Work's Items of one kind on one day: its poster and label linking to
 * the Work, the Items and kind, and their state. In a narrow container the
 * state sits on its own line under the words. A row of several Items opens
 * to list them.
 */
export function CalendarRow({
  row,
  types,
  withKind,
}: {
  row: Row;
  types: readonly WorksContentType[];
  withKind: boolean;
}) {
  const words = rowWords(row, withKind);
  const mark =
    row.state !== null || !row.monitored ? (
      <StateMark state={row.state} met={row.met} of={row.of} monitored={row.monitored} />
    ) : undefined;
  return (
    <ListRow
      leading={<Artwork use="poster" size="row" url={row.work.posterUrl ?? undefined} />}
      title={
        <Link
          to="/works/$workId"
          params={{ workId: row.work.id }}
          className={`rounded-item hover:underline ${stretched}`}
        >
          {row.work.label}
        </Link>
      }
      meta={words === "" ? undefined : words}
      trailing={
        mark === undefined ? undefined : <span className="hidden @min-[560px]:inline">{mark}</span>
      }
      detail={
        <>
          {mark !== undefined && <div className="pl-11 @min-[560px]:hidden">{mark}</div>}
          {row.items.length > 1 && <RowItems row={row} types={types} />}
        </>
      }
    />
  );
}
