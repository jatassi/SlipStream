import {
  BookOpen01Icon,
  Film01Icon,
  HeadphonesIcon,
  Image01Icon,
  MusicNote01Icon,
  PanelsTopLeftIcon,
  ShapesIcon,
  Tv01Icon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";

import type { WorksContentTypeIcon } from "../api/gen/model";

export const contentTypeIcons = {
  film: Film01Icon,
  tv: Tv01Icon,
  note: MusicNote01Icon,
  "open-book": BookOpen01Icon,
  headphones: HeadphonesIcon,
  panels: PanelsTopLeftIcon,
  image: Image01Icon,
  shapes: ShapesIcon,
} as const satisfies Record<WorksContentTypeIcon, IconSvgElement>;
