import type { GrabsGrab } from "../api/gen/model";

const stageText: Record<GrabsGrab["stage"], string> = {
  fetching: "Fetching",
  downloading: "Downloading",
  probing: "Probing",
  transferring: "Transferring",
  importing: "Importing",
  imported: "Imported",
  failed: "Failed",
};

const moving = new Set<GrabsGrab["stage"]>(["downloading", "transferring"]);

export function GrabStatus({ grab }: { grab: GrabsGrab }) {
  return (
    <span className="flex flex-wrap items-center gap-x-3">
      <span>{stageText[grab.stage]}</span>
      {moving.has(grab.stage) ? (
        <progress
          aria-label={`${stageText[grab.stage]} ${grab.releaseTitle}`}
          max={grab.progress?.total}
          value={grab.progress?.done}
          className="h-2 w-40"
        />
      ) : null}
      {grab.failure === undefined ? null : (
        <span className="text-fail-ink">{grab.failure.message}</span>
      )}
    </span>
  );
}
