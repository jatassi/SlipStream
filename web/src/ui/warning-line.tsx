import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { Button } from "./button";
import type { AlertTone } from "./count";
import { Data } from "./data";
import { Icon } from "./icon";
import { Sheet } from "./sheet";
import { useShell } from "./shell-context";
import { SignalChip } from "./signal-chip";

export type WarningLineProps = {
  signal: AlertTone;
  chip: string;
  subject: string;
  detail: string;
  since?: string;
  action: { label: string; onClick: () => void };
};

const looks: { readonly [S in AlertTone]: { line: string; dim: string } } = {
  warn: {
    line: "rounded-card bg-warn-card text-(length:--body-size) leading-(--body-line-height) text-warn-text ring-1 ring-warn-line ring-inset",
    dim: "text-warn-dim",
  },
  fail: {
    line: "rounded-card bg-fail-card text-(length:--body-size) leading-(--body-line-height) text-fail-text ring-1 ring-fail-line ring-inset",
    dim: "text-fail-dim",
  },
};

export function WarningLine(props: WarningLineProps) {
  return useShell() === "phone" ? <WarningRow {...props} /> : <DesktopLine {...props} />;
}

function DesktopLine({ signal, chip, subject, detail, since, action }: WarningLineProps) {
  const { line, dim } = looks[signal];
  return (
    <div className={`flex flex-wrap items-center gap-x-3 gap-y-2 px-5 py-3 ${line}`}>
      <SignalChip kind={signal}>{chip}</SignalChip>
      <span className="font-bold">{subject}</span>
      <span className={`min-w-0 flex-[1_1_16ch] ${dim}`}>{detail}</span>
      {since !== undefined && <Since since={since} dim={dim} />}
      <Button look={signal} onClick={action.onClick}>
        {action.label}
      </Button>
    </div>
  );
}

function WarningRow({ signal, chip, subject, detail, since, action }: WarningLineProps) {
  const [open, setOpen] = useState(false);
  const { line, dim } = looks[signal];
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
          <SignalChip kind={signal}>{chip}</SignalChip>
          <span className="min-w-0 flex-1 truncate font-bold">{subject}</span>
          <Icon icon={ArrowRight01Icon} className={`size-4 shrink-0 ${dim}`} />
        </button>
      }
    >
      <div className={`flex flex-col items-start gap-3 p-4 ${line}`}>
        <SignalChip kind={signal}>{chip}</SignalChip>
        <p className={dim}>{detail}</p>
        {since !== undefined && <Since since={since} dim={dim} />}
      </div>
      <Button
        look={signal}
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

function Since({ since, dim }: { since: string; dim: string }) {
  return (
    <span className={`whitespace-nowrap ${dim}`}>
      since <Data>{since}</Data>
    </span>
  );
}
