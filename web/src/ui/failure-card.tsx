import { useId } from "react";

import { Button } from "./button";
import { Fact, type FactRow, Facts } from "./facts";
import { Menu } from "./menu";
import { RelativeTime } from "./relative-time";
import { useShell } from "./shell-context";
import { SignalChip } from "./signal-chip";
import { type Stage, StageTrack, stageLabel } from "./stage-track";

export type CardAction = { label: string; onClick: () => void };

type Form = "full" | "compact" | "phone";

const noActions: readonly CardAction[] = [];

function CardFacts({ facts }: { facts: readonly FactRow[] }) {
  return (
    <Facts look="failure">
      {facts.map(({ label, value }) => (
        <Fact key={label} label={label}>
          {value}
        </Fact>
      ))}
    </Facts>
  );
}

function PhoneActions({ fix, actions }: { fix: CardAction; actions: readonly CardAction[] }) {
  return (
    <div className="flex gap-2">
      <Button look="fail" onClick={fix.onClick}>
        {fix.label}
      </Button>
      {actions.length > 0 && (
        <Menu
          label="More actions"
          trigger={<Button look="on-fail">More</Button>}
          items={actions.map(({ label, onClick }) => ({ label, onSelect: onClick }))}
        />
      )}
    </div>
  );
}

function Actions({ fix, actions }: { fix: CardAction; actions: readonly CardAction[] }) {
  return (
    <div className="flex flex-wrap justify-end gap-2">
      <Button look="fail" onClick={fix.onClick}>
        {fix.label}
      </Button>
      {actions.map(({ label, onClick }) => (
        <Button key={label} look="on-fail" onClick={onClick}>
          {label}
        </Button>
      ))}
    </div>
  );
}

function Work({
  id,
  work,
  stage,
  track,
}: {
  id: string;
  work: string;
  stage: Stage;
  track: { skipped: readonly Stage[] } | undefined;
}) {
  return (
    <div className="flex flex-col gap-3">
      <h3 id={id} className="text-card-title text-fail-text">
        {work}
      </h3>
      {track && (
        <StageTrack
          title={work}
          progress={{ kind: "failed", stage }}
          skipped={track.skipped}
          labels="ends"
        />
      )}
    </div>
  );
}

export type FailureCardProps = {
  stage: Stage;
  /** The instant it failed. */
  at: string;
  work: string;
  facts: readonly FactRow[];
  fix: CardAction;
  actions?: readonly CardAction[];
  track?: { skipped: readonly Stage[] };
  form?: "full" | "compact";
};

export function FailureCard({
  stage,
  at,
  work,
  facts,
  fix,
  actions = noActions,
  track,
  form = "full",
}: FailureCardProps) {
  const shown: Form = useShell() === "phone" ? "phone" : form;
  const titleId = useId();
  const subject = (
    <Work id={titleId} work={work} stage={stage} track={shown === "compact" ? undefined : track} />
  );

  return (
    <article
      aria-labelledby={titleId}
      className={`flex flex-col gap-4 bg-fail-card px-5 pt-4 pb-5 text-(length:--body-size) leading-(--body-line-height) text-fail-text shadow-signal-lift ${shown === "phone" ? "rounded-[16px]" : "rounded-card"}`}
    >
      <header className="flex flex-wrap items-center gap-3">
        <SignalChip kind="fail">Failed at {stageLabel[stage]}</SignalChip>
        <span className="text-fail-dim">
          <RelativeTime at={at} form="clock" data />
        </span>
        {shown === "full" && (
          <div className="ml-auto">
            <Actions fix={fix} actions={actions} />
          </div>
        )}
      </header>
      {shown === "phone" ? (
        <div className="flex flex-col gap-3">
          {subject}
          <CardFacts facts={facts} />
          <PhoneActions fix={fix} actions={actions} />
        </div>
      ) : (
        <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] gap-6">
          {subject}
          <CardFacts facts={facts} />
        </div>
      )}
      {shown === "compact" && <Actions fix={fix} actions={actions} />}
    </article>
  );
}
