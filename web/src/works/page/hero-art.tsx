import { useMemo, useSyncExternalStore } from "react";

import type { WorksArt, WorksContentType, WorksWorkView } from "../../api/gen/model";
import { ambientScene, subscribeScene, useAmbient } from "../../shell/ambient/use-ambient";
import { useShellSlots } from "../../shell/page";
import { Artwork, sized } from "../../ui/artwork";

const neutral = { kind: "neutral" } as const;

/** The artwork route of a Work's image of a kind, without a size. */
export const artOf = (
  holder: { readonly artwork: readonly WorksArt[] },
  kind: string | null,
): string | undefined => holder.artwork.find((art) => art.kind === kind)?.url;

// The sharp backdrop shows once the ambient holds it, at the brightness
// the ambient measured, so text over its faded edge keeps the ceiling.
export function Backdrop({ route }: { route: string }) {
  const { backdrops } = useSyncExternalStore(subscribeScene, ambientScene);
  const url = sized(route, "large");
  const shown = backdrops.find((backdrop) => !backdrop.leaving && backdrop.url === url);
  if (shown === undefined) {
    return null;
  }
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[min(56vw,440px)] overflow-hidden [mask-image:linear-gradient(to_bottom,black_20%,transparent_92%),linear-gradient(to_right,transparent,black_12%,black_88%,transparent)] [mask-composite:intersect]"
    >
      <Artwork use="backdrop" size="backdrop" url={route} brightness={shown.brightness} />
    </div>
  );
}

/**
 * The title treatment as the h1's image, the label its accessible name; the label in text without one.
 * The image holds its full height before it loads, so the actions under it stay where a tap lands.
 */
export function TitleHeading({
  label,
  treatment,
}: {
  label: string;
  treatment: string | undefined;
}) {
  // A pushed page's nav bar holds its h1, so the hero's title shows without repeating it.
  const Title = useShellSlots().heading.kind === "bar" ? "p" : "h1";
  const hidden = Title === "p" ? { "aria-hidden": true } : {};
  return (
    <Title className="text-title-large text-ink-1" {...hidden}>
      <Artwork use="logo" size="title" url={treatment} label={label} />
    </Title>
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
          : { kind: "artwork", url: sized(backdrop, "large"), workId: work.id },
      [backdrop, failed, work.id],
    ),
  );
}
