import { AlertCircleIcon } from "@hugeicons/core-free-icons";
import { type ReactNode, useId, useState } from "react";

import { Data } from "./data";
import { Icon } from "./icon";
import { useShell } from "./shell-context";
import { type Stage, StageTrack, stageLabel } from "./stage-track";

export type CardAction = { label: string; onClick: () => void };

export type Fact = { label: string; value: ReactNode };

type Form = "full" | "compact" | "phone";

const noActions: readonly CardAction[] = [];

const actionLook = {
  fix: "bg-fail text-fail-deep",
  other: "bg-[rgba(255,220,216,.1)] text-fail-text ring-1 ring-[rgba(255,220,216,.16)] ring-inset",
} as const;

function CardButton({
  look,
  children,
  ...props
}: {
  look: keyof typeof actionLook;
  children: ReactNode;
  onClick: () => void;
  "aria-expanded"?: boolean;
}) {
  return (
    <button
      type="button"
      {...props}
      className={`inline-flex h-(--control-height) items-center rounded-capsule px-3.5 text-(length:--body-size) leading-none font-semibold whitespace-nowrap transition-transform duration-(--duration-control) ease-standard active:scale-97 motion-reduce:transition-none ${actionLook[look]}`}
    >
      {children}
    </button>
  );
}

function FailChip({ stage }: { stage: Stage }) {
  return (
    <span className="inline-flex h-6 items-center gap-1.5 rounded-chip bg-fail px-2 text-chip font-bold text-fail-deep">
      <Icon icon={AlertCircleIcon} className="size-3.5" />
      Failed at {stageLabel[stage]}
    </span>
  );
}

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
          <dd className={`text-fail-text ${columns === "one" ? "mb-2 last:mb-0" : ""}`}>{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function PhoneActions({ fix, actions }: { fix: CardAction; actions: readonly CardAction[] }) {
  const [more, setMore] = useState(false);
  return (
    <div className="flex flex-col gap-2">
      <div className="flex gap-2">
        <CardButton look="fix" onClick={fix.onClick}>
          {fix.label}
        </CardButton>
        {actions.length > 0 && (
          <CardButton
            look="other"
            aria-expanded={more}
            onClick={() => {
              setMore(!more);
            }}
          >
            More
          </CardButton>
        )}
      </div>
      {more && (
        <div className="flex flex-wrap gap-2">
          {actions.map(({ label, onClick }) => (
            <CardButton key={label} look="other" onClick={onClick}>
              {label}
            </CardButton>
          ))}
        </div>
      )}
    </div>
  );
}

function Actions({ fix, actions }: { fix: CardAction; actions: readonly CardAction[] }) {
  return (
    <div className="flex flex-wrap justify-end gap-2">
      <CardButton look="fix" onClick={fix.onClick}>
        {fix.label}
      </CardButton>
      {actions.map(({ label, onClick }) => (
        <CardButton key={label} look="other" onClick={onClick}>
          {label}
        </CardButton>
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
        <FailChip stage={stage} />
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
    </article>
  );
}
