import { type ReactNode, useState } from "react";

/** The sizes the artwork proxy serves an image at. */
export type ArtSize = "small" | "medium" | "large";

/** An artwork route at a size, the one place a `?size=` is written. */
export const sized = (route: string, size: ArtSize): string => `${route}?size=${size}`;

/** What a slot shows with no image: nothing, the title, the initials, or the label as text. */
type Placeholder = "nothing" | "title" | "initials" | "label";

type Fade = "artwork" | "ambient" | "none";

type Look = {
  readonly box: string;
  readonly image: string;
  readonly art: ArtSize;
  readonly loading: "lazy" | "eager";
  readonly placeholder: Placeholder;
  readonly fade: Fade;
};

const framed =
  "relative block shrink-0 overflow-hidden bg-glass-raised ring-1 ring-hairline ring-inset";

const cover = "absolute inset-0 size-full object-cover";

const looks = {
  option: {
    box: `${framed} h-9 w-6 rounded-[4px]`,
    image: cover,
    art: "small",
    loading: "lazy",
    placeholder: "nothing",
    fade: "artwork",
  },
  row: {
    box: `${framed} h-12 w-8 rounded-item`,
    image: cover,
    art: "small",
    loading: "lazy",
    placeholder: "nothing",
    fade: "artwork",
  },
  preview: {
    box: `${framed} h-45 w-30 rounded-card`,
    image: cover,
    art: "medium",
    loading: "eager",
    placeholder: "title",
    fade: "artwork",
  },
  hero: {
    box: `${framed} aspect-2/3 w-28 rounded-card sm:w-40`,
    image: cover,
    art: "medium",
    loading: "eager",
    placeholder: "title",
    fade: "artwork",
  },
  card: {
    box: `${framed} aspect-2/3 w-full rounded-card shadow-poster transition-opacity duration-(--duration-control)`,
    image: cover,
    art: "medium",
    loading: "lazy",
    placeholder: "title",
    fade: "artwork",
  },
  wide: {
    box: `${framed} aspect-video w-full rounded-item`,
    image: cover,
    art: "medium",
    loading: "eager",
    placeholder: "title",
    fade: "artwork",
  },
  tile: {
    box: `${framed} aspect-2/3 w-24 rounded-item`,
    image: cover,
    art: "medium",
    loading: "lazy",
    placeholder: "initials",
    fade: "artwork",
  },
  title: {
    box: "block",
    image: "block h-28 w-auto max-w-[min(420px,100%)] object-contain object-left-bottom",
    art: "medium",
    loading: "eager",
    placeholder: "label",
    fade: "artwork",
  },
  mark: {
    box: "flex shrink-0",
    image: "h-5 max-w-24 object-contain",
    art: "small",
    loading: "lazy",
    placeholder: "nothing",
    fade: "none",
  },
  backdrop: {
    box: "block size-full",
    image: "size-full object-cover",
    art: "large",
    loading: "eager",
    placeholder: "nothing",
    fade: "ambient",
  },
} as const satisfies Record<string, Look>;

const fades: { readonly [F in Fade]: string } = {
  artwork:
    "transition-opacity duration-(--duration-artwork) ease-standard motion-reduce:transition-none",
  ambient:
    "transition-opacity duration-(--duration-ambient) ease-standard motion-reduce:transition-none",
  none: "",
};

/** "Timothée Chalamet" → "TC", "Zendaya" → "Z": the first letters of the first and last words. */
export function initialsOf(name: string): string {
  const words = name
    .trim()
    .split(/\s+/u)
    .filter((word) => word !== "");
  const ends = words.length > 1 ? [words[0], words.at(-1)] : words;
  return ends
    .map((word) => Array.from(word ?? "")[0] ?? "")
    .join("")
    .toLocaleUpperCase();
}

// Each draws with or without an image in the slot. Initials sit behind a
// headshot so the photo fades in over them, and the title only fills a slot
// known to have no image, so a fast scroll doesn't flash a title on each card.
const placeholders: {
  readonly [P in Placeholder]: (label: string, imaged: boolean) => ReactNode;
} = {
  nothing: () => null,
  title: (label, imaged) =>
    imaged ? null : (
      <span className="absolute inset-0 flex items-center justify-center px-3 text-center">
        <span className="line-clamp-3 text-meta font-semibold text-balance text-ink-3">
          {label}
        </span>
      </span>
    ),
  initials: (label) => (
    <span className="absolute inset-0 flex items-center justify-center text-label text-ink-3">
      {initialsOf(label)}
    </span>
  ),
  label: (label, imaged) => (imaged ? <span className="sr-only">{label}</span> : label),
};

type Seen = { readonly url: string; readonly as: "faded" | "at-once" | "failed" };

/** One image's lifecycle: no image, waiting for it, shown with or without its fade, or failed. */
export type ArtPhase = "none" | "loading" | Seen["as"];

const phaseOf = (url: string | undefined, seen: Seen | undefined): ArtPhase => {
  if (url === undefined) {
    return "none";
  }
  return seen?.url === url ? seen.as : "loading";
};

export type ArtworkProps =
  | { use: "poster"; size: "option" | "row"; url: string | undefined }
  | { use: "poster"; size: "preview" | "hero"; url: string | undefined; label: string }
  | { use: "poster"; size: "card"; url: string | undefined; label: string; quiet: boolean }
  | { use: "still"; size: "wide"; url: string; label: string }
  | { use: "headshot"; size: "tile"; url: string | undefined; label: string }
  | { use: "logo"; size: "title"; url: string | undefined; label: string }
  | { use: "logo"; size: "mark"; url: string | undefined }
  | { use: "backdrop"; size: "backdrop"; url: string; brightness: number };

/**
 * An image from an artwork route without a size, in the slot its use and
 * size name. The slot holds its shape while the image loads, fades the image
 * in, shows an image the browser has already decoded at once, and keeps the
 * slot's placeholder when there is no image or it fails.
 */
export function Artwork(props: ArtworkProps) {
  const { url, size, use } = props;
  const look: Look = looks[size];
  const [seen, setSeen] = useState<Seen>();
  const phase = phaseOf(url, seen);
  const imaged = url !== undefined && phase !== "failed";
  const label = "label" in props ? props.label : "";
  const fade = phase === "at-once" ? "" : fades[look.fade];
  return (
    <span
      aria-hidden={look.placeholder === "label" ? undefined : true}
      data-art={use}
      data-art-state={phase}
      className={`${look.box} ${"quiet" in props && props.quiet ? "opacity-55" : ""}`}
    >
      {placeholders[look.placeholder](label, imaged)}
      {imaged && (
        <img
          key={url}
          ref={(image) => {
            if (phase === "loading" && image?.complete === true && image.naturalWidth > 0) {
              setSeen({ url, as: "at-once" });
            }
          }}
          src={sized(url, look.art)}
          alt=""
          loading={look.loading}
          decoding="async"
          className={`${look.image} ${phase === "loading" ? "opacity-0" : ""} ${fade}`}
          style={"brightness" in props ? { filter: `brightness(${props.brightness})` } : undefined}
          onLoad={() => {
            setSeen((prior) => (prior?.url === url ? prior : { url, as: "faded" }));
          }}
          onError={() => {
            setSeen({ url, as: "failed" });
          }}
        />
      )}
    </span>
  );
}

/** A poster's empty frame, for a skeleton standing in for a card or the hero. */
export function ArtworkFrame({ size }: { size: "card" | "hero" }) {
  return <span className={looks[size].box} />;
}
