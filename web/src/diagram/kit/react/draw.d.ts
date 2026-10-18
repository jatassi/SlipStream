import type { JSX, ReactNode } from "react";

import type { Flat, LegendKey, LineTone, SolidPaths, Tone } from "../iso-kit";

export function Solid(props: {
  readonly paths: SolidPaths;
  readonly tone?: Tone;
  readonly crease?: Tone | "none";
  readonly halo?: boolean;
  readonly lit?: boolean;
  readonly flat?: boolean;
  readonly className?: string;
  readonly inner?: ReactNode;
  readonly children?: ReactNode;
}): JSX.Element;

export function Line(props: {
  readonly d: string;
  readonly tone?: LineTone;
  readonly dotted?: boolean;
  readonly dashed?: boolean;
  readonly flow?: boolean;
  readonly className?: string;
  readonly free?: boolean | "start" | "end";
}): JSX.Element | null;

export function Wire(props: {
  readonly from: Readonly<Flat>;
  readonly to: Readonly<Flat>;
  readonly via?: readonly Readonly<Flat>[];
  readonly tone?: LineTone;
  readonly dotted?: boolean;
  readonly dashed?: boolean;
  readonly flow?: boolean;
  readonly ends?: "both" | "from" | "to" | "none";
  readonly size?: number;
}): JSX.Element;

export function Dots(props: {
  readonly points: readonly (Readonly<Flat> | readonly [number, number, number])[];
  readonly size?: number;
  readonly tone?: Tone;
  readonly pulse?: boolean;
}): JSX.Element;

export function FaceText(props: {
  readonly transform: string;
  readonly children: ReactNode;
  readonly size?: number;
  readonly tone?: Tone;
  readonly anchor?: "start" | "middle" | "end";
}): JSX.Element;

export function IsoFigure(props: {
  readonly width: number;
  readonly height: number;
  readonly label: string;
  readonly className?: string;
  readonly children: ReactNode;
}): JSX.Element;

export function Plate(props: {
  readonly fig: string;
  readonly title?: string;
  readonly hint?: ReactNode;
  readonly readout?: ReactNode;
  readonly keys?: readonly LegendKey[];
  readonly caption?: ReactNode;
  readonly children: ReactNode;
}): JSX.Element;
