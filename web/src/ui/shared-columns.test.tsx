import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Section } from "./section";
import { SharedColumns } from "./shared-columns";
import { type Column, Table } from "./table";

type Torrent = { id: string; name: string; rate: string; peers: number };

const columns: readonly [Column<Torrent>, ...Column<Torrent>[]] = [
  { id: "name", header: "Name", cell: (row) => row.name },
  { id: "rate", header: "Rate", cell: (row) => row.rate },
  { id: "peers", header: "Peers", cell: (row) => row.peers, hideBelow: 980 },
  { id: "action", header: "Action", cell: () => "Pause", align: "end" },
];

const short: Torrent = { id: "a", name: "Arrival", rate: "0 B/s", peers: 2 };
const long: Torrent = {
  id: "b",
  name: "A name long enough to need several lines once the first column has only the room the others leave it, and then some more words after that",
  rate: "4.2 megabytes every second",
  peers: 31,
};

function Tables({ width }: { width: number }) {
  return (
    <div style={{ width }}>
      <SharedColumns columns={columns}>
        <Section title="Short">
          <Table label="Short" columns={columns} rows={[short]} rowKey={(t) => t.id} />
        </Section>
        <Section title="Long">
          <Table label="Long" columns={columns} rows={[long]} rowKey={(t) => t.id} />
        </Section>
      </SharedColumns>
    </div>
  );
}

const edgesOf = (name: string) =>
  page
    .getByRole("table", { name })
    .getByRole("columnheader")
    .all()
    .map((header) => {
      const { left, width } = header.element().getBoundingClientRect();
      return { left, width };
    });

test("tables share the widths their widest content asks for", async () => {
  renderKit(<Tables width={1100} />);
  await expect.element(page.getByRole("table", { name: "Long" })).toBeVisible();
  expect(edgesOf("Short")).toEqual(edgesOf("Long"));
  const rate = page.getByRole("cell", { name: "4.2 megabytes every second" }).element();
  expect(rate.scrollWidth).toBeLessThanOrEqual(rate.clientWidth);
  expect(page.getByRole("cell", { name: "0 B/s" }).element().clientWidth).toBe(rate.clientWidth);
});

test("a row keeps its height until its content needs more", async () => {
  renderKit(<Tables width={700} />);
  const arrival = page.getByRole("cell", { name: "Arrival" });
  await expect.element(arrival).toBeVisible();
  expect(arrival.element().getBoundingClientRect().height).toBe(44);
  const tall = page.getByRole("cell", { name: long.name }).element();
  expect(tall.getBoundingClientRect().height).toBeGreaterThan(44);
  expect(tall.scrollHeight).toBeLessThanOrEqual(tall.clientHeight);
  const rate = page.getByRole("cell", { name: long.rate }).element();
  expect(rate.getBoundingClientRect().height).toBe(tall.getBoundingClientRect().height);
});

test("a column that hides still leaves the last column at the card's edge", async () => {
  renderKit(<Tables width={700} />);
  const table = page.getByRole("table", { name: "Short" });
  await expect.element(table).toBeVisible();
  expect(getComputedStyle(page.getByText("31").element()).display).toBe("none");
  expect(edgesOf("Short")).toEqual(edgesOf("Long"));
  const pause = table.getByRole("cell", { name: "Pause" }).element().getBoundingClientRect();
  expect(pause.right).toBe(table.element().getBoundingClientRect().right);
});
