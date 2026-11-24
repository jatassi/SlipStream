import type { ReachabilityFixView } from "../api/gen/model";
import { dataFont } from "../ui/data";
import { Details } from "../ui/details";

/** What happened, then what to do: the fix's title and step, its command, and the rest under Details. */
export function FixText({ fix }: { fix: ReachabilityFixView }) {
  return (
    <div className="flex min-w-0 flex-col items-start gap-2">
      <p>
        {fix.title} {fix.step}
      </p>
      {fix.command !== undefined && (
        <pre
          className={`max-w-full overflow-x-auto rounded-item bg-well px-3 py-2 text-data whitespace-pre-wrap ${dataFont}`}
        >
          {fix.command}
        </pre>
      )}
      {fix.details !== undefined && (
        <Details>
          <p>{fix.details}</p>
        </Details>
      )}
    </div>
  );
}
