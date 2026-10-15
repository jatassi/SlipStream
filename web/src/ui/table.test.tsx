import { describe, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { type Column, Table } from "./table";

type Torrent = { id: string; name: string; rate: string; peers: number };

const torrents: readonly Torrent[] = [
  { id: "a", name: "Dune: Part Two", rate: "4.2 MB/s", peers: 31 },
  { id: "b", name: "Arrival", rate: "0 B/s", peers: 2 },
];

const columns: readonly [Column<Torrent>, ...Column<Torrent>[]] = [
  { id: "name", header: "Name", cell: (row) => row.name },
  { id: "rate", header: "Rate", cell: (row) => row.rate, align: "end" },
  { id: "peers", header: "Peers", cell: (row) => row.peers, hideBelow: 980, align: "end" },
];

const shells = [
  { shell: "desktop", row: "44px" },
  { shell: "phone", row: "54px" },
] as const satisfies readonly { shell: ShellKind; row: string }[];

describe.each(shells)("in the $shell shell", ({ shell, row }) => {
  test("a table names its columns and rows", async () => {
    renderKit(<Table label="Torrents" columns={columns} rows={torrents} rowKey={(t) => t.id} />, {
      shell,
    });
    const table = page.getByRole("table", { name: "Torrents" });
    await expect.element(table.getByRole("columnheader", { name: "Name" })).toBeVisible();
    await expect.element(table.getByRole("columnheader", { name: "Rate" })).toBeVisible();
    await expect.element(table.getByRole("row")).toHaveLength(3);
    const cell = table.getByRole("cell", { name: "4.2 MB/s" });
    expect(getComputedStyle(cell.element()).height).toBe(row);
  });
});

test("arrow keys move the selection between rows", async () => {
  const picked: string[] = [];
  renderKit(
    <Table
      label="Torrents"
      columns={columns}
      rows={torrents}
      rowKey={(t) => t.id}
      selection={{
        selected: "a",
        onSelect: (t) => {
          picked.push(t.id);
        },
      }}
    />,
  );
  const dune = page.getByRole("button", { name: "Dune: Part Two" });
  await expect.element(dune).toHaveAttribute("aria-pressed", "true");
  await expect
    .element(page.getByRole("button", { name: "Arrival" }))
    .toHaveAttribute("aria-pressed", "false");
  dune.element().focus();
  await userEvent.keyboard("{ArrowDown}");
  await expect.element(page.getByRole("button", { name: "Arrival" })).toHaveFocus();
  expect(picked).toStrictEqual(["b"]);
});

test.each([
  { width: 1000, peers: "table-cell" },
  { width: 600, peers: "none" },
])("a $width px container shows the peers column as $peers", async ({ width, peers }) => {
  renderKit(
    <div style={{ width }}>
      <Table
        label="Torrents"
        columns={columns}
        rows={torrents}
        rowKey={(t) => t.id}
        stackBelow={500}
      />
    </div>,
  );
  await expect.element(page.getByRole("table", { name: "Torrents" })).toBeVisible();
  expect(getComputedStyle(page.getByText("31").element()).display).toBe(peers);
  expect(getComputedStyle(page.getByText("4.2 MB/s").element()).display).toBe("table-cell");
});

test("a narrow container stacks the rows", async () => {
  renderKit(
    <div style={{ width: 400 }}>
      <Table
        label="Torrents"
        columns={columns}
        rows={torrents}
        rowKey={(t) => t.id}
        stackBelow={500}
      />
    </div>,
  );
  const cell = page.getByText("4.2 MB/s");
  await expect.element(cell).toBeVisible();
  expect(getComputedStyle(cell.element()).display).toBe("block");
});
