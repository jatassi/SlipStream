import { sentenceOf } from "../api/errors";
import { AlertLine } from "../ui/alert-line";
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
      <AlertLine text={sentenceOf(error)} />
    </div>
  );
}
