import { Meter as BaseMeter } from "@base-ui/react/meter";

import type { AlertTone } from "./count";

const indicators: { readonly [T in AlertTone | "plain"]: string } = {
  plain: "bg-ink-2",
  warn: "bg-warn",
  fail: "bg-fail",
};

// Base UI appends a hidden "x" to the meter's children, as it does to a
// progress bar's (see live-bar.tsx).
export function Meter({
  label,
  value,
  max = 100,
  tone,
}: {
  label: string;
  value: number;
  max?: number;
  tone?: AlertTone | undefined;
}) {
  const track = (
    <BaseMeter.Track className="h-1 overflow-hidden rounded-capsule bg-fill">
      <BaseMeter.Indicator className={`rounded-capsule ${indicators[tone ?? "plain"]}`} />
    </BaseMeter.Track>
  );
  return (
    <BaseMeter.Root
      value={value}
      max={max}
      aria-label={label}
      className="w-full"
      render={(props) => <div {...props}>{track}</div>}
    />
  );
}
