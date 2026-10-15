import { useCallback, useState } from "react";

// A portaled popup leaves the [data-density] ancestor that scopes the density
// variables, so it carries the density of the place it was opened from.
export function usePortalDensity() {
  const [density, setDensity] = useState<string>();
  const probe = useCallback((element: HTMLElement | null) => {
    if (element) {
      setDensity(element.closest<HTMLElement>("[data-density]")?.dataset["density"]);
    }
  }, []);
  return { density, probe: <span hidden ref={probe} /> };
}
