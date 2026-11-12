import type { ImportsImportFile } from "../api/gen/model";
import { bytes } from "../format";
import { useQualityNames } from "../targets/quality-names";
import { Data } from "../ui/data";
import { Details } from "../ui/details";

const attemptsWords = (attempts: number) =>
  attempts === 1 ? "SlipStream tried once." : `SlipStream tried ${attempts} times.`;

/** A waiting file's machinery: its path, its reading, each Target's verdict, its tries and its staged bytes. */
export function FileDetails({
  file,
  targetName,
}: {
  file: ImportsImportFile;
  targetName: (id: string) => string;
}) {
  const quality = useQualityNames();
  return (
    <Details>
      <dl className="mt-1 grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1 rounded-item bg-well px-3 py-2">
        <dt>File</dt>
        <dd className="wrap-anywhere">
          <Data>{file.path}</Data>
        </dd>
        <dt>Reads as</dt>
        <dd>
          <Data>
            {file.quality === null ? "No quality read" : quality(file.quality)} · {bytes(file.size)}
          </Data>
        </dd>
        {file.verdicts.map(({ targetId, sentence }) => (
          <div key={targetId} className="contents">
            <dt>{targetName(targetId)}</dt>
            <dd>{sentence}</dd>
          </div>
        ))}
        {file.hold !== null && file.hold.attempts > 0 && (
          <>
            <dt>Tries</dt>
            <dd>{attemptsWords(file.hold.attempts)}</dd>
          </>
        )}
        {file.staged > 0 && (
          <>
            <dt>Staged</dt>
            <dd>Its copy waiting here uses {bytes(file.staged)}.</dd>
          </>
        )}
      </dl>
    </Details>
  );
}
