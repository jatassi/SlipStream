import { useRender } from "@base-ui/react/use-render";
import type { KeyboardEvent, ReactNode } from "react";

import { moveBetweenRows, rowGroup, rowTarget } from "./row-keys";
import "./rows.css";

export function List({ label, children }: { label: string; children: ReactNode }) {
  return (
    <ul
      aria-label={label}
      {...{ [rowGroup]: "" }}
      className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
    >
      {children}
    </ul>
  );
}

export function ListRow({
  title,
  meta,
  detail,
  leading,
  trailing,
  render,
}: {
  title: ReactNode;
  meta?: ReactNode;
  /** Lines below the meta line, which wrap instead of truncating. */
  detail?: ReactNode;
  leading?: ReactNode;
  trailing?: ReactNode;
  render?: useRender.RenderProp;
}) {
  const row = useRender({
    render,
    props: {
      ...(render && {
        [rowTarget]: "",
        onKeyDown: (event: KeyboardEvent<HTMLElement>) => {
          moveBetweenRows(event);
        },
      }),
      className: `flex w-full items-center gap-3 rounded-[inherit] px-4 py-2 text-left ${meta === undefined ? "min-h-(--row-height)" : "min-h-(--row-height-two)"} ${render ? "transition-colors duration-350 ease-standard hover:bg-(--row-wash) focus-visible:bg-(--row-wash)" : ""}`,
      children: (
        <>
          {leading}
          <span className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="truncate text-(length:--body-size) leading-(--body-line-height) text-ink-1 in-data-[density=touch]:text-card-title">
              {title}
            </span>
            {meta !== undefined && <span className="truncate text-meta text-ink-3">{meta}</span>}
            {detail}
          </span>
          {trailing !== undefined && <span className="shrink-0 text-ink-3">{trailing}</span>}
        </>
      ),
    },
  });
  return (
    <li className="relative first:rounded-t-card last:rounded-b-card not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:not-first:before:left-4">
      {row}
    </li>
  );
}
