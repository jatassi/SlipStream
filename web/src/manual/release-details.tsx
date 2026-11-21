import { use } from "react";

import type { GrabsReleaseDetails } from "../api/gen/model";
import { lostAtSentence } from "../grabs/lost-at-sentence";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { Fact, Facts } from "../ui/facts";
import { RelativeTime } from "../ui/relative-time";
import { ReadingContext } from "./reading";

/**
 * A Release's Details: its raw title, its verdict at each Target, where it
 * ranked below the plan's pick, where it was listed, and Open in tester.
 */
export function ReleaseDetails({ release }: { release: GrabsReleaseDetails }) {
  const { targets, onTest } = use(ReadingContext);
  const lostAt = release.lostAt;
  return (
    <div className="flex flex-col gap-2 py-1">
      <Data className="text-meta break-all text-ink-2">{release.title}</Data>
      <ul aria-label="Verdicts" className="flex list-disc flex-col gap-0.5 pl-5">
        {release.verdicts.map((verdict) => (
          <li key={verdict}>{verdict}</li>
        ))}
      </ul>
      {lostAt !== null && (
        <p>
          {lostAtSentence(
            lostAt,
            "first",
            lostAt.targetId === null ? undefined : targets.get(lostAt.targetId),
          )}
        </p>
      )}
      <div className="text-meta in-data-[density=touch]:text-body-touch">
        <Facts look="inline">
          <Fact label="Indexers">{release.indexers.join(", ")}</Fact>
          {release.tracker !== undefined && <Fact label="Tracker">{release.tracker}</Fact>}
          {release.publishedAt !== null && (
            <Fact label="Published">
              <RelativeTime at={release.publishedAt} form="age" />
            </Fact>
          )}
          <Fact label="Seeders">
            <Data>{release.seeders}</Data>
          </Fact>
        </Facts>
      </div>
      <Button
        look="ghost"
        size="small"
        className="self-start"
        onClick={() => {
          onTest(release.title);
        }}
      >
        Open in tester
      </Button>
    </div>
  );
}
