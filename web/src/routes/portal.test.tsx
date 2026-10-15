import { expect, onTestFinished, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { signedInAsGrace } from "../test/handlers";
import { asPhone } from "../test/viewport";
import { renderApp } from "../test/render-app";

const mainNav = () => page.getByRole("navigation", { name: "Main" });

test("a Member lands in the portal, under its top nav, at the Touch density", async () => {
  const { router } = renderApp({ path: "/", handlers: [signedInAsGrace] });
  const portal = page.getByRole("navigation", { name: "Portal" });
  await expect
    .element(portal.getByRole("link", { name: "Home" }))
    .toHaveAttribute("aria-current", "page");
  expect(router.state.location.href).toBe("/portal");
  expect(
    portal
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Home", "Search", "Requests", "Account"]);
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect.element(mainNav()).not.toBeInTheDocument();
  expect(
    page
      .getByRole("link")
      .all()
      .filter((link) => link.element().getAttribute("aria-current") === "page")
      .map((link) => link.element().textContent),
  ).toEqual(["Home"]);
  expect(document.documentElement.dataset["density"]).toBe("touch");
  expect(document.title).toBe("Home · SlipStream");
});

test("the top nav turns to glass after 24px of scroll", async () => {
  renderApp({ path: "/portal", handlers: [signedInAsGrace] });
  const bar = page.getByRole("banner");
  await expect.element(bar).toBeVisible();
  const filter = () => getComputedStyle(bar.element()).backdropFilter;
  expect(filter()).toBe("none");

  document.body.style.minHeight = "3000px";
  onTestFinished(() => {
    document.body.style.minHeight = "";
  });
  window.scrollTo({ top: 40 });
  await expect.poll(filter).toBe("blur(24px) saturate(1.6)");
  window.scrollTo({ top: 0 });
  await expect.poll(filter).toBe("none");
});

test("the avatar menu opens Account and the page shows the Member's name", async () => {
  const { router } = renderApp({ path: "/portal", handlers: [signedInAsGrace] });
  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Account" }));
  await expect.poll(() => router.state.location.href).toBe("/portal/account");
  await expect.element(page.getByText("You're signed in as Grace Lindqvist.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Sign out" })).toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "Back to Library" }))
    .not.toBeInTheDocument();

  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await expect
    .poll(() =>
      page
        .getByRole("menuitem")
        .all()
        .map((item) => item.element().textContent),
    )
    .toEqual(["Account", "Sign out"]);
});

test("a Member who opens an Admin link is taken to the portal", async () => {
  const { router } = renderApp({ path: "/activity", handlers: [signedInAsGrace] });
  await expect.poll(() => router.state.location.href).toBe("/portal");
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect.element(mainNav()).not.toBeInTheDocument();
});

test("an Admin can open the portal to see what Members see", async () => {
  const { router } = renderApp({ path: "/portal/requests" });
  await expect.element(page.getByRole("heading", { name: "Requests", level: 1 })).toBeVisible();
  expect(router.state.location.href).toBe("/portal/requests");
  await expect
    .element(
      page.getByRole("navigation", { name: "Portal" }).getByRole("link", { name: "Requests" }),
    )
    .toHaveAttribute("aria-current", "page");

  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Back to Library" }));
  await expect.poll(() => router.state.location.href).toBe("/library");
  await expect.element(mainNav()).toBeVisible();
});

test("on a phone an Admin finds the way back to Library on the portal's Account page", async () => {
  await asPhone();
  const { router } = renderApp({ path: "/portal/account" });
  await userEvent.click(page.getByRole("button", { name: "Back to Library" }));
  await expect.poll(() => router.state.location.href).toBe("/library");
  await expect
    .element(page.getByRole("navigation", { name: "Tabs" }).getByRole("link", { name: "Library" }))
    .toHaveAttribute("aria-current", "page");
});

test("on a phone the portal has its own tab bar and no palette", async () => {
  await asPhone();
  const { router } = renderApp({ path: "/portal", handlers: [signedInAsGrace] });
  const tabs = page.getByRole("navigation", { name: "Tabs" });
  await expect
    .poll(() =>
      tabs
        .getByRole("listitem")
        .all()
        .map((tab) => tab.element().textContent),
    )
    .toEqual(["Home", "Search", "Requests", "Account"]);
  await expect
    .element(tabs.getByRole("link", { name: "Home" }))
    .toHaveAttribute("aria-current", "page");
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Search" })).not.toBeInTheDocument();
  await expect.element(mainNav()).not.toBeInTheDocument();

  await userEvent.click(tabs.getByRole("link", { name: "Search" }));
  await expect.poll(() => router.state.location.href).toBe("/portal/search");
  await expect
    .element(tabs.getByRole("link", { name: "Search" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(tabs.getByRole("link", { name: "Home" }))
    .not.toHaveAttribute("aria-current");
});
