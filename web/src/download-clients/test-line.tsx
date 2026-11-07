import { Icon } from "../ui/icon";
import { signals } from "../ui/signal-chip";
import type { TestResult } from "./client-form";

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

/** The Test's result: who responded and any warnings on a pass, the Test's own sentence on a fail. */
export function TestLine({ result, node }: { result: TestResult; node: string }) {
  if (!result.passed) {
    return (
      <div role="alert">
        <Line signal="fail" ink="text-fail-ink" text={result.sentence} />
      </div>
    );
  }
  const { product, version, warnings } = result.report;
  return (
    <output className="flex flex-col gap-1.5">
      <Line
        signal="ok"
        ink="text-ink-1"
        text={`The Test passed. ${node} reached ${product} ${version}.`}
      />
      {warnings.map((warning) => (
        <Line key={warning} signal="warn" ink="text-warn" text={warning} />
      ))}
    </output>
  );
}
