import { useNow as useShared } from "../ui/clock";

// The Work page and Library read the shared clock through this until S46
// merges, when they call useNow from ui/clock and this file goes.
export const useNow: (everyMs: number) => Date = () => useShared();
