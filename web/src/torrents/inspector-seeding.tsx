import { Link } from "@tanstack/react-router";

import type {
  DownloadclientsSeedingViewAtRisk,
  DownloadclientsTorrentDetail,
} from "../api/gen/model";
import { InspectorSection } from "../ui/inspector";
import { LaneMeter } from "../ui/lane-meter";
import { SignalChip } from "../ui/signal-chip";
import { FactList } from "./inspector-facts";
import {
  endlessSentence,
  goalSentence,
  lanesOf,
  marginSentence,
  owedWords,
  ruleSentence,
} from "./seeding";

function RiskLine({
  tracker,
  node,
  risk,
}: {
  tracker: string;
  node: string;
  risk: DownloadclientsSeedingViewAtRisk;
}) {
  switch (risk) {
    case "yes":
      return (
        <p className="flex flex-wrap items-center gap-2">
          <SignalChip kind="warn">At risk</SignalChip>
          {`It stopped seeding before ${tracker}'s rule was met. Seed it again before ${tracker} counts a hit and run.`}
        </p>
      );
    case "unknown":
      return (
        <p className="text-ink-2">{`${node} is offline, so whether this torrent is seeding is unknown until ${node} reconnects.`}</p>
      );
    case "no":
      return null;
    default:
      return risk satisfies never;
  }
}

export function Seeding({ detail, node }: { detail: DownloadclientsTorrentDetail; node: string }) {
  const { seeding } = detail;
  const tracker = detail.tracker?.name ?? "its Tracker";
  const sentences = [
    ruleSentence(tracker, seeding),
    seeding.owes === undefined
      ? undefined
      : `It still owes ${owedWords(seeding.owes, seeding.clears)}.`,
    seeding.owes === undefined ? undefined : marginSentence(tracker, seeding),
    goalSentence(seeding),
    endlessSentence(tracker, seeding),
  ].filter((sentence) => sentence !== undefined);
  return (
    <InspectorSection title="Seeding">
      <div className="flex flex-col gap-3">
        <FactList
          facts={[
            [
              "Tracker",
              detail.tracker === null ? (
                "Not known"
              ) : (
                <Link
                  key="tracker"
                  to="/settings/trackers/$trackerId/seeding"
                  params={{ trackerId: detail.tracker.id }}
                  className="rounded-item font-medium underline-offset-4 hover:underline"
                >
                  {detail.tracker.name}
                </Link>
              ),
            ],
          ]}
        />
        {seeding.goalMet && (
          <p>
            <SignalChip kind="ok">Goal met</SignalChip>
          </p>
        )}
        <LaneMeter lanes={lanesOf(seeding)} risk={seeding.atRisk === "yes"} size="large" />
        <RiskLine tracker={tracker} node={node} risk={seeding.atRisk} />
        {sentences.map((sentence) => (
          <p key={sentence} className="text-ink-2">
            {sentence}
          </p>
        ))}
      </div>
    </InspectorSection>
  );
}
