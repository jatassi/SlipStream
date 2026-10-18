import { FitToScreenIcon, ZoomInIcon, ZoomOutIcon } from "@hugeicons/core-free-icons";

import { IconButton } from "../ui/icon-button";
import type { Zoom } from "./use-view";

/** Zoom out, Zoom in and Fit, over the Diagram's top right corner. */
export function ZoomControls({ zoom }: { zoom: Zoom }) {
  return (
    <fieldset
      aria-label="Zoom"
      className="absolute top-3 right-3 flex items-center gap-0.5 rounded-capsule bg-popover p-1 shadow-float ring-1 ring-hairline backdrop-blur-[30px] backdrop-saturate-[1.7]"
    >
      <IconButton
        icon={ZoomOutIcon}
        label="Zoom out"
        disabled={zoom.fitted}
        onClick={() => {
          zoom.step(-1);
        }}
      />
      <IconButton
        icon={ZoomInIcon}
        label="Zoom in"
        disabled={zoom.most}
        onClick={() => {
          zoom.step(1);
        }}
      />
      <IconButton icon={FitToScreenIcon} label="Fit" disabled={zoom.fitted} onClick={zoom.fit} />
    </fieldset>
  );
}
