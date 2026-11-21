import {
  Clock01Icon,
  DashedLineCircleIcon,
  MinusSignCircleIcon,
  MoreHorizontalIcon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { useState } from "react";

import type { SubtitlesWantView } from "../../api/gen/model";
import { MarkBadDialog } from "../../subtitles/mark-bad";
import { openSubtitleSearch, openSubtitleUpload } from "../../subtitles/search-host";
import { useFileActions } from "../../subtitles/work-api";
import {
  type Reading,
  type RowState,
  rowStateOf,
  rowWords,
  stateLine,
  wantLabel,
} from "../../subtitles/work-words";
import { Button } from "../../ui/button";
import { useClock } from "../../ui/clock";
// oxlint-disable-next-line import/max-dependencies -- a row reads its want, words its state and offers every subtitle action
import { Icon } from "../../ui/icon";
import { IconButton } from "../../ui/icon-button";
import { type MenuEntry, Menu } from "../../ui/menu";
import { useShell } from "../../ui/shell-context";
import { nameOf } from "../paths";
import { SidecarDetails, type SubtitleFile, stemOf } from "./subtitle-details";

const marks: {
  readonly [S in RowState]: { readonly icon: IconSvgElement; readonly ink: string };
} = {
  met: { icon: Tick02Icon, ink: "text-ink-1" },
  embedded: { icon: Tick02Icon, ink: "text-ink-1" },
  not_needed: { icon: MinusSignCircleIcon, ink: "text-ink-3" },
  waiting: { icon: Clock01Icon, ink: "text-ink-2" },
  missing: { icon: DashedLineCircleIcon, ink: "text-ink-2" },
};

/** A Wanted Language's state as an icon and its name; the state's word is read aloud and on hover. */
function LanguageMark({ view }: { view: SubtitlesWantView }) {
  const state = rowStateOf(view);
  const { icon, ink } = marks[state];
  return (
    <span
      title={rowWords[state]}
      className={`inline-flex h-6 items-center gap-1 rounded-chip bg-fill px-2 text-chip whitespace-nowrap ${ink}`}
    >
      <Icon icon={icon} className="size-3.5 shrink-0" />
      {wantLabel(view.wanted)}
      <span className="sr-only">, {rowWords[state]}</span>
    </span>
  );
}

type Act = { readonly label: string; readonly disabled: boolean; readonly run: () => void };

function useActs(view: SubtitlesWantView, file: SubtitleFile, markBad: () => void) {
  const { state, wanted, want } = view;
  const at = { workId: file.workId, fileId: file.subtitles.fileId, want };
  const actions = useFileActions(at.fileId);
  const subject = {
    ...at,
    wanted,
    fileName: nameOf(file.file.path),
    providers: file.providers,
  };
  const noHolder = state.missing === "no_holder";
  const wanting = state.kind === "missing" || state.kind === "met";
  const buttons: Act[] = [
    ...(state.kind === "missing"
      ? [{ label: "Search", disabled: noHolder, run: () => void actions.search(want, wanted) }]
      : []),
    ...(wanting
      ? [
          {
            label: "Manual search",
            disabled: noHolder,
            run: () => {
              openSubtitleSearch(subject);
            },
          },
          {
            label: "Upload",
            disabled: noHolder,
            run: () => {
              openSubtitleUpload(subject);
            },
          },
        ]
      : []),
  ];
  const { track } = state;
  const more: Act[] = [
    ...(state.sidecar === null ? [] : [{ label: "Mark bad…", disabled: false, run: markBad }]),
    ...(state.kind === "embedded" && track !== null
      ? [
          {
            label: "Don't count this track",
            disabled: false,
            run: () => void actions.ignore(track),
          },
        ]
      : []),
  ];
  return { buttons, more };
}

const entriesOf = (acts: readonly Act[]): MenuEntry[] =>
  acts.map(({ label, disabled, run }) => ({ label, disabled, onSelect: run }));

function RowActions({
  view,
  file,
  markBad,
}: {
  view: SubtitlesWantView;
  file: SubtitleFile;
  markBad: () => void;
}) {
  const { buttons, more } = useActs(view, file, markBad);
  const label = `More for ${wantLabel(view.wanted)} subtitles`;
  if (useShell() === "phone") {
    const all = [...buttons, ...more];
    return all.length === 0 ? null : (
      <Menu
        label={label}
        trigger={<IconButton icon={MoreHorizontalIcon} label={label} />}
        items={entriesOf(all)}
      />
    );
  }
  return (
    <span className="flex shrink-0 items-center gap-1">
      {buttons.map(({ label: words, disabled, run }) => (
        <Button key={words} look="ghost" size="small" disabled={disabled} onClick={run}>
          {words}
        </Button>
      ))}
      {more.length > 0 && (
        <Menu
          label={label}
          trigger={<IconButton icon={MoreHorizontalIcon} label={label} />}
          items={entriesOf(more)}
        />
      )}
    </span>
  );
}

function WantRow({
  view,
  file,
  reading,
}: {
  view: SubtitlesWantView;
  file: SubtitleFile;
  reading: Reading;
}) {
  const [marking, setMarking] = useState(false);
  const { sidecar } = view.state;
  return (
    <li className="flex flex-col gap-1 py-2">
      <div className="flex min-w-0 flex-wrap items-start gap-x-3 gap-y-1">
        <div className="basis-full sm:basis-auto">
          <LanguageMark view={view} />
        </div>
        <p className="min-w-0 flex-1 basis-60 text-ink-2">{stateLine(view, reading)}</p>
        <RowActions
          view={view}
          file={file}
          markBad={() => {
            setMarking(true);
          }}
        />
      </div>
      <SidecarDetails view={view} file={file} />
      {sidecar !== null && (
        <MarkBadDialog
          at={{ workId: file.workId, fileId: file.subtitles.fileId, want: view.want }}
          wanted={view.wanted}
          sidecar={sidecar}
          version={file.version}
          stem={stemOf(file.file.path)}
          open={marking}
          onOpenChange={setMarking}
        />
      )}
    </li>
  );
}

function useReading(file: SubtitleFile): Reading {
  return {
    clock: useClock(),
    providers: file.providers,
    noun: file.noun,
    kind: file.subtitles.season === null ? "feature" : "episode",
    noAudio: file.file.media?.audio.length === 0,
  };
}

function WantRows({ file }: { file: SubtitleFile }) {
  const reading = useReading(file);
  return (
    <ul aria-label="Subtitles" className="flex flex-col divide-y divide-hairline">
      {file.subtitles.wants.map((view) => (
        <WantRow key={view.want} view={view} file={file} reading={reading} />
      ))}
    </ul>
  );
}

/**
 * A file's subtitles, one row per Wanted Language in Profile order: its
 * state, its one sentence, and Search, Manual search and Upload. An episode
 * shows only its language chips, which open into the same rows.
 */
export function SubtitlesBlock({ file, compact }: { file: SubtitleFile; compact: boolean }) {
  const [open, setOpen] = useState(false);
  const { wants } = file.subtitles;
  if (wants.length === 0) {
    return null;
  }
  if (!compact) {
    return <WantRows file={file} />;
  }
  return (
    <div className="flex flex-col gap-1">
      <button
        type="button"
        aria-expanded={open}
        aria-label={`Subtitles: ${wants.map((view) => `${wantLabel(view.wanted)}, ${rowWords[rowStateOf(view)]}`).join("; ")}`}
        onClick={() => {
          setOpen(!open);
        }}
        className="-mx-1 flex flex-wrap items-center gap-1 self-start rounded-item px-1 py-0.5 hover:bg-fill"
      >
        {wants.map((view) => (
          <LanguageMark key={view.want} view={view} />
        ))}
      </button>
      {open && <WantRows file={file} />}
    </div>
  );
}
