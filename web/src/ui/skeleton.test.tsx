import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Fact, Facts } from "./facts";
import { List, ListRow } from "./list";
import type { ShellKind } from "./shell-context";
import { Skeleton, type SkeletonProps } from "./skeleton";

const shells = ["desktop", "phone"] satisfies ShellKind[];

const shapes: readonly SkeletonProps[] = [
  { label: "Loading the Jobs", shape: "rows", lines: 2 },
  { label: "Loading Torrents", shape: "table", lines: 2 },
  { label: "Loading the Library", shape: "grid" },
  { label: "Loading the Node", shape: "facts" },
  { label: "Loading the Profile", shape: "form" },
];

const heightOf = (element: Element) => element.getBoundingClientRect().height;

test.each(shells.flatMap((shell) => shapes.map((props) => [shell, props] as const)))(
  "in the %s shell a skeleton is a status named for what loads: %o",
  async (shell, props) => {
    renderKit(<Skeleton {...props} />, { shell });
    await expect.element(page.getByRole("status", { name: props.label })).toBeVisible();
  },
);

test.each(shells)(
  "in the %s shell a two-line rows skeleton is as tall as the List it stands in for",
  async (shell) => {
    renderKit(
      <>
        <Skeleton label="Loading the Jobs" shape="rows" lines={2} />
        <List label="Jobs">
          {["a", "b", "c", "d", "e", "f"].map((key) => (
            <ListRow key={key} title="Dune: Part Two" meta="Searching" />
          ))}
        </List>
      </>,
      { shell },
    );
    const skeleton = page.getByRole("status", { name: "Loading the Jobs" });
    await expect.element(skeleton).toBeVisible();
    expect(heightOf(skeleton.element())).toBe(
      heightOf(page.getByRole("list", { name: "Jobs" }).element()),
    );
  },
);

test.each(shells)(
  "in the %s shell a facts skeleton is as tall as the Facts it stands in for",
  async (shell) => {
    renderKit(
      <>
        <Skeleton label="Loading the Node" shape="facts" />
        <div data-testid="facts">
          <Facts look="sheet">
            {["Name", "Version", "Address", "Started"].map((label) => (
              <Fact key={label} label={label}>
                den
              </Fact>
            ))}
          </Facts>
        </div>
      </>,
      { shell },
    );
    const skeleton = page.getByRole("status", { name: "Loading the Node" });
    await expect.element(skeleton).toBeVisible();
    expect(heightOf(skeleton.element())).toBe(heightOf(page.getByTestId("facts").element()));
  },
);

test("on a phone a table skeleton draws as the rows the table turns into", async () => {
  renderKit(
    <>
      <Skeleton label="Loading Torrents" shape="table" lines={2} />
      <Skeleton label="Loading the Jobs" shape="rows" lines={2} />
    </>,
    { shell: "phone" },
  );
  const table = page.getByRole("status", { name: "Loading Torrents" });
  const rows = page.getByRole("status", { name: "Loading the Jobs" });
  await expect.element(table).toBeVisible();
  expect(heightOf(table.element())).toBe(heightOf(rows.element()));
});
