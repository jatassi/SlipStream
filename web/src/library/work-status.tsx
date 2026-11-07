import { Icon } from "../ui/icon";
import { SignalChip } from "../ui/signal-chip";
import type { LibraryWork } from "./rows";
import { type StatusMark, type Tone, marksOf, statusMarks } from "./status";

const inks: { readonly [T in Exclude<Tone, "fail" | "warn">]: string } = {
  quiet: "text-ink-3",
  neutral: "text-ink-2",
  live: "text-ink-1 [&_svg]:text-glow",
  met: "text-ink-1",
};

function Mark({ mark, tally }: { mark: StatusMark; tally: string | undefined }) {
  const words = tally === undefined ? mark.word : `${mark.word} · ${tally}`;
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

/**
 * A Work's status in its words and icons, then a red Failed chip with its
 * count whenever a wanted cell failed, whatever the status says.
 */
export function WorkStatus({ work }: { work: LibraryWork }) {
  const failedMark = statusMarks.failed;
  const marks = marksOf(work).filter((mark) => work.failed === 0 || mark !== failedMark);
  const status = work.status === undefined ? undefined : statusMarks[work.status];
  return (
    <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
      {work.failed > 0 && (
        <SignalChip kind="fail">
          {failedMark.word} <span className="tabular-nums">{work.failed}</span>
        </SignalChip>
      )}
      {marks.map((mark) => (
        <Mark key={mark.word} mark={mark} tally={mark === status ? tallyOf(work) : undefined} />
      ))}
    </span>
  );
}
