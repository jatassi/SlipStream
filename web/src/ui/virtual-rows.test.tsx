import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { VirtualRows } from "./virtual-rows";

const lines = (from: number, count: number) =>
  Array.from({ length: count }, (_, index) => `line ${from + index}`);

function Log({ first }: { first: number }) {
  const [rows, setRows] = useState(() => lines(1, first));
  const [follow, setFollow] = useState(true);
  return (
    <div className="flex h-[300px] flex-col">
      <button
        type="button"
        onClick={() => {
          setRows((shown) => [...shown, ...lines(shown.length + 1, 5)]);
        }}
      >
        More
      </button>
      <output>{follow ? "Following" : "Not following"}</output>
      <VirtualRows
        label="Log"
        rows={rows}
        rowKey={(row) => row}
        estimate={30}
        follow={follow}
        onFollowChange={setFollow}
        renderRow={(row) => <div className="h-[30px]">{row}</div>}
      />
    </div>
  );
}

const shownRows = () =>
  page
    .getByRole("listitem")
    .all()
    .map((row) => row.element().textContent);

const scrollTo = (top: "start" | "end") => {
  const scroller = page.getByRole("list", { name: "Log" }).element().parentElement;
  if (scroller === null) {
    throw new Error("the list has no scroller");
  }
  scroller.scrollTop = top === "start" ? 0 : scroller.scrollHeight;
};

test("a long list draws only the rows near its view and starts at the tail", async () => {
  renderKit(<Log first={10_000} />);
  await expect.element(page.getByText("line 10000", { exact: true })).toBeVisible();
  expect(shownRows().length).toBeLessThan(40);
  expect(shownRows()).not.toContain("line 1");
});

test("the list follows new rows until scrolled up, and again once back at the tail", async () => {
  renderKit(<Log first={50} />);
  await expect.element(page.getByText("line 50", { exact: true })).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "More" }));
  await expect.element(page.getByText("line 55", { exact: true })).toBeVisible();

  scrollTo("start");
  await expect.element(page.getByRole("status")).toHaveTextContent("Not following");
  await userEvent.click(page.getByRole("button", { name: "More" }));
  await expect.element(page.getByText("line 1", { exact: true })).toBeVisible();
  expect(shownRows()).not.toContain("line 60");

  scrollTo("end");
  await expect.element(page.getByRole("status")).toHaveTextContent("Following");
  await userEvent.click(page.getByRole("button", { name: "More" }));
  await expect.element(page.getByText("line 65", { exact: true })).toBeVisible();
});
