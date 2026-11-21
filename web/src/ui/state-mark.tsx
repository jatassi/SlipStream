import { NotificationOff01Icon } from "@hugeicons/core-free-icons";

import type { FilesState } from "../api/gen/model";
import { Defined } from "./defined";
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

/**
 * A status's word and icon, with Failed's count and a tally such as "12 of 40 met" after it.
 * A mark whose row has a definition is a button that opens it on hover, tap and Enter.
 */
export function Mark({
  mark,
  count,
  tally,
}: {
  mark: StatusMark;
  count?: number | undefined;
  tally?: string | undefined;
}) {
  return (
    <Defined term={mark}>
      {(word) => {
        const words = (
          <>
            {word}
            {count !== undefined && <span className="tabular-nums"> {count}</span>}
            {tally !== undefined && ` · ${tally}`}
          </>
        );
        return mark.tone === "fail" || mark.tone === "warn" ? (
          <SignalChip kind={mark.tone}>{words}</SignalChip>
        ) : (
          <span
            className={`inline-flex items-center gap-1 text-meta whitespace-nowrap ${inks[mark.tone]}`}
          >
            <Icon icon={mark.icon} className="size-3.5 shrink-0" />
            {words}
          </span>
        );
      }}
    </Defined>
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
