import { Meter as BaseMeter } from "@base-ui/react/meter";

// Base UI appends a hidden "x" to the meter's children, as it does to a
// progress bar's (see live-bar.tsx).
export function Meter({ label, value, max = 100 }: { label: string; value: number; max?: number }) {
  const track = (
    <BaseMeter.Track className="h-1 overflow-hidden rounded-capsule bg-fill">
      <BaseMeter.Indicator className="rounded-capsule bg-ink-2" />
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
