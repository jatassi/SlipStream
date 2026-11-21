import type { ImportsHint, WorksContentType } from "../api/gen/model";
import type { RootEntry } from "./root-entry";
import { bytes } from "../format";
import { Quality } from "../release/quality";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";
import { named, reasonOf } from "./words";

const providers = { tmdb: "TMDB", imdb: "IMDb", tvdb: "TVDB" } as const;

const hintWords = ({ title, year, ids }: ImportsHint) =>
  [named(title, year), ...ids.map(({ provider, value }) => `${providers[provider]} ${value}`)].join(
    " · ",
  );

/** A waiting folder's machinery: what its name reads as, and each file with its reading and reason. */
export function EntryDetails({ entry, type }: { entry: RootEntry; type: WorksContentType }) {
  return (
    <Details>
      <div className="mt-1">
        <Facts look="well">
          <Fact label="From the name">{hintWords(entry.hint)}</Fact>
          {entry.files.map((file) => (
            <Fact key={file.path} label={<Data>{file.path}</Data>}>
              <Data>
                {file.quality === null ? (
                  "No quality found"
                ) : (
                  <Quality quality={file.quality} look="word" />
                )}{" "}
                · {bytes(file.size)}
              </Data>{" "}
              {reasonOf(file.reason, type)}
              {file.reason.line !== undefined && (
                <span className="block">
                  <Data>{file.reason.line}</Data>
                </span>
              )}
            </Fact>
          ))}
        </Facts>
      </div>
    </Details>
  );
}
