import type { FilesCell, FilesLibraryFile, GrabsGrab } from "../../api/gen/model";
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

/**
 * What a cell's state adds under its chip: its file's quality, its download
 * and the file it replaces, or why it failed and the file it kept.
 */
export function CellFacts({ cell, grabs }: { cell: FilesCell; grabs: readonly GrabsGrab[] }) {
  switch (cell.state) {
    case "met":
    case "upgradable":
      return <FileFacts file={cell.file} />;
    case "downloading": {
      const grab = grabs.find(({ id }) => id === cell.grabId);
      return (
        <>
          {grab !== undefined && <ProgressCell grab={grab} />}
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
