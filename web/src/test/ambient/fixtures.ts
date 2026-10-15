import { http, HttpResponse } from "msw";

import black from "./black.png?inline";
import red from "./red.png?inline";
import twoTone from "./two-tone.png?inline";
import white from "./white.png?inline";

/** The sampler's fixture images as data URLs, written by fixtures.go. */
export const artwork = { white, black, red, "two-tone": twoTone } as const;

export type Artwork = keyof typeof artwork;

export const artworkUrl = (name: Artwork) => `/artwork/${name}.png`;

/** Serves each fixture at artworkUrl(name), as the Node will serve a Work's artwork. */
export const artworkHandlers = Object.entries(artwork).map(([name, dataUrl]) =>
  http.get(`/artwork/${name}.png`, () => {
    const bytes = Uint8Array.from(
      atob(dataUrl.slice(dataUrl.indexOf(",") + 1)),
      (c) => c.codePointAt(0) ?? 0,
    );
    return HttpResponse.arrayBuffer(bytes.buffer, { headers: { "Content-Type": "image/png" } });
  }),
);
