import { createMemoryHistory } from "@tanstack/react-router";
import type { RequestHandler } from "msw";
import { type Root, createRoot } from "react-dom/client";
import { expect } from "vitest";

import { App, type AppInstance, createApp } from "../app";
import { worker } from "./worker";

type Mounted = { app: AppInstance; root: Root; container: HTMLElement };

const mounted = new Set<Mounted>();

export function renderApp({
  path = "/",
  handlers = [],
}: { path?: string; handlers?: RequestHandler[] } = {}) {
  worker.use(...handlers);
  const app = createApp(createMemoryHistory({ initialEntries: [path] }));
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  root.render(<App app={app} />);
  mounted.add({ app, root, container });
  return app;
}

const settled = ({ router, queryClient }: AppInstance) =>
  router.state.status === "idle" && queryClient.isFetching() === 0;

// An app torn down mid-load keeps loading, and its requests then reach the next
// test's handlers, so each app settles against its own test's server first.
export async function unmountApps() {
  const apps = [...mounted];
  mounted.clear();
  try {
    await Promise.all(
      apps.map(({ app }) =>
        expect.poll(() => settled(app), { message: "the app settles before teardown" }).toBe(true),
      ),
    );
  } finally {
    for (const { app, root, container } of apps) {
      root.unmount();
      container.remove();
      app.dispose();
    }
  }
}
