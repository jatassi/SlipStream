import { type ReactNode, useId } from "react";

import { Button } from "./button";
import { Data } from "./data";
import { Menu } from "./menu";
import { useShell } from "./shell-context";
import { SignalChip } from "./signal-chip";
import { type Stage, StageTrack, stageLabel } from "./stage-track";
import { WarningLine, type WarningLineProps } from "./warning-line";

export type CardAction = { label: string; onClick: () => void };

export type Fact = { label: string; value: ReactNode };

type Form = "full" | "compact" | "phone";

const noActions: readonly CardAction[] = [];

function Facts({ facts, columns }: { facts: readonly Fact[]; columns: "pair" | "one" }) {
  return (
    <dl
      className={
        columns === "pair"
          ? "grid grid-cols-[max-content_minmax(0,78ch)] gap-x-4 gap-y-2"
          : "flex flex-col gap-1"
      }
    >
      {facts.map(({ label, value }) => (
        <div key={label} className="contents">
          <dt className="pt-px text-label-caps text-fail-ink uppercase">{label}</dt>
          <dd
            className={`wrap-anywhere text-fail-text ${columns === "one" ? "mb-2 last:mb-0" : ""}`}
          >
            {value}
          </dd>
        </div>
      ))}
    </dl>
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
        <StageTrack title={work} progress={{ kind: "failed", stage }} skipped={track.skipped} />
      )}
    </div>
  );
}

export type FailureCardProps = {
  stage: Stage;
  time: string;
  work: string;
  facts: readonly Fact[];
  fix: CardAction;
  actions?: readonly CardAction[];
  track?: { skipped: readonly Stage[] };
  form?: "full" | "compact";
  warning?: WarningLineProps;
};

export function FailureCard({
  stage,
  time,
  work,
  facts,
  fix,
  actions = noActions,
  track,
  form = "full",
  warning,
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
        <Data className="text-fail-dim">{time}</Data>
        {shown === "full" && (
          <div className="ml-auto">
            <Actions fix={fix} actions={actions} />
          </div>
        )}
      </header>
      {shown === "phone" ? (
        <div className="flex flex-col gap-3">
          {subject}
          <Facts facts={facts} columns="one" />
          <PhoneActions fix={fix} actions={actions} />
        </div>
      ) : (
        <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] gap-6">
          {subject}
          <Facts facts={facts} columns="pair" />
        </div>
      )}
      {shown === "compact" && <Actions fix={fix} actions={actions} />}
      {warning && <WarningLine {...warning} />}
    </article>
  );
}
