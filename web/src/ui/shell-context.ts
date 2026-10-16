import { createContext, use } from "react";

export type ShellKind = "desktop" | "phone";

export const ShellContext = createContext<ShellKind>("desktop");

export const useShell = (): ShellKind => use(ShellContext);

export type Density = "console" | "touch";

// The root sets [data-density] on <html> in an effect, after children first
// render, so a component that must know its density in script reads it here.
export const DensityContext = createContext<Density>("console");

export const useDensity = (): Density => use(DensityContext);
