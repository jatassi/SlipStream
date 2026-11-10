import { ManualSearchHost } from "../manual/sheet";
import { AddSheet } from "../works/add/add-sheet";
import { Palette } from "./palette/palette";
import { Restart } from "./restart/restart-overlay";
import { SessionDialogs } from "./session-dialogs";

export function Overlays() {
  return (
    <>
      <Palette />
      <AddSheet />
      <ManualSearchHost />
      <Restart />
      <SessionDialogs />
    </>
  );
}
