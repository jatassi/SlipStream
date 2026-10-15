import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { Button } from "./button";
import { Data } from "./data";
import { Icon } from "./icon";
import { Sheet } from "./sheet";
import { useShell } from "./shell-context";
import { SignalChip } from "./signal-chip";

export type WarningLineProps = {
  chip: string;
  subject: string;
  detail: string;
  since: string;
  action: { label: string; onClick: () => void };
};

const line =
  "rounded-card bg-warn-card text-(length:--body-size) leading-(--body-line-height) text-warn-text ring-1 ring-warn-line ring-inset";

export function WarningLine(props: WarningLineProps) {
  return useShell() === "phone" ? <WarningRow {...props} /> : <DesktopLine {...props} />;
}

function DesktopLine({ chip, subject, detail, since, action }: WarningLineProps) {
  return (
    <div className={`flex flex-wrap items-center gap-x-3 gap-y-2 px-5 py-3 ${line}`}>
      <SignalChip kind="warn">{chip}</SignalChip>
      <span className="font-bold">{subject}</span>
      <span className="min-w-0 flex-[1_1_16ch] text-warn-dim">{detail}</span>
      <Since since={since} />
      <Button look="warn" onClick={action.onClick}>
        {action.label}
      </Button>
    </div>
  );
}

function WarningRow({ chip, subject, detail, since, action }: WarningLineProps) {
  const [open, setOpen] = useState(false);
  return (
    <Sheet
      title={subject}
      open={open}
      onOpenChange={setOpen}
      trigger={
        <button
          type="button"
          className={`flex min-h-(--control-height) w-full items-center gap-3 px-4 py-3 text-left ${line}`}
        >
          <SignalChip kind="warn">{chip}</SignalChip>
          <span className="min-w-0 flex-1 truncate font-bold">{subject}</span>
          <Icon icon={ArrowRight01Icon} className="size-4 shrink-0 text-warn-dim" />
        </button>
      }
    >
      <div className={`flex flex-col items-start gap-3 p-4 ${line}`}>
        <SignalChip kind="warn">{chip}</SignalChip>
        <p className="text-warn-dim">{detail}</p>
        <Since since={since} />
      </div>
      <Button
        look="warn"
        className="w-full"
        onClick={() => {
          setOpen(false);
          action.onClick();
        }}
      >
        {action.label}
      </Button>
    </Sheet>
  );
}

function Since({ since }: { since: string }) {
  return (
    <span className="whitespace-nowrap text-warn-dim">
      since <Data>{since}</Data>
    </span>
  );
}
