import type { GrabsGrabClientKind } from "./grabsGrabClientKind.ts";

export interface GrabsGrabClient {
  id: string;
  kind: GrabsGrabClientKind;
  name: string;
}
