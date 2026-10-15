import { Collapsible } from "@base-ui/react/collapsible";
import type { ReactNode } from "react";

import { Chevron } from "./chevron";
import { Data } from "./data";
import { useShell } from "./shell-context";
import { SignalChip } from "./signal-chip";

const signalLook = {
  fail: { row: "bg-fail-card text-fail-text shadow-signal-lift", dim: "text-fail-dim" },
  warn: {
    row: "bg-warn-card text-warn-text ring-1 ring-warn-line ring-inset",
    dim: "text-warn-dim",
  },
} as const;

export function SignalRow({
  signal,
  chip,
  subject,
  verdict,
  time,
  children,
}: {
  signal: keyof typeof signalLook;
  chip: string;
  subject: string;
  verdict: string;
  time: string;
  children: ReactNode;
}) {
  const look = signalLook[signal];
  const phone = useShell() === "phone";
  return (
    <Collapsible.Root className={`rounded-card ${look.row}`}>
      <Collapsible.Trigger
        className={`group grid w-full items-center gap-x-3 gap-y-1 rounded-card px-4 py-3 text-left text-(length:--body-size) leading-(--body-line-height) ${
          phone
            ? "grid-cols-[auto_auto_minmax(0,1fr)] [grid-template-areas:'chevron_chip_time'_'._subject_subject'_'._verdict_verdict']"
            : "grid-cols-[auto_auto_minmax(0,1fr)_minmax(0,2fr)_auto] [grid-template-areas:'chevron_chip_subject_verdict_time']"
        }`}
      >
        <span className="flex text-ink-4 [grid-area:chevron]">
          <Chevron />
        </span>
        <span className="flex justify-self-start [grid-area:chip]">
          <SignalChip kind={signal}>{chip}</SignalChip>
        </span>
        <span className="truncate font-semibold [grid-area:subject]">{subject}</span>
        <span className={`line-clamp-2 [grid-area:verdict] ${look.dim}`}>{verdict}</span>
        <Data className={`justify-self-end [grid-area:time] ${look.dim}`}>{time}</Data>
      </Collapsible.Trigger>
      <Collapsible.Panel className="px-4 pb-4">{children}</Collapsible.Panel>
    </Collapsible.Root>
  );
}
