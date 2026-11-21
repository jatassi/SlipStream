import { sentenceOf } from "../api/errors";
import { SignalLine } from "../ui/signal-line";
import { Spinner } from "../ui/spinner";

/** A read of the plan in flight, or why it failed. */
export function Reading({
  label,
  pending,
  error,
  className = "",
}: {
  label: string;
  pending: boolean;
  error: unknown;
  className?: string;
}) {
  return pending ? (
    <Spinner label={label} className={className} />
  ) : (
    <div className={className}>
      <SignalLine tone="fail" announce>
        {sentenceOf(error)}
      </SignalLine>
    </div>
  );
}
