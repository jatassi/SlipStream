import { type AnyRouteMatch, useMatches } from "@tanstack/react-router";

/** A page's title that comes from its data: its loader returns `{ title }`. */
export const loaderTitle: unique symbol = Symbol("the loader's title");

declare module "@tanstack/react-router" {
  interface StaticDataRouteOption {
    title?: string | typeof loaderTitle;
  }
}

export type Crumb = { readonly title: string; readonly pathname: string };

/**
 * A loader's title. `trail` names the pages above one that the route tree
 * doesn't nest it under, such as the Library a Work belongs to.
 */
export type Titled = { title: string; trail?: readonly Crumb[] };

const isTitled = (data: unknown): data is Titled =>
  typeof data === "object" && data !== null && "title" in data && typeof data.title === "string";

function crumbsOf(match: AnyRouteMatch): readonly Crumb[] {
  const pathname = placeOf(match.pathname);
  const { title } = match.staticData;
  if (title !== loaderTitle) {
    return title === undefined ? [] : [{ title, pathname }];
  }
  const data: unknown = match.loaderData;
  return isTitled(data) ? [...(data.trail ?? []), { title: data.title, pathname }] : [];
}

export const placeOf = (pathname: string): string =>
  pathname.length > 1 ? pathname.replace(/\/$/u, "") : pathname;

// An index route sits at its layout's place, so the two make one crumb and
// the deeper title wins.
export function breadcrumbs(matches: readonly AnyRouteMatch[]): Crumb[] {
  const crumbs: Crumb[] = [];
  for (const crumb of matches.flatMap((match) => crumbsOf(match))) {
    if (crumbs.at(-1)?.pathname === crumb.pathname) {
      crumbs[crumbs.length - 1] = crumb;
    } else {
      crumbs.push(crumb);
    }
  }
  return crumbs;
}

export const useBreadcrumbs = (): Crumb[] => breadcrumbs(useMatches());

export const useRouteTitle = (): string | undefined => useBreadcrumbs().at(-1)?.title;

export function documentTitle(matches: readonly AnyRouteMatch[]): string {
  const title = breadcrumbs(matches).at(-1)?.title;
  return title === undefined ? "SlipStream" : `${title} · SlipStream`;
}
