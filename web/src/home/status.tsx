import type { HealthReport, HomeSummary, ShellCounts } from "../api/gen/model";
import { useMedia, wideDesktopQuery } from "../shell/media";
import { type StatusPart, StatusSentence } from "../ui/status-sentence";
import { Term } from "../ui/term";
import { bodyText } from "../ui/type";
import { counted, signalParts, todayWords } from "./words";

/** The jobs part, which says they wait for Reconciling while a restore is matched. */
function jobsStatus(moving: number, counts: ShellCounts): StatusPart {
  if (moving === 0) {
    return { kind: "plain", text: "Nothing running" };
  }
  if (counts.reconciling === null) {
    return { kind: "live", text: counted(moving, "job running", "jobs running") };
  }
  const jobs = counted(moving, "job", "jobs");
  return {
    kind: "plain",
    text: `${jobs} paused while Reconciling`,
    words: (
      <>
        {jobs} paused while <Term id="reconciling" />
      </>
    ),
  };
}

/** Home's first line: what moves, what failed, what waits and Health's worst, with today's totals beside it in a wide window. */
export function HomeStatus({
  summary,
  counts,
  report,
}: {
  summary: HomeSummary;
  counts: ShellCounts;
  report: HealthReport;
}) {
  const wide = useMedia(wideDesktopQuery);
  const today = todayWords(summary.recent.today);
  return (
    <div className="mb-6 flex flex-wrap items-baseline justify-between gap-x-6 gap-y-2">
      <StatusSentence
        parts={[jobsStatus(summary.jobs.moving, counts), ...signalParts(summary, counts, report)]}
      />
      {wide && today !== undefined && (
        <p className={`text-ink-2 tabular-nums ${bodyText}`}>{today}</p>
      )}
    </div>
  );
}
