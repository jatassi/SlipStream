import { ArrowDown01Icon } from "@hugeicons/core-free-icons";
import { memo } from "react";

import { Icon } from "../../ui/icon";
import { RowSwitch } from "../../ui/row-switch";
import { StateBar } from "../../ui/state-square";
import { ColumnNames } from "./columns";
import type { LevelTarget, LevelView } from "./model";

export type HeaderActions = {
  readonly toggle: (level: LevelView) => void;
  readonly monitor: (level: LevelView, on: boolean) => void;
};

/** "6 of 10 met", "Not monitored", or "No longer listed by TVDB" for a withdrawn level. */
export function tallyWords(level: LevelView, source: string): string {
  if (level.level.withdrawnAt !== null) {
    return `No longer listed by ${source}`;
  }
  return level.tally === null ? "Not monitored" : `${level.tally.met} of ${level.tally.of} met`;
}

function Poster({ url }: { url: string | undefined }) {
  return (
    <span className="block h-11 w-7.5 shrink-0 overflow-hidden rounded-[4px] bg-glass-raised ring-1 ring-hairline ring-inset">
      {url !== undefined && (
        <img src={`${url}?size=small`} alt="" loading="lazy" className="size-full object-cover" />
      )}
    </span>
  );
}

function Title({
  level,
  open,
  phone,
  source,
  onToggle,
}: {
  level: LevelView;
  open: boolean;
  phone: boolean;
  source: string;
  onToggle: () => void;
}) {
  return (
    <h3 className="flex min-w-0 flex-1 flex-col gap-1">
      <button
        type="button"
        aria-expanded={open}
        onClick={onToggle}
        className="flex min-w-0 items-center gap-2 text-left font-semibold text-ink-1"
      >
        <span className="truncate">{level.level.label}</span>
        <Icon
          icon={ArrowDown01Icon}
          className={`size-4 shrink-0 text-ink-3 transition-[rotate] duration-(--duration-chevron) motion-reduce:transition-none ${open ? "" : "-rotate-90"}`}
        />
      </button>
      <span className="flex min-w-0 items-center gap-3 text-meta font-normal text-ink-3 tabular-nums">
        {!phone && <StateBar counts={level.counts} />}
        <span className="truncate">{tallyWords(level, source)}</span>
      </span>
    </h3>
  );
}

/** A withdrawn level has no switch, only its place. */
function LevelSwitch({ level, actions }: { level: LevelView; actions: HeaderActions }) {
  return level.level.withdrawnAt === null ? (
    <RowSwitch
      label={`Monitor ${level.level.label}`}
      checked={level.monitor}
      onCheckedChange={(on) => {
        actions.monitor(level, on);
      }}
    />
  ) : (
    <span className="w-8.5 shrink-0" />
  );
}

/**
 * A level's header, its card's top or the whole closed card: poster, name,
 * state bar and tally, the column names while open, then its Monitor switch
 * and a Search slot.
 */
export const LevelHeader = memo(function LevelHeader({
  level,
  open,
  first,
  phone,
  targets,
  source,
  actions,
}: {
  level: LevelView;
  open: boolean;
  first: boolean;
  phone: boolean;
  targets: readonly LevelTarget[];
  source: string;
  actions: HeaderActions;
}) {
  return (
    <div className={first ? "" : "pt-3"}>
      <div
        className={`flex h-14 items-center gap-3 bg-glass px-4 ${open ? "rounded-t-card" : "rounded-card"}`}
      >
        <Poster url={level.poster} />
        <Title
          level={level}
          open={open}
          phone={phone}
          source={source}
          onToggle={() => {
            actions.toggle(level);
          }}
        />
        {open && !phone && <ColumnNames columns={level.columns} targets={targets} />}
        <LevelSwitch level={level} actions={actions} />
        {!phone && (
          <span
            aria-hidden="true"
            className="w-[calc(var(--spacing)*11+var(--icon-button-size))] shrink-0"
          />
        )}
      </div>
    </div>
  );
});

export function FooterRow() {
  return <div className="h-3 rounded-b-card bg-glass" />;
}
