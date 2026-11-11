import { Icon } from "../ui/icon";
import { SignalChip } from "../ui/signal-chip";
import type { LibraryWork } from "./rows";
import { type StatusMark, type Tone, statusMarks } from "../ui/state-marks";
import { marksOf } from "./status";

const inks: { readonly [T in Exclude<Tone, "fail" | "warn">]: string } = {
  quiet: "text-ink-3",
  neutral: "text-ink-2",
  live: "text-ink-1 [&_svg]:text-glow",
  met: "text-ink-1",
};

function Mark({
  mark,
  count,
  tally,
}: {
  mark: StatusMark;
  count: number | undefined;
  tally: string | undefined;
}) {
  const words = (
    <>
      {mark.word}
      {count !== undefined && <span className="tabular-nums"> {count}</span>}
      {tally !== undefined && ` · ${tally}`}
    </>
  );
  if (mark.tone === "fail" || mark.tone === "warn") {
    return <SignalChip kind={mark.tone}>{words}</SignalChip>;
  }
  return (
    <span
      className={`inline-flex items-center gap-1 text-meta whitespace-nowrap ${inks[mark.tone]}`}
    >
      <Icon icon={mark.icon} className="size-3.5 shrink-0" />
      {words}
    </span>
  );
}

/** For a Work of many Items, how far along it is, such as "12 of 40 met". */
const tallyOf = (work: LibraryWork) =>
  work.type.items === "many" ? `${work.itemStatus.met ?? 0} of ${work.items} met` : undefined;

/** A Work's status in its words and icons. Failed carries its count of failed cells, such as "Failed 3". */
export function WorkStatus({ work }: { work: LibraryWork }) {
  const status = work.status === undefined ? undefined : statusMarks[work.status];
  return (
    <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
      {marksOf(work).map((mark) => (
        <Mark
          key={mark.word}
          mark={mark}
          count={mark === statusMarks.failed ? work.cells.failed : undefined}
          tally={mark === status ? tallyOf(work) : undefined}
        />
      ))}
    </span>
  );
}
