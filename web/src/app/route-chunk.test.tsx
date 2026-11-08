import {
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router";
import { HttpResponse, http } from "msw";
import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { UnreachableError } from "../api/errors";
import { RouteError } from "../shell/errors";
import { thisPage } from "../shell/reload";
import { renderKit } from "../test/render-kit";
import { worker } from "../test/worker";
import { PageDownloadError, lazyRouteComponent } from "./route-chunk";

const missing = () => new TypeError("Failed to fetch dynamically imported module: /assets/x.js");

const Page = () => <h1>The page</h1>;

function flakyImport() {
  const state = { failing: true, imports: 0 };
  const importer = async () => {
    state.imports += 1;
    await Promise.resolve();
    if (state.failing) {
      throw missing();
    }
    return { Page };
  };
  return { state, importer };
}

function renderPage(importer: () => Promise<{ Page: typeof Page }>) {
  const root = createRootRoute({
    component: () => (
      <main>
        <p>The shell</p>
        <Outlet />
      </main>
    ),
  });
  const pageRoute = createRoute({
    getParentRoute: () => root,
    path: "/",
    component: lazyRouteComponent(importer, "Page"),
  });
  const router = createRouter({
    routeTree: root.addChildren([pageRoute]),
    history: createMemoryHistory(),
    defaultErrorComponent: RouteError,
  });
  renderKit(<RouterProvider router={router} />);
}

const nodeDown = http.get("*/api/v1/system", () => HttpResponse.error());

test("a page whose code fails to download while the Node is down says so in the shell, and Try again opens it", async () => {
  let down = true;
  worker.use(http.get("*/api/v1/system", () => (down ? HttpResponse.error() : undefined)));
  const { state, importer } = flakyImport();
  renderPage(importer);

  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("SlipStream isn't responding. Check that it's running, then try again.");
  await expect.element(page.getByText("The shell")).toBeVisible();

  down = false;
  state.failing = false;
  await userEvent.click(page.getByRole("button", { name: "Try again" }));
  await expect.element(page.getByRole("heading", { name: "The page" })).toBeVisible();
  await expect.element(page.getByRole("alert")).not.toBeInTheDocument();
});

test("a page whose code fails to download while the Node responds offers a reload and doesn't reload by itself", async () => {
  const reload = vi.spyOn(thisPage, "reload").mockImplementation(() => {});
  renderPage(flakyImport().importer);

  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("This page didn't download. Reload to try again.");
  await expect.element(page.getByText("The shell")).toBeVisible();
  expect(reload).not.toHaveBeenCalled();

  await userEvent.click(page.getByRole("button", { name: "Reload" }));
  expect(reload).toHaveBeenCalledOnce();
});

test("a failed download rejects by whether the Node responds, and the next preload imports again", async () => {
  const { state, importer } = flakyImport();
  const component = lazyRouteComponent(importer, "Page");

  worker.use(nodeDown);
  await expect(component.preload()).rejects.toBeInstanceOf(UnreachableError);
  worker.resetHandlers();
  await expect(component.preload()).rejects.toBeInstanceOf(PageDownloadError);

  state.failing = false;
  await Promise.all([component.preload(), component.preload()]);
  expect(state.imports).toBe(3);
  await component.preload();
  expect(state.imports).toBe(3);
});
