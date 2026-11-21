import { describe, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Data } from "./data";
import { Fact, Facts } from "./facts";

const noop = () => {};

const boxOf = (element: Element) => element.getBoundingClientRect();

const pairOf = (label: string) => {
  const term = page.getByRole("term").filter({ hasText: label }).element();
  const definition = term.nextElementSibling;
  if (definition === null) {
    throw new Error(`${label} has no definition`);
  }
  return { term: boxOf(term), definition: boxOf(definition) };
};

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

test("in the desktop shell a label sits beside its value", async () => {
  renderKit(<Overview rename={noop} />);
  await expect.element(page.getByRole("term").first()).toBeVisible();
  const { term, definition } = pairOf("Key");
  expect(definition.top).toBe(term.top);
  expect(definition.left).toBeGreaterThan(term.right);
});

test("in the phone shell a label sits over its value", async () => {
  renderKit(<Overview rename={noop} />, { shell: "phone" });
  await expect.element(page.getByRole("term").first()).toBeVisible();
  const { term, definition } = pairOf("Key");
  expect(definition.top).toBeGreaterThanOrEqual(term.bottom);
});

test("labels line up down a list", async () => {
  renderKit(
    <Facts look="well">
      <Fact label="Route">Direct</Fact>
      <Fact label="Round trip">
        <Data>12 ms</Data>
      </Fact>
    </Facts>,
  );
  await expect.element(page.getByRole("definition").first()).toBeVisible();
  const lefts = page
    .getByRole("definition")
    .elements()
    .map((value) => boxOf(value).left);
  expect(new Set(lefts).size).toBe(1);
});

describe.each(["desktop", "phone"] as const)("in the %s shell", (shell) => {
  test("a fact's action is a button inside its definition, on the value's line", async () => {
    let renamed = 0;
    renderKit(
      <Overview
        rename={() => {
          renamed += 1;
        }}
      />,
      { shell },
    );
    const definition = page.getByRole("definition").filter({ hasText: "vps" });
    const button = definition.getByRole("button", { name: "Rename" });
    await userEvent.click(button);
    expect(renamed).toBe(1);
    const value = boxOf(definition.getByText("vps").element());
    const action = boxOf(button.element());
    expect(action.left).toBeGreaterThan(value.right);
    expect(action.top).toBeLessThan(value.bottom);
    expect(action.bottom).toBeGreaterThan(value.top);
  });
});

test("a string value is words and a Data value is Mono", async () => {
  renderKit(<Overview rename={noop} />);
  await expect.element(page.getByText("vps")).toBeVisible();
  const words = page.getByText("vps").element();
  const data = page.getByText("SHA256:9f2c41").element();
  expect(getComputedStyle(words).fontFamily).not.toMatch(/Mono/u);
  expect(getComputedStyle(data).fontFamily).toMatch(/Mono/u);
});

test("a failure list sets its labels in Label caps", async () => {
  renderKit(
    <Facts look="failure">
      <Fact label="Cause">The Indexer refused the download.</Fact>
    </Facts>,
  );
  await expect.element(page.getByRole("term")).toBeVisible();
  expect(getComputedStyle(page.getByRole("term").element()).textTransform).toBe("uppercase");
});

test("a well on a red row takes the row's dim ink for its labels", async () => {
  renderKit(
    <>
      <span className="text-fail-dim">probe</span>
      <Facts look="well" signal="fail">
        <Fact label="Free">12 GB</Fact>
      </Facts>
    </>,
  );
  await expect.element(page.getByRole("term")).toBeVisible();
  expect(getComputedStyle(page.getByRole("term").element()).color).toBe(
    getComputedStyle(page.getByText("probe").element()).color,
  );
});
