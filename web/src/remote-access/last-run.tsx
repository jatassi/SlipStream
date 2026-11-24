import type { ReachabilityLastRun } from "../api/gen/model";
import { runRemoteAccess } from "../api/gen/query/reachability/reachability";
import { listed } from "../targets/words";
import { useAttempt } from "../targets/attempt";
import { Button } from "../ui/button";
import { RelativeTime } from "../ui/relative-time";
import { toast } from "../ui/toast";

export function LastRun({ run }: { run: ReachabilityLastRun | null }) {
  const again = useAttempt(async () => {
    await runRemoteAccess();
    toast("Checking every route again");
  });
  const testers = run?.testers.map((tester) => tester.name) ?? [];
  return (
    <div className="flex flex-col items-start gap-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2 text-ink-2">
        <p>
          {run === null ? (
            "Not checked yet"
          ) : (
            <>
              Last run <RelativeTime at={run.at} form="clock" />
            </>
          )}
          {testers.length > 0 && (
            <>
              {" "}
              <span aria-hidden="true" className="px-1 text-ink-4">
                ·
              </span>{" "}
              Tested from {listed(testers)}
            </>
          )}
        </p>
        <Button size="small" busy={again.busy} onClick={again.run}>
          Run again
        </Button>
      </div>
      {again.failure}
    </div>
  );
}
