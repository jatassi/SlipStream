import type { FormatsPastedFormat } from "../api/gen/model";
import { Details } from "../ui/details";
import { RadioGroup } from "../ui/radio";
import { Segmented } from "../ui/segmented";
import type { Previewed } from "./paste-flow";
import { type App, appWords, outcomeWords } from "./paste-words";

const apps = [
  { value: "radarr", label: "Radarr" },
  { value: "sonarr", label: "Sonarr" },
] as const;

const clashChoices = [
  { value: "replace", label: "Replace" },
  { value: "keep", label: "Keep both" },
] as const;

function AppChoice({ app, onApp }: { app: App | undefined; onApp: (app: App) => void }) {
  return (
    <RadioGroup<App | "">
      label="Written for"
      options={apps}
      value={app ?? ""}
      onValueChange={(next) => {
        if (next !== "") {
          onApp(next);
        }
      }}
    />
  );
}

export function AppQuestion({ onApp }: { onApp: (app: App) => void }) {
  return (
    <div className="flex flex-col gap-1.5">
      <p className="text-ink-1">
        Say whether this was written for Radarr or Sonarr. Their values for a condition differ.
      </p>
      <AppChoice app={undefined} onApp={onApp} />
    </div>
  );
}

function Pasted({
  format,
  onReplacing,
}: {
  format: FormatsPastedFormat;
  onReplacing: (replacing: boolean) => void;
}) {
  return (
    <li className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-3">
      <span className="flex min-w-0 flex-col">
        <span className="font-semibold text-ink-1">{format.name}</span>
        <span className="text-meta text-ink-2 in-data-[density=touch]:text-body-touch">
          {outcomeWords(format)}
        </span>
      </span>
      {format.sameName !== undefined && (
        <Segmented
          label={`${format.name}: replace or keep both`}
          options={clashChoices}
          value={format.outcome === "replace" ? "replace" : "keep"}
          onValueChange={(choice) => {
            onReplacing(choice === "replace");
          }}
        />
      )}
    </li>
  );
}

export function PastePreview({
  shown,
  onApp,
  onReplace,
}: {
  shown: Previewed;
  onApp: (app: App) => void;
  onReplace: (replace: readonly string[]) => void;
}) {
  return (
    <>
      <Details summary={`Read as ${appWords[shown.app]} formats.`}>
        <AppChoice app={shown.app} onApp={onApp} />
      </Details>
      <ul
        aria-label="What the paste adds"
        className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        {shown.pasted.formats.map((format, at) => (
          <Pasted
            // oxlint-disable-next-line react/no-array-index-key -- a paste may hold two formats of one name, and its order never changes
            key={at}
            format={format}
            onReplacing={(replacing) => {
              const id = format.sameName?.id ?? "";
              onReplace(
                replacing ? [...shown.replace, id] : shown.replace.filter((other) => other !== id),
              );
            }}
          />
        ))}
      </ul>
    </>
  );
}
