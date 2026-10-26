import { useState } from "react";

const sizes = {
  row: { box: "h-12 w-8 rounded-item", art: "small" },
  option: { box: "h-9 w-6 rounded-[4px]", art: "small" },
  preview: { box: "h-45 w-30 rounded-card", art: "medium" },
} as const;

/**
 * A Work's poster from an artwork route without a size: 32 by 48 in a row,
 * 24 by 36 in a palette option and 120 by 180 in a preview. With no artwork,
 * or one the Node can't fetch, the slot stays a raised placeholder, so every
 * row keeps its shape.
 */
export function Poster({
  url,
  size = "row",
}: {
  url: string | undefined;
  size?: keyof typeof sizes;
}) {
  const [failed, setFailed] = useState<string>();
  return (
    <span
      aria-hidden="true"
      data-poster={url === undefined ? "none" : "art"}
      className={`block shrink-0 overflow-hidden bg-glass-raised ring-1 ring-hairline ring-inset ${sizes[size].box}`}
    >
      {url !== undefined && failed !== url && (
        <img
          src={`${url}?size=${sizes[size].art}`}
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
