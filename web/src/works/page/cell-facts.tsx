import type {
  FilesCell,
  FilesHeldCell,
  FilesLibraryFile,
  FilesStep,
  GrabsGrab,
} from "../../api/gen/model";
import { useListNodesSuspense } from "../../api/gen/query/nodes/nodes";
import { bytes } from "../../format";
import { ProgressCell } from "../../grabs/progress-cell";
import { failureWords, isFailed } from "../../grabs/stages";
import { useQualityNames } from "../../targets/quality-names";
import { Data } from "../../ui/data";

function FileFacts({ file }: { file: FilesLibraryFile }) {
  const quality = useQualityNames();
  return (
    <span className="text-meta text-ink-3">
      <Data>
        {quality(file.quality)} · {file.probe.width}×{file.probe.height} · {bytes(file.size)}
      </Data>
    </span>
  );
}

function Replaces({ file }: { file: FilesLibraryFile }) {
  const quality = useQualityNames();
  return <span className="text-meta text-ink-3">Replaces {quality(file.quality)}</span>;
}

/** The step a held cell's Grab or its Import is at; Sending names the Node the download is on. */
const stepWords: { readonly [S in FilesStep]: (node: string) => string } = {
  fetching: () => "Fetching",
  downloading: () => "Downloading",
  reading: () => "Reading",
  sending: (node) => `Sending from ${node}`,
  placing: () => "Importing",
  waiting: () => "Waiting for you",
};

const time = new Intl.DateTimeFormat(undefined, { hour: "numeric", minute: "2-digit" });

/** "Waiting for the import delay, until 4:10 PM." */
export const delayWords = (until: string): string =>
  `Waiting for the import delay, until ${time.format(new Date(until))}.`;

type Held = Pick<FilesHeldCell, "grabId" | "until"> & { readonly step: FilesStep | null };

function HeldFacts({ held, grabs }: { held: Held; grabs: readonly GrabsGrab[] }) {
  const { nodes } = useListNodesSuspense().data;
  const nodeId = grabs.find(({ id }) => id === held.grabId)?.nodeId;
  const node = nodes.find(({ id }) => id === nodeId)?.name ?? "another Node";
  return (
    <>
      {held.step !== null && (
        <span className="text-meta text-ink-2">{stepWords[held.step](node)}</span>
      )}
      {held.until !== null && (
        <span className="text-meta text-ink-3">{delayWords(held.until)}</span>
      )}
    </>
  );
}

/**
 * What a cell's state adds under its chip: its file's quality, its download,
 * the step a Grab holding it is at and the file it replaces, or why it failed
 * and the file it kept.
 */
export function CellFacts({ cell, grabs }: { cell: FilesCell; grabs: readonly GrabsGrab[] }) {
  switch (cell.state) {
    case "met":
      return (
        <>
          <FileFacts file={cell.file} />
          {cell.held !== null && <HeldFacts held={cell.held} grabs={grabs} />}
        </>
      );
    case "upgradable":
      return <FileFacts file={cell.file} />;
    case "downloading": {
      const grab = grabs.find(({ id }) => id === cell.grabId);
      return (
        <>
          {grab !== undefined && <ProgressCell grab={grab} />}
          <HeldFacts held={cell} grabs={grabs} />
          {cell.file !== null && <Replaces file={cell.file} />}
        </>
      );
    }
    case "failed": {
      const grab = grabs.find(({ id }) => id === cell.grabId);
      return (
        <>
          {grab !== undefined && isFailed(grab) && (
            <span className="line-clamp-2 text-meta text-ink-3">
              {failureWords[grab.failure.code].cause}
            </span>
          )}
          {cell.file !== null && <FileFacts file={cell.file} />}
        </>
      );
    }
    case "missing":
    case "unreleased":
      return null;
    default:
      return cell satisfies never;
  }
}
