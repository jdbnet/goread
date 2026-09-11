import { getAuthStatus } from "./auth";
import type { AccentId } from "./types";

export type AccentPalette = {
  id: AccentId;
  label: string;
  accent: string;
  strong: string;
  soft: string;
  bar: string;
  barDark: string;
  muted: string;
  mutedFg: string;
  mutedDark: string;
  mutedFgDark: string;
};

export const ACCENTS: AccentPalette[] = [
  {
    id: "amber",
    label: "Amber",
    accent: "#b45309",
    strong: "#92400e",
    soft: "#fbbf24",
    bar: "#d97706",
    barDark: "#f59e0b",
    muted: "#fef3c7",
    mutedFg: "#92400e",
    mutedDark: "#451a03",
    mutedFgDark: "#fcd34d",
  },
  {
    id: "orange",
    label: "Orange",
    accent: "#c2410c",
    strong: "#9a3412",
    soft: "#fb923c",
    bar: "#ea580c",
    barDark: "#f97316",
    muted: "#ffedd5",
    mutedFg: "#9a3412",
    mutedDark: "#431407",
    mutedFgDark: "#fdba74",
  },
  {
    id: "rose",
    label: "Rose",
    accent: "#be123c",
    strong: "#9f1239",
    soft: "#fb7185",
    bar: "#e11d48",
    barDark: "#f43f5e",
    muted: "#ffe4e6",
    mutedFg: "#9f1239",
    mutedDark: "#4c0519",
    mutedFgDark: "#fda4af",
  },
  {
    id: "red",
    label: "Red",
    accent: "#b91c1c",
    strong: "#991b1b",
    soft: "#f87171",
    bar: "#dc2626",
    barDark: "#ef4444",
    muted: "#fee2e2",
    mutedFg: "#991b1b",
    mutedDark: "#450a0a",
    mutedFgDark: "#fca5a5",
  },
  {
    id: "emerald",
    label: "Emerald",
    accent: "#047857",
    strong: "#065f46",
    soft: "#34d399",
    bar: "#059669",
    barDark: "#10b981",
    muted: "#d1fae5",
    mutedFg: "#065f46",
    mutedDark: "#022c22",
    mutedFgDark: "#6ee7b7",
  },
  {
    id: "teal",
    label: "Teal",
    accent: "#0f766e",
    strong: "#115e59",
    soft: "#2dd4bf",
    bar: "#0d9488",
    barDark: "#14b8a6",
    muted: "#ccfbf1",
    mutedFg: "#115e59",
    mutedDark: "#042f2e",
    mutedFgDark: "#5eead4",
  },
  {
    id: "sky",
    label: "Sky",
    accent: "#0369a1",
    strong: "#075985",
    soft: "#38bdf8",
    bar: "#0284c7",
    barDark: "#0ea5e9",
    muted: "#e0f2fe",
    mutedFg: "#075985",
    mutedDark: "#082f49",
    mutedFgDark: "#7dd3fc",
  },
  {
    id: "indigo",
    label: "Indigo",
    accent: "#4338ca",
    strong: "#3730a3",
    soft: "#818cf8",
    bar: "#4f46e5",
    barDark: "#6366f1",
    muted: "#e0e7ff",
    mutedFg: "#3730a3",
    mutedDark: "#1e1b4b",
    mutedFgDark: "#a5b4fc",
  },
  {
    id: "violet",
    label: "Violet",
    accent: "#6d28d9",
    strong: "#5b21b6",
    soft: "#a78bfa",
    bar: "#7c3aed",
    barDark: "#8b5cf6",
    muted: "#ede9fe",
    mutedFg: "#5b21b6",
    mutedDark: "#2e1065",
    mutedFgDark: "#c4b5fd",
  },
  {
    id: "pink",
    label: "Pink",
    accent: "#be185d",
    strong: "#9d174d",
    soft: "#f472b6",
    bar: "#db2777",
    barDark: "#ec4899",
    muted: "#fce7f3",
    mutedFg: "#9d174d",
    mutedDark: "#500724",
    mutedFgDark: "#f9a8d4",
  },
];

const DEFAULT_ACCENT: AccentId = "emerald";

export function applyAccent(id: string): void {
  const p = ACCENTS.find((a) => a.id === id) ?? ACCENTS.find((a) => a.id === DEFAULT_ACCENT)!;
  const root = document.documentElement;
  root.style.setProperty("--accent", p.accent);
  root.style.setProperty("--accent-strong", p.strong);
  root.style.setProperty("--accent-soft", p.soft);
  root.style.setProperty("--accent-bar", p.bar);
  root.style.setProperty("--accent-bar-dark", p.barDark);
  root.style.setProperty("--accent-muted", p.muted);
  root.style.setProperty("--accent-muted-fg", p.mutedFg);
  root.style.setProperty("--accent-muted-dark", p.mutedDark);
  root.style.setProperty("--accent-muted-fg-dark", p.mutedFgDark);
  root.dataset.accent = p.id;
}

export async function loadAccent(): Promise<void> {
  try {
    const status = await getAuthStatus();
    applyAccent(status.accent || DEFAULT_ACCENT);
  } catch {
    applyAccent(DEFAULT_ACCENT);
  }
}
