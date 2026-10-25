import { useState } from "react";

/**
 * A Work's poster at row size, 32 by 48, from an artwork route without a
 * size. With no artwork, or one the Node can't fetch, the slot stays a
 * raised placeholder, so every row keeps its shape.
 */
export function Poster({ url }: { url: string | undefined }) {
  const [failed, setFailed] = useState<string>();
  return (
    <span
      aria-hidden="true"
      data-poster={url === undefined ? "none" : "art"}
      className="block h-12 w-8 shrink-0 overflow-hidden rounded-item bg-glass-raised ring-1 ring-hairline ring-inset"
    >
      {url !== undefined && failed !== url && (
        <img
          src={`${url}?size=small`}
          alt=""
          loading="lazy"
          decoding="async"
          className="size-full object-cover"
          onError={() => {
            setFailed(url);
          }}
        />
      )}
    </span>
  );
}
