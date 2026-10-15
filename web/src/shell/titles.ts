import { type AnyRouteMatch, useMatches } from "@tanstack/react-router";

/** A page's title that comes from its data: its loader returns `{ title }`. */
export const loaderTitle: unique symbol = Symbol("the loader's title");

declare module "@tanstack/react-router" {
  interface StaticDataRouteOption {
    title?: string | typeof loaderTitle;
  }
}

export type Crumb = { readonly title: string; readonly pathname: string };

type Titled = { title: string };

const isTitled = (data: unknown): data is Titled =>
  typeof data === "object" && data !== null && "title" in data && typeof data.title === "string";

function titleOf({ staticData, loaderData }: AnyRouteMatch): string | undefined {
  if (staticData.title !== loaderTitle) {
    return staticData.title;
  }
  return isTitled(loaderData) ? loaderData.title : undefined;
}

export const placeOf = (pathname: string): string =>
  pathname.length > 1 ? pathname.replace(/\/$/u, "") : pathname;

// An index route sits at its layout's place, so the two make one crumb and
// the deeper title wins.
export function breadcrumbs(matches: readonly AnyRouteMatch[]): Crumb[] {
  const crumbs: Crumb[] = [];
  for (const match of matches) {
    const title = titleOf(match);
    if (title === undefined) {
      continue;
    }
    const crumb = { title, pathname: placeOf(match.pathname) };
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
