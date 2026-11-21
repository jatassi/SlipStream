import { SignalLine } from "../ui/signal-line";

/** A Test's outcome as the sheet shows it: the pass with any warnings, or the failure's sentence. */
export type TestOutcome =
  | { readonly passed: true; readonly sentence: string; readonly warnings?: readonly string[] }
  | { readonly passed: false; readonly sentence: string };

/** The Test line under a settings sheet's fields: a pass with its warnings, or an alert with the failure. */
export function TestLine({ outcome }: { outcome: TestOutcome }) {
  if (!outcome.passed) {
    return (
      <SignalLine tone="fail" announce>
        {outcome.sentence}
      </SignalLine>
    );
  }
  return (
    <output className="flex flex-col gap-1.5">
      <SignalLine tone="ok">{outcome.sentence}</SignalLine>
      {outcome.warnings?.map((warning) => (
        <SignalLine key={warning} tone="warn">
          {warning}
        </SignalLine>
      ))}
    </output>
  );
}
