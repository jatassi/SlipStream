import { ArrowDown01Icon } from "@hugeicons/core-free-icons";
import { memo } from "react";

import { type Noun, countWords } from "../../format";
import { Icon } from "../../ui/icon";
import { Popover } from "../../ui/popover";
import { StateBar } from "../../ui/state-square";
import { lowered } from "../monitoring/labels";
import { ColumnNames } from "./columns";
import { LevelControls } from "./controls";
import type { LevelSearch } from "./level-search";
// oxlint-disable-next-line import/max-dependencies -- the header holds its name, state bar, tally, search state word and controls
import type { LevelTarget, LevelView } from "./model";

export type HeaderActions = {
  readonly toggle: (level: LevelView) => void;
  readonly monitor: (level: LevelView, on: boolean) => void;
  readonly search: (level: LevelView) => void;
  readonly manual: (level: LevelView) => void;
  readonly grabNow: (level: LevelView, search: LevelSearch) => void;
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

/** The group's state word after the tally; it opens its full sentence when it has one. */
function StateWord({ search }: { search: LevelSearch | undefined }) {
  if (search?.word === undefined) {
    return null;
  }
  if (search.sentence === undefined) {
    return <span className="shrink-0 text-ink-2">{search.word}</span>;
  }
  return (
    <Popover
      label={search.word}
      opensOn="hover"
      trigger={
        <button type="button" className="shrink-0 text-ink-2 underline decoration-dotted">
          {search.word}
        </button>
      }
    >
      <p className="text-ink-1">{search.sentence}</p>
    </Popover>
  );
}

function Title({
  level,
  open,
  phone,
  item,
  search,
  onToggle,
}: {
  level: LevelView;
  open: boolean;
  phone: boolean;
  item: Noun;
  search: LevelSearch | undefined;
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
        <StateWord search={search} />
      </span>
    </h3>
  );
}

/**
 * A level's header, its card's top or the whole closed card: poster, name,
 * state bar, tally and search state word, the column names while open,
 * then its Monitor switch, Search and a menu with Manual search, and Grab
 * now while a cell of it waits for a better Release. A phone leaves out the
 * poster and state bar, so the tally and state word fit beside the controls.
 */
export const LevelHeader = memo(function LevelHeader({
  level,
  open,
  first,
  phone,
  targets,
  item,
  search,
  actions,
}: {
  level: LevelView;
  open: boolean;
  first: boolean;
  phone: boolean;
  targets: readonly LevelTarget[];
  item: Noun;
  search: LevelSearch | undefined;
  actions: HeaderActions;
}) {
  return (
    <div className={first ? "" : "pt-3"}>
      <div
        className={`flex h-14 items-center gap-3 bg-glass px-4 ${open ? "rounded-t-card" : "rounded-card"}`}
      >
        {!phone && <Poster url={level.poster} />}
        <Title
          level={level}
          open={open}
          phone={phone}
          item={item}
          search={search}
          onToggle={() => {
            actions.toggle(level);
          }}
        />
        {open && !phone && <ColumnNames columns={level.columns} targets={targets} />}
        <LevelControls level={level} phone={phone} search={search} actions={actions} />
      </div>
    </div>
  );
});

export function FooterRow() {
  return <div className="h-3 rounded-b-card bg-glass" />;
}
