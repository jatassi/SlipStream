import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { type ReactNode, useState } from "react";

import { Button, type ButtonLook } from "./button";
import type { AlertTone } from "./count";
import { Data } from "./data";
import { Icon } from "./icon";
import { Sheet } from "./sheet";
import { useShell } from "./shell-context";
import { SignalChip } from "./signal-chip";
import { signalLook } from "./signal-row";

/** The one fix a signal offers, as a Warn or Fail button. */
export type CalloutAction = {
  readonly label: string;
  readonly onClick: () => void;
  readonly busy?: boolean;
};

type Shared = {
  /** Names the condition, such as Changed or Restart needed. Every Callout has one (Never Alone Rule). */
  readonly chip: string;
  /**
   * A short headline, set bold, for a standing warning on a page. With one,
   * the children read as its detail in the dim ink, and the phone shell shows
   * the Callout as a row that opens a sheet titled with it.
   */
  readonly subject?: string;
  /** When the condition started, as sinceWords gives it. */
  readonly since?: string;
  /** The sentence, or a body such as a list of files. */
  readonly children: ReactNode;
};

export type CalloutProps = Shared &
  (
    | { readonly tone: "warn"; readonly action?: CalloutAction }
    | {
        readonly tone: "fail";
        readonly action?: CalloutAction;
        /** Buttons after the fix, in the on-fail look, as a failure card has. */
        readonly more?: readonly CalloutAction[];
      }
  );

const card = "rounded-card text-(length:--body-size) leading-(--body-line-height)";

type Shown = { readonly action: CalloutAction; readonly look: ButtonLook };

/** The fix in the tone's own look, then a red Callout's other buttons in on-fail. */
function buttonsOf(props: CalloutProps): readonly Shown[] {
  const fix: Shown[] =
    props.action === undefined ? [] : [{ action: props.action, look: props.tone }];
  const more = props.tone === "fail" ? (props.more ?? []) : [];
  return [...fix, ...more.map((action): Shown => ({ action, look: "on-fail" }))];
}

function Since({ since, tone }: { since: string; tone: AlertTone }) {
  return (
    <span className={`whitespace-nowrap ${signalLook[tone].dim}`}>
      since <Data>{since}</Data>
    </span>
  );
}

function Line(props: CalloutProps) {
  const { tone, chip, subject, since, children } = props;
  const look = signalLook[tone];
  return (
    <output
      className={`flex flex-wrap items-baseline gap-x-3 gap-y-2 px-4 py-3 ${card} ${look.row}`}
    >
      <SignalChip kind={tone}>{chip}</SignalChip>
      {subject !== undefined && <span className="font-bold">{subject}</span>}
      <div className={`min-w-0 flex-[1_1_24ch] ${subject === undefined ? "" : look.dim}`}>
        {children}
      </div>
      {since !== undefined && <Since since={since} tone={tone} />}
      {buttonsOf(props).map(({ action, look: button }) => (
        <Button
          key={action.label}
          look={button}
          busy={action.busy ?? false}
          onClick={action.onClick}
        >
          {action.label}
        </Button>
      ))}
    </output>
  );
}

function Row(props: CalloutProps & { readonly subject: string }) {
  const [open, setOpen] = useState(false);
  const { tone, chip, subject, since, children } = props;
  const look = signalLook[tone];
  return (
    <Sheet
      title={subject}
      open={open}
      onOpenChange={setOpen}
      trigger={
        <button
          type="button"
          className={`flex min-h-(--control-height) w-full items-center gap-3 px-4 py-3 text-left ${card} ${look.row}`}
        >
          <SignalChip kind={tone}>{chip}</SignalChip>
          <span className="min-w-0 flex-1 truncate font-bold">{subject}</span>
          <Icon icon={ArrowRight01Icon} className={`size-4 shrink-0 ${look.dim}`} />
        </button>
      }
    >
      <div className={`flex flex-col items-start gap-3 p-4 ${card} ${look.row}`}>
        <SignalChip kind={tone}>{chip}</SignalChip>
        <div className={look.dim}>{children}</div>
        {since !== undefined && <Since since={since} tone={tone} />}
      </div>
      {buttonsOf(props).map(({ action, look: button }) => (
        <Button
          key={action.label}
          look={button}
          className="w-full"
          onClick={() => {
            setOpen(false);
            action.onClick();
          }}
        >
          {action.label}
        </Button>
      ))}
    </Sheet>
  );
}

/**
 * An amber or red card that names a condition to weigh: a chip, the sentence,
 * and at most one fix. A Signal line reports what was just done instead.
 */
export function Callout(props: CalloutProps) {
  const phone = useShell() === "phone";
  return phone && props.subject !== undefined ? (
    <Row {...props} subject={props.subject} />
  ) : (
    <Line {...props} />
  );
}
