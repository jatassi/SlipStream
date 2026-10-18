import {
  type Flat,
  type LineTone,
  type Plan,
  type Plane,
  type Point3,
  type Projection,
  type Tone,
  cylinder,
  extrude,
  haloOf,
  iso,
  pathOf,
  ring as ringAt,
  roundedPlan,
  sideMatrix,
  slabOf,
} from "@anatomy/iso-kit";
import { type Faded, type Frame, disc as discOf, prismOf } from "@anatomy/lathe";
import { Dots, Line, Solid } from "@anatomy/react/draw";
import { Fragment, type ReactNode } from "react";

/**
 * The true shape of a drawn solid, in world units, for the anatomy audit.
 * The page never reads it.
 */
export type Shape = {
  readonly name: string;
  /** The solid this one is set into on purpose, so the audit excuses their overlap. */
  readonly owner?: string;
} & (
  | { readonly kind: "slab"; readonly plan: Plan; readonly z: number; readonly h: number; readonly steps: number }
  | { readonly kind: "extruded"; readonly ring: readonly Plane[]; readonly z: number; readonly h: number }
  | {
      readonly kind: "prism";
      readonly frame: Frame;
      readonly polygon: readonly Plane[];
      readonly s0: number;
      readonly s1: number;
    }
  | { readonly kind: "disc"; readonly frame: Frame; readonly s0: number; readonly s1: number; readonly r: number }
);

/** What one call draws: its markup, and the shapes of the solids in it. */
export type Drawn = { readonly el: ReactNode; readonly shapes: readonly Shape[] };

/** A pipe or cable drawn in an item: its centreline and radius. */
export type Pipe = { readonly name: string; readonly points: readonly Point3[]; readonly r: number };

/**
 * One painted item: a group of parts drawn in the order given, sorted
 * against other items as a whole.
 */
export type Item = {
  readonly name: string;
  readonly at: Point3;
  readonly el: ReactNode;
  readonly shapes: readonly Shape[];
  readonly pipes: readonly Pipe[];
};

type SolidStyle = {
  readonly tone?: Tone;
  readonly crease?: Tone | "none";
  readonly inner?: ReactNode;
  readonly lit?: boolean;
  readonly className?: string;
};

const styled = (paths: Parameters<typeof Solid>[0]["paths"], style: SolidStyle): ReactNode => (
  <Solid
    paths={paths}
    tone={style.tone ?? "mid"}
    crease={style.crease ?? "faint"}
    inner={style.inner}
    lit={style.lit ?? false}
    {...(style.className === undefined ? {} : { className: style.className })}
  />
);

/** A rounded box standing at z. */
export function box(
  P: Projection,
  name: string,
  plan: Plan,
  z: number,
  h: number,
  style: SolidStyle & { readonly steps?: number; readonly bevel?: number } = {},
): Drawn {
  const steps = style.steps ?? 3;
  return {
    el: styled(slabOf(plan, z, h, P, steps, style.bevel ?? 0.5), style),
    shapes: [{ kind: "slab", name, plan, z, h, steps }],
  };
}

/** A round post, puck or foot standing at z. */
export function post(
  P: Projection,
  name: string,
  [cx, cy]: Plane,
  r: number,
  z: number,
  h: number,
  style: SolidStyle & { readonly steps?: number; readonly bevel?: number } = {},
): Drawn {
  return {
    el: styled(cylinder(cx, cy, r, z, h, P, style.steps ?? 24, style.bevel ?? 0), style),
    shapes: [{ kind: "disc", name, frame: upright([cx, cy]), s0: z, s1: z + h, r }],
  };
}

const upright = ([x, y]: Plane): Frame => ({ o: [x, y, 0], a: [0, 0, 1], u: [1, 0, 0], v: [0, 1, 0] });

/** Any floor outline, extruded straight up from z. */
export function block(
  P: Projection,
  name: string,
  ring: readonly Plane[],
  z: number,
  h: number,
  style: SolidStyle = {},
): Drawn {
  return { el: styled(extrude(ring, z, h, P), style), shapes: [{ kind: "extruded", name, ring, z, h }] };
}

/** A polygon in the frame's u, v plane, extruded from s0 to s1 along its axis. */
export function prism(
  P: Projection,
  name: string,
  frame: Frame,
  polygon: readonly Plane[],
  s0: number,
  s1: number,
  style: SolidStyle = {},
): Drawn {
  return {
    el: styled(prismOf(frame, polygon, s0, s1, P), style),
    shapes: [{ kind: "prism", name, frame, polygon, s0, s1 }],
  };
}

/** A disc on any axis: a flange, a collar, a roller, a plug. */
export function disc(
  P: Projection,
  name: string,
  frame: Frame,
  s0: number,
  s1: number,
  r: number,
  style: SolidStyle = {},
): Drawn {
  return { el: styled(discOf(s0, s1, r, frame, P), style), shapes: [{ kind: "disc", name, frame, s0, s1, r }] };
}

export const line = (d: string, tone: LineTone = "lo", free?: true | "start" | "end"): Drawn => ({
  el: <Line d={d} tone={tone} {...(free === undefined ? {} : { free })} />,
  shapes: [],
});

export const dots = (
  points: readonly (Readonly<Flat> | readonly [number, number, number])[],
  size: number,
  tone: Tone = "mid",
): Drawn => ({ el: <Dots points={points} size={size} tone={tone} />, shapes: [] });

/** Marks what is drawn as set into `owner` on purpose: a band on a mast, a rail in a slot. */
export const owned = (drawn: Drawn, owner: string): Drawn => ({
  ...drawn,
  shapes: drawn.shapes.map((shape) => ({ ...shape, owner })),
});

/** Markup with no solid in it: a face's engraving, a clip, a live element. */
export const mark = (el: ReactNode): Drawn => ({ el, shapes: [] });

/** Several drawn things in one group, with the attributes the live loop finds them by. */
export function group(attributes: Readonly<Record<string, string>>, parts: readonly Drawn[]): Drawn {
  return {
    el: <g {...attributes}>{keyed(parts)}</g>,
    shapes: parts.flatMap(({ shapes }) => shapes),
  };
}

export const keyed = (parts: readonly Drawn[]): ReactNode =>
  parts.map(({ el }, index) => <Fragment key={index}>{el}</Fragment>);

export function item(name: string, at: Point3, parts: readonly Drawn[], pipes: readonly Pipe[] = []): Item {
  return {
    name,
    at,
    el: <g data-item={name}>{keyed(parts)}</g>,
    shapes: parts.flatMap(({ shapes }) => shapes),
    pipes,
  };
}

/** Screws: a ring and a dot at each point on a top face at z. */
export function screws(P: Projection, points: readonly Plane[], z: number): Drawn[] {
  return [
    line(points.map(([x, y]) => ringAt(x, y, 1.3, z, P, 12)).join(""), "lo"),
    dots(
      points.map(([x, y]) => iso([x, y, z], P)),
      0.42,
    ),
  ];
}

/** The soft shadow a plan casts on the floor. */
export const halo = (P: Projection, plan: Plan, z: number, h: number, blur: number): Drawn =>
  mark(<path className="iso-halo" d={haloOf(plan, z, h, P)} style={{ filter: `blur(${String(blur)}px)` }} />);

/** A rounded rectangle on a face, from (u0, z0) to (u1, z1), in a face's own coordinates. */
export const roundRect = (u0: number, z0: number, u1: number, z1: number, r: number, steps = 4): string =>
  pathOf(
    roundedPlan({ x: u0, y: z0, w: u1 - u0, d: z1 - z0, r }, steps).map(([u, z]): Flat => [u, -z]),
    true,
  );

/** The projection that draws local coordinates at `by` in the world. */
export const movedP = (P: Projection, by: Point3): Projection => ({ ...P, origin: iso(by, P) });

const add = ([x, y, z]: Point3, [dx, dy, dz]: Point3): Point3 => [x + dx, y + dy, z + dz];

/** A shape moved by `by` and scaled about the origin by `scale`, as the audit places it in the world. */
export function placedShape(shape: Shape, by: Point3, scale = 1): Shape {
  const [dx, dy, dz] = by;
  const at = (point: Point3): Point3 => add([point[0] * scale, point[1] * scale, point[2] * scale], by);
  const frame = (f: Frame): Frame => ({ ...f, o: at(f.o) });
  switch (shape.kind) {
    case "slab": {
      const { plan } = shape;
      return {
        ...shape,
        plan: { x: plan.x * scale + dx, y: plan.y * scale + dy, w: plan.w * scale, d: plan.d * scale, r: plan.r * scale },
        z: shape.z * scale + dz,
        h: shape.h * scale,
      };
    }
    case "extruded": {
      return {
        ...shape,
        ring: shape.ring.map(([x, y]): Plane => [x * scale + dx, y * scale + dy]),
        z: shape.z * scale + dz,
        h: shape.h * scale,
      };
    }
    case "prism": {
      return {
        ...shape,
        frame: frame(shape.frame),
        polygon: shape.polygon.map(([u, v]): Plane => [u * scale, v * scale]),
        s0: shape.s0 * scale,
        s1: shape.s1 * scale,
      };
    }
    case "disc": {
      return { ...shape, frame: frame(shape.frame), s0: shape.s0 * scale, s1: shape.s1 * scale, r: shape.r * scale };
    }
  }
}

export const placedPipe = (pipe: Pipe, by: Point3, scale = 1): Pipe => ({
  ...pipe,
  points: pipe.points.map((point) => add([point[0] * scale, point[1] * scale, point[2] * scale], by)),
  r: pipe.r * scale,
});

/** An item drawn in local coordinates, with its shapes and anchor moved by `by` into its parent's. */
export const placed = (it: Item, by: Point3): Item => ({
  ...it,
  at: add(it.at, by),
  shapes: it.shapes.map((shape) => placedShape(shape, by)),
  pipes: it.pipes.map((pipe) => placedPipe(pipe, by)),
});

/** Strokes that fade by their alpha, grouped so each alpha is one path. */
export function faded(list: readonly Faded[], tone: LineTone = "lo"): Drawn {
  const groups = new Map<number, string>();
  for (const { d, alpha } of list) {
    const level = Math.min(1, Math.round(alpha * 20) / 20);
    if (level > 0) {
      groups.set(level, (groups.get(level) ?? "") + d);
    }
  }
  return mark(
    <>
      {[...groups].map(([alpha, d]) => (
        <path key={alpha} className="iso-line" data-tone={tone} d={d} opacity={alpha < 1 ? alpha : undefined} />
      ))}
    </>,
  );
}

/** The transform onto the face at y, looking from +y: u runs along x, up is z. */
export const faceY = (P: Projection, y: number): string => sideMatrix([0, y], 0, P, "left");
/** The transform onto the face at x, looking from +x: u runs along -y, up is z. */
export const faceX = (P: Projection, x: number): string => sideMatrix([x, 0], 0, P, "right");
