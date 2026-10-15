import { Alert02Icon } from "@hugeicons/core-free-icons";

import { Data } from "./data";
import { Icon } from "./icon";

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
      <span className="inline-flex h-6 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-chip bg-warn px-2 text-chip font-bold text-warn-deep">
        <Icon icon={Alert02Icon} className="size-3.5" />
        {chip}
      </span>
      <span className="font-bold">{subject}</span>
      <span className="min-w-0 flex-[1_1_16ch] text-warn-dim">{detail}</span>
      <span className="whitespace-nowrap text-warn-dim">
        since <Data>{since}</Data>
      </span>
      <button
        type="button"
        onClick={action.onClick}
        className="inline-flex h-(--control-height) items-center rounded-capsule bg-warn px-3.5 text-(length:--body-size) leading-none font-semibold whitespace-nowrap text-warn-deep transition-transform duration-(--duration-control) ease-standard active:scale-97 motion-reduce:transition-none"
      >
        {action.label}
      </button>
    </div>
  );
}
