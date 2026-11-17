import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";

import type { HistoryOneEntry } from "../api/gen/model";
import { getGetHistoryEntryQueryKey, getHistoryEntry } from "../api/gen/query/history/history";
import { isApiError, sentenceOf } from "../api/errors";
import { timeOfDay } from "../format";
import { stageWord } from "../grabs/stages";
import { itemAnchor } from "../health/levels";
import { Spinner } from "../ui/spinner";

const browserZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone;

const stageWords = new Map<string, string>(Object.entries(stageWord));

/** "library.disk" as the Health page's anchor for that item. */
function healthAnchor(key: string): string {
  const dot = key.indexOf(".");
  return itemAnchor(key.slice(0, dot), key.slice(dot + 1));
}

const heading = "mt-3 mb-1 text-meta font-semibold text-ink-2";

function Files({ files }: { files: HistoryOneEntry["files"] }) {
  return (
    <>
      <h4 className={heading}>Files</h4>
      <ul>
        {files.map((file) => (
          <li key={`${file.items.join()}:${file.quality}`}>
            {[...file.items, file.quality].join(" · ")}
            {file.replaced !== "" && (
              <span className="block text-ink-3">
                {file.replacedLabel === ""
                  ? `It replaced ${file.replaced}.`
                  : `It replaced the ${file.replaced} file for ${file.replacedLabel}.`}
              </span>
            )}
          </li>
        ))}
      </ul>
    </>
  );
}

function Links({ entry }: { entry: HistoryOneEntry["entry"] }) {
  return (
    <p className="mt-3 flex flex-wrap gap-x-4 gap-y-1">
      {entry.work !== null && (
        <Link
          to="/works/$workId"
          params={{ workId: entry.work.id }}
          className="font-semibold text-ink-1 underline"
        >
          Open {entry.work.label}
        </Link>
      )}
      {entry.links.health !== null && (
        <Link
          to="/health"
          hash={healthAnchor(entry.links.health)}
          className="font-semibold text-ink-1 underline"
        >
          Open on Health
        </Link>
      )}
    </p>
  );
}

function Parts({ one }: { one: HistoryOneEntry }) {
  const { entry, lines, files, stages, health } = one;
  return (
    <>
      {lines.map((line) => (
        <p key={line}>{line}</p>
      ))}
      {files.length > 0 && <Files files={files} />}
      {stages.length > 0 && (
        <>
          <h4 className={heading}>Stages</h4>
          <ul>
            {stages.map((stage) => (
              <li key={stage.stage}>
                {stageWords.get(stage.stage)} at {timeOfDay(stage.at)}
              </li>
            ))}
          </ul>
        </>
      )}
      {health.length > 0 && (
        <>
          <h4 className={heading}>Health</h4>
          <ul>
            {health.map((item) => (
              <li key={item.subject}>
                {item.subject}: {item.text}
              </li>
            ))}
          </ul>
        </>
      )}
      {(entry.work !== null || entry.links.health !== null) && <Links entry={entry} />}
    </>
  );
}

/** One entry in full, its times in the browser's zone, with a link to each page that still shows what it names. */
export function HistoryEntry({ id }: { id: number }) {
  const params = { zone: browserZone() };
  const one = useQuery({
    queryKey: getGetHistoryEntryQueryKey(id, params),
    queryFn: ({ signal }) => getHistoryEntry(id, params, { signal }),
  });
  return (
    <div className="flex flex-col gap-0.5 pt-1 pb-2 text-meta text-ink-2">
      {one.isPending ? (
        <Spinner />
      ) : one.isError ? (
        <p>
          {isApiError(one.error) && one.error.detail !== undefined ? `${one.error.title}. ` : ""}
          {sentenceOf(one.error)}
        </p>
      ) : (
        <Parts one={one.data} />
      )}
    </div>
  );
}
