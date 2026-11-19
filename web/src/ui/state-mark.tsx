import { Popover as BasePopover } from "@base-ui/react/popover";
import { NotificationOff01Icon } from "@hugeicons/core-free-icons";
import { type ReactNode, useId } from "react";

import type { FilesState } from "../api/gen/model";
import { labelPopup } from "./floating";
import { Icon } from "./icon";
import { hoverOpening } from "./popover";
import { usePortalDensity } from "./portal-density";
import { SignalChip } from "./signal-chip";
import { type Definition, type StatusMark, type Tone, statusMarks } from "./state-marks";

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

const definedWord = "underline decoration-ink-4 decoration-dotted underline-offset-3";

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
  const words = (
    <>
      {mark.definition === undefined ? mark.word : <span className={definedWord}>{mark.word}</span>}
      {count !== undefined && <span className="tabular-nums"> {count}</span>}
      {tally !== undefined && ` · ${tally}`}
    </>
  );
  const look =
    mark.tone === "fail" || mark.tone === "warn" ? (
      <SignalChip kind={mark.tone}>{words}</SignalChip>
    ) : (
      <span
        className={`inline-flex items-center gap-1 text-meta whitespace-nowrap ${inks[mark.tone]}`}
      >
        <Icon icon={mark.icon} className="size-3.5 shrink-0" />
        {words}
      </span>
    );
  return mark.definition === undefined ? (
    look
  ) : (
    <Defined word={mark.word} definition={mark.definition}>
      {look}
    </Defined>
  );
}

const trigger =
  "relative inline-flex rounded-chip text-left before:absolute before:-inset-x-1 before:-inset-y-2 focus-visible:outline-2 focus-visible:outline-offset-2";

const definitionPopup = `max-w-[min(280px,var(--available-width))] text-pretty ${labelPopup}`;

// The definition stays the button's description, so a screen reader hears it on focus without opening it.
function Defined({
  word,
  definition,
  children,
}: {
  word: string;
  definition: Definition;
  children: ReactNode;
}) {
  const id = useId();
  const { density, probe } = usePortalDensity();
  return (
    <>
      {probe}
      <BasePopover.Root>
        <BasePopover.Trigger
          render={
            <button type="button" aria-describedby={id} className={trigger}>
              {children}
            </button>
          }
          {...hoverOpening}
        />
        <span id={id} hidden>
          {definition}
        </span>
        <BasePopover.Portal data-density={density}>
          <BasePopover.Positioner sideOffset={6} collisionPadding={12}>
            <BasePopover.Popup aria-label={word} initialFocus={false} className={definitionPopup}>
              {definition}
            </BasePopover.Popup>
          </BasePopover.Positioner>
        </BasePopover.Portal>
      </BasePopover.Root>
    </>
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
