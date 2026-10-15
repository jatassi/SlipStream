import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { onTestFinished } from "vitest";

import { ShellContext, type ShellKind } from "../ui/shell-context";

export type Density = "console" | "touch";

export const densityOf: { readonly [S in ShellKind]: Density } = {
  desktop: "console",
  phone: "touch",
};

export function renderKit(
  ui: ReactNode,
  { shell = "desktop", density = densityOf[shell] }: { shell?: ShellKind; density?: Density } = {},
) {
  const container = document.createElement("div");
  container.dataset["density"] = density;
  document.body.append(container);
  const root = createRoot(container);
  root.render(<ShellContext value={shell}>{ui}</ShellContext>);
  onTestFinished(() => {
    root.unmount();
    container.remove();
  });
  return container;
}
