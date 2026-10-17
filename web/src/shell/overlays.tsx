import { Palette } from "./palette/palette";
import { Restart } from "./restart/restart-overlay";
import { SessionDialogs } from "./session-dialogs";

export function Overlays() {
  return (
    <>
      <Palette />
      <Restart />
      <SessionDialogs />
    </>
  );
}
