import type { ImportsImportFile } from "../api/gen/model";
import { bytes } from "../format";
import { Quality } from "../release/quality";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";

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
  return (
    <Details>
      <div className="mt-1">
        <Facts look="well">
          <Fact label="File">
            <Data>{file.path}</Data>
          </Fact>
          <Fact label="Detected as">
            <Data>
              {file.quality === null ? (
                "No quality found"
              ) : (
                <Quality quality={file.quality} look="word" />
              )}{" "}
              · {bytes(file.size)}
            </Data>
          </Fact>
          {file.verdicts.map(({ targetId, sentence }) => (
            <Fact key={targetId} label={targetName(targetId)}>
              {sentence}
            </Fact>
          ))}
          {file.hold !== null && file.hold.attempts > 0 && (
            <Fact label="Tries">{attemptsWords(file.hold.attempts)}</Fact>
          )}
          {file.staged > 0 && (
            <Fact label="Staged">Its copy waiting here uses {bytes(file.staged)}.</Fact>
          )}
        </Facts>
      </div>
    </Details>
  );
}
