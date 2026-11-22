import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Data } from "./data";
import { Fact, Facts } from "./facts";

const noop = () => {};

function Overview({ rename }: { rename: () => void }) {
  return (
    <Facts look="sheet">
      <Fact label="Name" action={{ label: "Rename", onClick: rename }}>
        vps
      </Fact>
      <Fact label="Key">
        <Data>SHA256:9f2c41</Data>
      </Fact>
    </Facts>
  );
}

test("a list's terms and definitions read in order", async () => {
  renderKit(<Overview rename={noop} />);
  await expect.element(page.getByRole("term")).toHaveLength(2);
  expect(
    page
      .getByRole("term")
      .elements()
      .map((term) => term.textContent),
  ).toEqual(["Name", "Key"]);
  expect(
    page
      .getByRole("definition")
      .elements()
      .map((value) => value.textContent),
  ).toEqual(["vpsRename", "SHA256:9f2c41"]);
});

test("a fact's action is a button inside its definition", async () => {
  let renamed = 0;
  renderKit(
    <Overview
      rename={() => {
        renamed += 1;
      }}
    />,
  );
  const definition = page.getByRole("definition").filter({ hasText: "vps" });
  await userEvent.click(definition.getByRole("button", { name: "Rename" }));
  expect(renamed).toBe(1);
});
