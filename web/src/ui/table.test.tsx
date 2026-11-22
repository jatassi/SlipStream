import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";

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

test("a table names its columns and rows", async () => {
  renderKit(<Table label="Torrents" columns={columns} rows={torrents} rowKey={(t) => t.id} />);
  const table = page.getByRole("table", { name: "Torrents" });
  await expect.element(table.getByRole("columnheader", { name: "Name" })).toBeVisible();
  await expect.element(table.getByRole("columnheader", { name: "Rate" })).toBeVisible();
  await expect.element(table.getByRole("row")).toHaveLength(3);
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

function renderTorrents(width: number) {
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
}

test("a container wider than a column's breakpoint shows that column", async () => {
  renderTorrents(1000);
  await expect.element(page.getByRole("columnheader", { name: "Peers" })).toBeVisible();
  await expect.element(page.getByText("31")).toBeVisible();
});

test("a container narrower than a column's breakpoint hides that column and keeps the rest", async () => {
  renderTorrents(600);
  await expect.element(page.getByText("4.2 MB/s")).toBeVisible();
  await expect.element(page.getByRole("columnheader", { name: "Rate" })).toBeVisible();
  await expect.element(page.getByText("Peers")).not.toBeVisible();
  await expect.element(page.getByText("31")).not.toBeVisible();
});

test("a narrow container stacks the rows under no column headers", async () => {
  renderTorrents(400);
  await expect.element(page.getByText("4.2 MB/s")).toBeVisible();
  await expect.element(page.getByText("Rate")).not.toBeVisible();
});

test("a stacked row leaves out a cell with nothing in it", async () => {
  const rated: readonly [Column<Torrent>, ...Column<Torrent>[]] = [
    { id: "name", header: "Name", cell: (row) => row.name },
    { id: "rate", header: "Rate", cell: (row) => (row.peers > 10 ? row.rate : null) },
  ];
  renderKit(
    <div style={{ width: 400 }}>
      <Table
        label="Torrents"
        columns={rated}
        rows={torrents}
        rowKey={(t) => t.id}
        stackBelow={500}
      />
    </div>,
  );
  await expect.element(page.getByText("4.2 MB/s")).toBeVisible();
  await expect.element(page.getByRole("row").nth(0).getByRole("cell")).toHaveLength(2);
  await expect.element(page.getByRole("row").nth(1).getByRole("cell")).toHaveLength(1);
});
