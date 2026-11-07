import type { FilesCell, GrabsGrab } from "../../api/gen/model";
import { bytes } from "../../format";
import { ProgressCell } from "../../grabs/progress-cell";
import { failureWords, isFailed } from "../../grabs/stages";
import { useQualityNames } from "../../targets/quality-names";
import { Data } from "../../ui/data";

/** What a cell's state adds under its chip: its file's quality, its download, or why it failed. */
export function CellFacts({ cell, grab }: { cell: FilesCell; grab: GrabsGrab | undefined }) {
  const quality = useQualityNames();
  switch (cell.state) {
    case "met":
    case "upgradable":
      return (
        <span className="text-meta text-ink-3">
          <Data>
            {quality(cell.file.quality)} · {cell.file.probe.width}×{cell.file.probe.height} ·{" "}
            {bytes(cell.file.size)}
          </Data>
        </span>
      );
    case "downloading":
      return grab === undefined ? null : <ProgressCell grab={grab} />;
    case "failed":
      return grab !== undefined && isFailed(grab) ? (
        <span className="line-clamp-2 text-meta text-ink-3">
          {failureWords[grab.failure.code].cause}
        </span>
      ) : null;
    case "missing":
    case "unreleased":
      return null;
    default:
      return cell satisfies never;
  }
}
