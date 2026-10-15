import { createContext, use } from "react";

export type ShellKind = "desktop" | "phone";

export const ShellContext = createContext<ShellKind>("desktop");

export const useShell = (): ShellKind => use(ShellContext);
