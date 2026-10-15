import { Progress } from "@base-ui/react/progress";

// Base UI appends a hidden "x" to the bar's children, which leaks into the
// text of the row around it, so the bar renders its track as its only child.
export function LiveBar({ label, value }: { label: string; value: number }) {
  const track = (
    <Progress.Track className="h-1.5 overflow-hidden rounded-capsule bg-[hsl(var(--h)_60%_80%/.14)] ring-1 ring-[hsl(var(--h)_90%_65%/.32)] ring-inset">
      <Progress.Indicator className="relative overflow-hidden rounded-capsule bg-linear-to-r from-glow to-solid shadow-live-glow transition-[width] duration-(--duration-live-tick) ease-linear after:absolute after:inset-0 after:animate-sheen after:bg-linear-to-r after:from-transparent after:via-[rgba(255,255,255,.55)] after:to-transparent motion-reduce:transition-none motion-reduce:after:animate-none" />
    </Progress.Track>
  );
  return (
    <Progress.Root
      value={value}
      aria-label={label}
      className="w-full"
      render={(props) => <div {...props}>{track}</div>}
    />
  );
}
