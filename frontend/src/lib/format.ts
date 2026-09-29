export function truncateHash(value: string, lead = 6, tail = 6): string {
  if (value.length <= lead + tail + 1) return value;
  return `${value.slice(0, lead)}…${value.slice(-tail)}`;
}

export function formatScore(value: number, digits = 2): string {
  return value.toFixed(digits);
}

export function formatTimestamp(iso: string): string {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat("en-GB", {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(date);
}

export function formatRankShift(shift: number): { label: string; tone: "up" | "down" | "flat" } {
  if (shift > 0) return { label: `+${shift}`, tone: "up" };
  if (shift < 0) return { label: `${shift}`, tone: "down" };
  return { label: "0", tone: "flat" };
}

export function flagLabel(flag: string): string {
  return flag.replace(/_/g, " ");
}

export async function copyToClipboard(value: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(value);
    return true;
  } catch {
    return false;
  }
}
