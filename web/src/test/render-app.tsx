import { createMemoryHistory } from "@tanstack/react-router";
import type { RequestHandler } from "msw";
import { createRoot } from "react-dom/client";
import { onTestFinished } from "vitest";

import { App, createApp } from "../app";
import { worker } from "./worker";

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
  onTestFinished(() => {
    root.unmount();
    container.remove();
    app.dispose();
  });
  return app;
}
