import { Mark } from "../ui/state-mark";
import { statusMarks } from "../ui/state-marks";
import type { LibraryWork } from "./rows";
import { marksOf } from "./status";

/** For a Work of many Items, how far along it is, such as "12 of 40 episodes met". */
const tallyOf = (work: LibraryWork) =>
  work.type.items === "many"
    ? `${work.itemStatus.met ?? 0} of ${work.items} ${work.type.item.many.toLowerCase()} met`
    : undefined;

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
