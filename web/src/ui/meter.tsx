import { Meter as BaseMeter } from "@base-ui/react/meter";

export function Meter({ label, value, max = 100 }: { label: string; value: number; max?: number }) {
  return (
    <BaseMeter.Root value={value} max={max} aria-label={label} className="w-full">
      <BaseMeter.Track className="h-1 overflow-hidden rounded-capsule bg-fill">
        <BaseMeter.Indicator className="rounded-capsule bg-ink-2" />
      </BaseMeter.Track>
    </BaseMeter.Root>
  );
}
