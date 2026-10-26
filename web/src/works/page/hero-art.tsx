import { useMemo, useState, useSyncExternalStore } from "react";

import type { WorksContentType, WorksWorkView } from "../../api/gen/model";
import { ambientScene, subscribeScene, useAmbient } from "../../shell/ambient/use-ambient";

const neutral = { kind: "neutral" } as const;

/** The artwork route of the Work's image of a kind, without a size. */
export const artOf = (work: WorksWorkView, kind: string | null): string | undefined =>
  work.artwork.find((art) => art.kind === kind)?.url;

// The sharp backdrop shows once the ambient holds it, at the brightness
// the ambient measured, so text over its faded edge keeps the ceiling.
export function Backdrop({ url }: { url: string }) {
  const { backdrops } = useSyncExternalStore(subscribeScene, ambientScene);
  const shown = backdrops.find((backdrop) => !backdrop.leaving && backdrop.url === url);
  if (shown === undefined) {
    return null;
  }
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[min(56vw,440px)] overflow-hidden [mask-image:linear-gradient(to_bottom,black_20%,transparent_92%),linear-gradient(to_right,transparent,black_12%,black_88%,transparent)] [mask-composite:intersect]"
    >
      <img
        src={url}
        alt=""
        className="size-full object-cover"
        style={{ filter: `brightness(${shown.brightness})` }}
      />
    </div>
  );
}

export function PosterArt({ url }: { url: string | undefined }) {
  const [failed, setFailed] = useState<string>();
  return (
    <span
      aria-hidden="true"
      className="block aspect-2/3 w-28 shrink-0 overflow-hidden rounded-card bg-glass-raised ring-1 ring-hairline ring-inset sm:w-40"
    >
      {url !== undefined && failed !== url && (
        <img
          src={`${url}?size=medium`}
          alt=""
          className="size-full object-cover"
          onError={() => {
            setFailed(url);
          }}
        />
      )}
    </span>
  );
}

/** The title treatment as the heading's image, the label its accessible name; the label in text without one. */
export function TitleHeading({
  label,
  treatment,
}: {
  label: string;
  treatment: string | undefined;
}) {
  const [failed, setFailed] = useState<string>();
  if (treatment === undefined || failed === treatment) {
    return <h1 className="text-title-large text-ink-1">{label}</h1>;
  }
  return (
    <h1>
      <img
        src={`${treatment}?size=medium`}
        alt=""
        className="max-h-28 max-w-[min(420px,100%)] object-contain object-left"
        onError={() => {
          setFailed(treatment);
        }}
      />
      <span className="sr-only">{label}</span>
    </h1>
  );
}

/** The Work's backdrop as the ambient, neutral while it has none or its cells have failed. */
export function useWorkAmbient(type: WorksContentType, work: WorksWorkView, failed: boolean) {
  const backdrop = artOf(work, type.art.backdrop);
  useAmbient(
    useMemo(
      () =>
        backdrop === undefined || failed
          ? neutral
          : { kind: "artwork", url: `${backdrop}?size=large`, workId: work.id },
      [backdrop, failed, work.id],
    ),
  );
}
