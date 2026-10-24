import { Progress } from "@base-ui/react/progress";

// Base UI appends a hidden "x" to the bar's children, so the fill renders its
// indicator through render, as LiveBar does.
export function RowFill({
  label,
  percent,
  paused,
  className,
}: {
  label: string;
  percent: number;
  paused: boolean;
  className: string;
}) {
  const indicator = (
    <Progress.Indicator
      className={`h-full bg-live-fill transition-[width,opacity] duration-(--duration-live-tick) ease-linear motion-reduce:transition-none ${paused ? "opacity-40" : ""}`}
    />
  );
  return (
    <Progress.Root
      value={percent}
      aria-label={label}
      getAriaValueText={(formatted) => (paused ? `${formatted}, paused` : formatted)}
      render={(props) => (
        <div
          {...props}
          className={`pointer-events-none absolute flex overflow-hidden ${className}`}
        >
          {indicator}
        </div>
      )}
    />
  );
}
