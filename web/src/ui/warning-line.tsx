import { Button } from "./button";
import { Data } from "./data";
import { SignalChip } from "./signal-chip";

export function WarningLine({
  chip,
  subject,
  detail,
  since,
  action,
}: {
  chip: string;
  subject: string;
  detail: string;
  since: string;
  action: { label: string; onClick: () => void };
}) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-card bg-warn-card px-5 py-3 text-(length:--body-size) leading-(--body-line-height) text-warn-text ring-1 ring-warn-line ring-inset">
      <SignalChip kind="warn">{chip}</SignalChip>
      <span className="font-bold">{subject}</span>
      <span className="min-w-0 flex-[1_1_16ch] text-warn-dim">{detail}</span>
      <span className="whitespace-nowrap text-warn-dim">
        since <Data>{since}</Data>
      </span>
      <Button look="warn" onClick={action.onClick}>
        {action.label}
      </Button>
    </div>
  );
}
