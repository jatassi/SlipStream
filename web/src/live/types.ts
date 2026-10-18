import type { InvalidateQueryFilters, QueryClient } from "@tanstack/react-query";

import type { LiveEvent, LiveSessionEnded } from "../api/gen/model";

export type { LiveEvent } from "../api/gen/model";

export type LiveEventName = LiveEvent["event"];
export type ControlEventName = Extract<LiveEventName, `live.${string}`>;
export type ProductEventName = Exclude<LiveEventName, ControlEventName>;
export type PayloadOf<N extends LiveEventName> = Extract<LiveEvent, { event: N }>["data"];
export type LiveEffect<N extends ProductEventName> = (
  data: PayloadOf<N>,
  ctx: { queryClient: QueryClient; refresh: (filters?: InvalidateQueryFilters) => void },
) => void;
export type LiveEffects = { readonly [N in ProductEventName]: LiveEffect<N> };

export type LiveState = "idle" | "connecting" | "open" | "reconnecting" | "paused";
export type SessionEndReason = LiveSessionEnded["reason"];
