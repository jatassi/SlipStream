import { ArrowDown01Icon, Search01Icon } from "@hugeicons/core-free-icons";
import { memo } from "react";

import { type Noun, countWords } from "../../format";
import { Icon } from "../../ui/icon";
import { IconButton } from "../../ui/icon-button";
import { RowSwitch } from "../../ui/row-switch";
import { StateBar } from "../../ui/state-square";
import { lowered } from "../monitoring/labels";
import { ColumnNames } from "./columns";
import type { LevelTarget, LevelView } from "./model";

export type HeaderActions = {
  readonly toggle: (level: LevelView) => void;
  readonly monitor: (level: LevelView, on: boolean) => void;
  readonly search: (level: LevelView) => void;
};

/** "6 of 10 met", "Not monitored", or what the withdrawn section holds, "2 episodes". */
function tallyWords(level: LevelView, item: Noun): string {
  if (level.withdrawn) {
    return countWords(level.gone.length, lowered(item));
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
  item,
  onToggle,
}: {
  level: LevelView;
  open: boolean;
  phone: boolean;
  item: Noun;
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
        <span className="truncate">{level.name}</span>
        <Icon
          icon={ArrowDown01Icon}
          className={`size-4 shrink-0 text-ink-3 transition-[rotate] duration-(--duration-chevron) motion-reduce:transition-none ${open ? "" : "-rotate-90"}`}
        />
      </button>
      <span className="flex min-w-0 items-center gap-3 text-meta font-normal text-ink-3 tabular-nums">
        {!phone && <StateBar counts={level.counts} />}
        <span className="truncate">{tallyWords(level, item)}</span>
      </span>
    </h3>
  );
}

/** The withdrawn section has no switch or Search, only their places. */
function LevelControls({ level, actions }: { level: LevelView; actions: HeaderActions }) {
  if (level.withdrawn) {
    return (
      <>
        <span className="w-8.5 shrink-0" />
        <span aria-hidden="true" className="size-(--icon-button-size) shrink-0" />
      </>
    );
  }
  return (
    <>
      <RowSwitch
        label={`Monitor ${level.name}`}
        checked={level.monitor}
        onCheckedChange={(on) => {
          actions.monitor(level, on);
        }}
      />
      <IconButton
        icon={Search01Icon}
        label={`Search ${level.name}`}
        onClick={() => {
          actions.search(level);
        }}
      />
    </>
  );
}

/**
 * A level's header, its card's top or the whole closed card: poster, name,
 * state bar and tally, the column names while open, then its Monitor switch
 * and Search.
 */
export const LevelHeader = memo(function LevelHeader({
  level,
  open,
  first,
  phone,
  targets,
  item,
  actions,
}: {
  level: LevelView;
  open: boolean;
  first: boolean;
  phone: boolean;
  targets: readonly LevelTarget[];
  item: Noun;
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
          item={item}
          onToggle={() => {
            actions.toggle(level);
          }}
        />
        {open && !phone && <ColumnNames columns={level.columns} targets={targets} />}
        <LevelControls level={level} actions={actions} />
        {!phone && <span aria-hidden="true" className="size-(--icon-button-size) shrink-0" />}
      </div>
    </div>
  );
});

export function FooterRow() {
  return <div className="h-3 rounded-b-card bg-glass" />;
}
