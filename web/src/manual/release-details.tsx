import { type ReactNode, use } from "react";

import type { GrabsReleaseDetails } from "../api/gen/model";
import { agoWords, bytes, countWords } from "../format";
import { lostAtSentence } from "../grabs/lost-at-sentence";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { ReadingContext } from "./reading";

const seeders = { one: "seeder", many: "seeders" };

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex gap-2">
      <dt className="w-20 shrink-0 text-ink-3">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  );
}

/**
 * A Release's Details: its raw title, its verdict at each Target, where it
 * ranked below the plan's pick, the facts v1 showed, and Open in tester.
 */
export function ReleaseDetails({ release }: { release: GrabsReleaseDetails }) {
  const { now, targets, onTest } = use(ReadingContext);
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
      <dl className="flex flex-col gap-0.5 text-meta in-data-[density=touch]:text-body-touch">
        <Fact label="Indexers">{release.indexers.join(", ")}</Fact>
        {release.tracker !== undefined && <Fact label="Tracker">{release.tracker}</Fact>}
        <Fact label="Size">
          <Data>{bytes(release.size)}</Data>
        </Fact>
        {release.publishedAt !== null && (
          <Fact label="Published">
            <Data>{agoWords(release.publishedAt, now)}</Data>
          </Fact>
        )}
        <Fact label="Seeders">
          <Data>{countWords(release.seeders, seeders)}</Data>
        </Fact>
      </dl>
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
