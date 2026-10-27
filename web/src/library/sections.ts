/** A run of the Library under one heading, which its sort's value decides. */
export type Section = { readonly key: string; readonly label: string };

export type Comparable = string | number;

const named = (label: string): Section => ({ key: label, label });

/** One section per value, labelled by it, with the Works that have none under `none`. */
export const namedSections =
  (none: string) =>
  (value: Comparable | undefined): Section =>
    named(value === undefined ? none : String(value));

const latin = /^[A-Z]$/u;

/** By first letter, with digits and other scripts under "#". */
export function letterSection(value: Comparable | undefined): Section {
  const letter = String(value ?? "")
    .normalize("NFD")
    .charAt(0)
    .toUpperCase();
  return named(latin.test(letter) ? letter : "#");
}

const gib = 1024 ** 3;

/** By size on disk, with the Works that have no files last. */
export function bytesSection(value: Comparable | undefined): Section {
  if (typeof value !== "number") {
    return named("No files");
  }
  if (value >= 50 * gib) {
    return named("Over 50 GB");
  }
  return named(value >= 10 * gib ? "10 to 50 GB" : "Under 10 GB");
}

const hourMs = 3_600_000;

/** By whole hours of a length, such as "1h to 2h". */
export const durationSections =
  (none: string) =>
  (value: Comparable | undefined): Section => {
    if (typeof value !== "number") {
      return named(none);
    }
    const hours = Math.floor(value / hourMs);
    return named(hours === 0 ? "Under 1h" : `${hours}h to ${hours + 1}h`);
  };

/** By a rating out of 10. */
export function ratingSection(value: Comparable | undefined): Section {
  if (typeof value !== "number") {
    return named("No rating");
  }
  if (value >= 8) {
    return named("8 and up");
  }
  if (value >= 6) {
    return named("6 to 8");
  }
  return named(value >= 4 ? "4 to 6" : "Under 4");
}

const datePattern = /^(\d{4})(?:-(\d{2})(?:-(\d{2}))?)?/u;

type Parts = { readonly year: number; readonly month?: number; readonly day?: number };

function partsOf(value: Comparable | undefined): Parts | undefined {
  const match = typeof value === "string" ? datePattern.exec(value) : null;
  if (match === null) {
    return undefined;
  }
  const [, year, month, day] = match;
  return {
    year: Number(year),
    ...(month !== undefined && { month: Number(month) }),
    ...(day !== undefined && { day: Number(day) }),
  };
}

function thisYear({ year }: Parts, today: Date): string | undefined {
  if (year > today.getFullYear()) {
    return "Later";
  }
  return year === today.getFullYear() ? "This year" : undefined;
}

function aroundToday({ year, month, day }: Parts & { month: number }, today: Date): string {
  const months = year * 12 + month - (today.getFullYear() * 12 + today.getMonth() + 1);
  if (months > 1) {
    return "Later";
  }
  if (months === 1) {
    return "Next month";
  }
  if (months === 0) {
    return day !== undefined && day > today.getDate() ? "Later this month" : "This month";
  }
  if (months === -1) {
    return "Last month";
  }
  return year === today.getFullYear() ? "Earlier this year" : String(year);
}

/**
 * By where a date falls from today, ahead as well as behind, so a date to
 * come is never read as past: Later, Next month, Later this month, This
 * month, Last month, Earlier this year, then each year. A Work with no
 * date is under "No date".
 */
export const dateSections =
  (today: Date) =>
  (value: Comparable | undefined): Section => {
    const parts = partsOf(value);
    if (parts === undefined) {
      return named("No date");
    }
    const { month } = parts;
    return named(
      month === undefined
        ? (thisYear(parts, today) ?? String(parts.year))
        : aroundToday({ ...parts, month }, today),
    );
  };
