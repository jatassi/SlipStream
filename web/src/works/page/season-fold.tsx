import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import type { ReactNode } from "react";

import { countWords } from "../../format";
import { usePref, writePref } from "../../shell/prefs";
import { Icon } from "../../ui/icon";

/** One season of a series' files, drawn only while open. */
export type Season = {
  /** null for files with no Aired season. */
  readonly season: number | null;
  readonly files: number;
  /** Its wanted subtitles that nothing meets yet. */
  readonly missing: number;
  readonly rows: () => ReactNode;
};

const keyOf = (season: number | null) => (season === null ? "other" : String(season));

const files = { one: "file", many: "files" };
const missingNoun = { one: "missing subtitle", many: "missing subtitles" };

function Fold({ season, open, onToggle }: { season: Season; open: boolean; onToggle: () => void }) {
  const name = season.season === null ? "Other files" : `Season ${season.season}`;
  return (
    <section aria-label={name} className="flex flex-col gap-2">
      <button
        type="button"
        aria-expanded={open}
        onClick={onToggle}
        className="-mx-2 flex items-center gap-2 self-start rounded-item px-2 py-1 text-heading text-ink-1 hover:bg-fill"
      >
        <Icon
          icon={ArrowRight01Icon}
          className={`size-4 shrink-0 transition-[rotate] duration-(--duration-chevron) ease-standard motion-reduce:transition-none ${open ? "rotate-90" : ""}`}
        />
        {name}
        <span className="text-label text-ink-3 tabular-nums">
          {countWords(season.files, files)}
          {season.missing > 0 && ` · ${countWords(season.missing, missingNoun)}`}
        </span>
      </button>
      {open && season.rows()}
    </section>
  );
}

/**
 * A series' files folded per season. Until the Admin opens or closes one,
 * the first season with a missing subtitle is open; the choice is kept per
 * Work in this browser.
 */
export function SeasonFolds({ workId, seasons }: { workId: string; seasons: readonly Season[] }) {
  const folds = usePref("seasonFolds");
  const firstMissing = seasons.find(({ missing }) => missing > 0);
  const open = new Set(
    folds[workId] ?? (firstMissing === undefined ? [] : [keyOf(firstMissing.season)]),
  );
  const toggle = (key: string) => {
    const next = open.has(key) ? [...open].filter((each) => each !== key) : [...open, key];
    writePref("seasonFolds", { ...folds, [workId]: next });
  };
  return (
    <div className="flex flex-col gap-3">
      {seasons.map((season) => {
        const key = keyOf(season.season);
        return (
          <Fold
            key={key}
            season={season}
            open={open.has(key)}
            onToggle={() => {
              toggle(key);
            }}
          />
        );
      })}
    </div>
  );
}
