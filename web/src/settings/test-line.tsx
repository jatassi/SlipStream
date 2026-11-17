import { Icon } from "../ui/icon";
import { signals } from "../ui/signal-chip";

/** A Test's outcome as the sheet shows it: the pass with any warnings, or the failure's sentence. */
export type TestOutcome =
  | { readonly passed: true; readonly sentence: string; readonly warnings?: readonly string[] }
  | { readonly passed: false; readonly sentence: string };

function Line({
  signal,
  ink,
  text,
}: {
  signal: "ok" | "warn" | "fail";
  ink: string;
  text: string;
}) {
  return (
    <p className={`flex items-start gap-1.5 text-pretty ${ink}`}>
      <Icon icon={signals[signal].icon} className="mt-[0.2em] size-4 shrink-0" />
      {text}
    </p>
  );
}

/** The Test line under a settings sheet's fields: a pass with its warnings, or an alert with the failure. */
export function TestLine({ outcome }: { outcome: TestOutcome }) {
  if (!outcome.passed) {
    return (
      <div role="alert">
        <Line signal="fail" ink="text-fail-ink" text={outcome.sentence} />
      </div>
    );
  }
  return (
    <output className="flex flex-col gap-1.5">
      <Line signal="ok" ink="text-ink-1" text={outcome.sentence} />
      {outcome.warnings?.map((warning) => (
        <Line key={warning} signal="warn" ink="text-warn" text={warning} />
      ))}
    </output>
  );
}
