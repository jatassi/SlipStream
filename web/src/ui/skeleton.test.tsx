import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Skeleton, type SkeletonProps } from "./skeleton";

const shapes: readonly SkeletonProps[] = [
  { label: "Loading the Jobs", shape: "rows", lines: 2 },
  { label: "Loading Torrents", shape: "table", lines: 2 },
  { label: "Loading the Library", shape: "grid", posters: "medium" },
  { label: "Loading the Node", shape: "facts" },
  { label: "Loading the Profile", shape: "form" },
  { label: "Loading the Work", shape: "hero" },
];

test.each(["desktop", "phone"] as const)(
  "in the %s shell each skeleton is a status named for what loads",
  async (shell) => {
    renderKit(
      <>
        {shapes.map((props) => (
          <Skeleton key={props.label} {...props} />
        ))}
      </>,
      { shell },
    );
    for (const { label } of shapes) {
      // oxlint-disable-next-line no-await-in-loop -- one status at a time names the one that fails
      await expect.element(page.getByRole("status", { name: label })).toBeVisible();
    }
  },
);
