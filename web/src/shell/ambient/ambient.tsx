import "./ambient.css";

import { useEffect, useSyncExternalStore } from "react";

import { backdropFilter, type Tint, wash } from "./sampler";
import { ambientScene, subscribeScene } from "./use-ambient";

const percent = (fraction: number) => `${Math.round(fraction * 100)}%`;

const washBackground = [
  ...wash.glows.map(
    ({ radii, centre, lightness, alpha }) =>
      `radial-gradient(${percent(radii[0])} ${percent(radii[1])} at ${percent(centre[0])} ${percent(centre[1])}, hsl(var(--h) var(--s) ${lightness}% / ${alpha}), transparent ${percent(wash.fade)})`,
  ),
  `linear-gradient(hsl(var(--h) ${wash.darkening.saturation}% ${wash.darkening.top}% / 0), hsl(var(--h) ${wash.darkening.saturation}% ${wash.darkening.bottom}% / ${wash.darkening.alpha}))`,
].join(", ");

/** The fixed layer behind the app that a Work's artwork tints. The shell mounts it once. */
export function Ambient() {
  const { tint, backdrops } = useSyncExternalStore(subscribeScene, ambientScene);

  useEffect(() => {
    tintRoot(tint);
  }, [tint]);

  useEffect(
    () => () => {
      document.documentElement.style.removeProperty("--h");
      document.documentElement.style.removeProperty("--s");
    },
    [],
  );

  return (
    <div aria-hidden className="pointer-events-none fixed inset-0 -z-10 overflow-hidden">
      {backdrops.map((backdrop) => (
        <div
          key={backdrop.id}
          data-ambient-backdrop={backdrop.leaving ? "leaving" : "current"}
          className="absolute -inset-32 bg-cover bg-center"
          style={{
            backgroundImage: `url("${backdrop.url}")`,
            filter: `blur(${backdropFilter.blur}px) saturate(${backdropFilter.saturate}) brightness(${backdrop.brightness})`,
          }}
        />
      ))}
      <div className="absolute inset-0" style={{ background: washBackground }} />
    </div>
  );
}

// --h is a plain number, so it animates along the number line. Setting the
// hue nearest the current one turns the tint the short way round the wheel.
function tintRoot({ hue, saturation }: Tint) {
  const root = document.documentElement;
  const from = Number(getComputedStyle(root).getPropertyValue("--h"));
  root.style.setProperty("--h", String(hue + 360 * Math.round((from - hue) / 360)));
  root.style.setProperty("--s", `${saturation}%`);
}
