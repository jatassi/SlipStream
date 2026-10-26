import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { asPhone } from "../test/viewport";
import { Select } from "./select";
import type { ShellKind } from "./shell-context";

const profiles = [
  { value: "any", label: "Any" },
  { value: "hd", label: "HD 1080p" },
  { value: "uhd", label: "Ultra HD" },
] as const;

function Profile() {
  const [value, setValue] = useState<(typeof profiles)[number]["value"]>("any");
  return <Select label="Quality" options={profiles} value={value} onValueChange={setValue} />;
}

test.each([
  ["desktop", "32px", "36px"],
  ["phone", "48px", "52px"],
] satisfies [ShellKind, string, string][])(
  "in the %s shell a %s select opens %s options, carrying the density it opened from, and picks one from the keyboard",
  async (shell, trigger, item) => {
    renderKit(<Profile />, { shell });
    const select = page.getByRole("combobox", { name: "Quality" });
    await expect.element(select).toHaveTextContent("Any");
    expect(getComputedStyle(select.element()).height).toBe(trigger);
    await userEvent.tab();
    await userEvent.keyboard("{ArrowDown}");
    const option = page.getByRole("option", { name: "HD 1080p" });
    await expect.element(option).toBeVisible();
    expect(getComputedStyle(option.element()).height).toBe(item);
    await expect.element(page.getByRole("option", { name: "Any" })).toHaveFocus();
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(option).toHaveFocus();
    await userEvent.keyboard("{Enter}");
    await expect.element(select).toHaveTextContent("HD 1080p");
    await expect.element(option).not.toBeInTheDocument();
  },
);

test("Esc closes the popup and leaves the value", async () => {
  renderKit(<Profile />);
  await page.getByRole("combobox", { name: "Quality" }).click();
  await expect.element(page.getByRole("listbox")).toBeVisible();
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByRole("listbox")).not.toBeInTheDocument();
  await expect.element(page.getByRole("combobox", { name: "Quality" })).toHaveTextContent("Any");
});

const dot = (word: string) => (
  <span data-testid={`dot-${word}`} className="size-1.5 rounded-capsule bg-ink-1">
    <span className="sr-only">{word}</span>
  </span>
);

const nodes = [
  { value: "home", label: "home", leading: dot("online") },
  { value: "seedbox", label: "seedbox", leading: dot("offline") },
] as const;

function Picker() {
  const [value, setValue] = useState<(typeof nodes)[number]["value"]>("home");
  return <Select label="Node" options={nodes} value={value} onValueChange={setValue} />;
}

test("an option's leading mark sits before its label in the list and on the trigger", async () => {
  renderKit(<Picker />);
  const select = page.getByRole("combobox", { name: "Node" });
  await expect.element(select).toHaveTextContent("onlinehome");
  await select.click();
  const seedbox = page.getByRole("option", { name: "offline seedbox" });
  await expect.element(seedbox).toBeVisible();
  await seedbox.click();
  await expect.element(select).toHaveTextContent("offlineseedbox");
  await expect.element(select.getByTestId("dot-offline")).toBeVisible();
});

const roots = [
  { value: "movies", label: "/srv/media/movies · 82 GB free" },
  {
    value: "archive",
    label: "/srv/media/archive/of/every/movie/this/library/has/ever/held · 82 GB free",
  },
] as const;

function Root() {
  const [value, setValue] = useState<(typeof roots)[number]["value"]>("movies");
  return <Select label="Library root" options={roots} value={value} onValueChange={setValue} />;
}

test("on a phone a list of options wider than the screen opens inside it", async () => {
  await asPhone();
  renderKit(<Root />, { shell: "phone" });
  await page.getByRole("combobox", { name: "Library root" }).click();
  const archive = page.getByRole("option", { name: /archive/u });
  await expect.element(archive).toBeVisible();
  const list = page.getByRole("listbox").element().getBoundingClientRect();
  expect(list.left, "the list starts on the screen").toBeGreaterThanOrEqual(0);
  expect(list.right, "the list ends on the screen").toBeLessThanOrEqual(innerWidth);
  expect(document.documentElement.scrollWidth, "the page doesn't scroll sideways").toBe(innerWidth);
  await archive.click();
  await expect
    .element(page.getByRole("combobox", { name: "Library root" }))
    .toHaveTextContent(roots[1].label);
});
