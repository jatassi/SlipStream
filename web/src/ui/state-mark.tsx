import { NotificationOff01Icon } from "@hugeicons/core-free-icons";

import type { FilesState } from "../api/gen/model";
import { Icon } from "./icon";
import { SignalChip } from "./signal-chip";
import { type StatusMark, type Tone, statusMarks } from "./state-marks";

export const notMonitored: StatusMark = {
  word: "Not monitored",
  icon: NotificationOff01Icon,
  tone: "quiet",
};

const inks: { readonly [T in Exclude<Tone, "fail" | "warn">]: string } = {
  quiet: "text-ink-3",
  neutral: "text-ink-2",
  live: "text-ink-1 [&_svg]:text-glow",
  met: "text-ink-1",
};

/** A status's word and icon, with Failed's count and a tally such as "12 of 40 met" after it. */
export function Mark({
  mark,
  count,
  tally,
}: {
  mark: StatusMark;
  count?: number | undefined;
  tally?: string | undefined;
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

/**
 * The state of a set of Items, the most urgent among them, with how many are
 * met while some aren't ("Missing · 8 of 10 met"), then Not monitored. A
 * null state, none of them with a Target, shows no state mark.
 */
export function StateMark({
  state,
  met,
  of,
  monitored,
}: {
  state: FilesState | null;
  met: number;
  of: number;
  monitored: boolean;
}) {
  return (
    <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
      {state !== null && (
        <Mark
          mark={statusMarks[state]}
          tally={of > 1 && met < of ? `${String(met)} of ${String(of)} met` : undefined}
        />
      )}
      {!monitored && <Mark mark={notMonitored} />}
    </span>
  );
}
