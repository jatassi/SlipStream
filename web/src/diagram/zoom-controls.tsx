import { FitToScreenIcon, ZoomInIcon, ZoomOutIcon } from "@hugeicons/core-free-icons";

import { IconButton } from "../ui/icon-button";
import { shortcut } from "../ui/kbd";
import { Tooltip } from "../ui/tooltip";
import type { Zoom } from "./use-view";

// The Diagram takes these keys while it has focus, and says so in aria-keyshortcuts.
const keys = { out: shortcut(["-"]), in: shortcut(["+"]), fit: shortcut(["0"]) } as const;

/** Zoom out, Zoom in and Fit, over the Diagram's top right corner, each naming its key. */
export function ZoomControls({ zoom }: { zoom: Zoom }) {
  return (
    <fieldset
      aria-label="Zoom"
      className="absolute top-3 right-3 flex items-center gap-2 rounded-capsule bg-popover px-2 py-1.5 shadow-float ring-1 ring-hairline backdrop-blur-[30px] backdrop-saturate-[1.7]"
    >
      <Tooltip label="Zoom out" shortcut={keys.out}>
        <IconButton
          icon={ZoomOutIcon}
          label="Zoom out"
          disabled={zoom.fitted}
          onClick={() => {
            zoom.step(-1);
          }}
        />
      </Tooltip>
      <Tooltip label="Zoom in" shortcut={keys.in}>
        <IconButton
          icon={ZoomInIcon}
          label="Zoom in"
          disabled={zoom.most}
          onClick={() => {
            zoom.step(1);
          }}
        />
      </Tooltip>
      <span aria-hidden="true" className="h-5 w-px bg-hairline" />
      <Tooltip label="Fit" shortcut={keys.fit}>
        <IconButton icon={FitToScreenIcon} label="Fit" disabled={zoom.fitted} onClick={zoom.fit} />
      </Tooltip>
    </fieldset>
  );
}
