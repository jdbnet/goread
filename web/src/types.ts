export type Book = {
  id: number;
  title: string;
  author: string;
  description: string;
  publisher: string;
  pub_date: string;
  isbn: string;
  language: string;
  cover_url: string;
  has_cover: boolean;
  total_chapters: number;
  file_missing: boolean;
  percent_completed: number;
  last_read_at: string | null;
  completed_at: string | null;
  total_time_read_seconds: number;
  current_cfi: string;
  series_id: number | null;
  series_name: string;
  sequence_number: number | null;
  created_at: string;
};

export type Series = {
  id: number;
  name: string;
  description: string;
  book_count: number;
  cover_urls?: string[];
  books?: Book[];
};

export type Progress = {
  book_id: number;
  current_cfi: string;
  percent_completed: number;
  total_time_read_seconds: number;
  last_read_at: string | null;
  completed_at: string | null;
};

export type AccentId =
  | "amber"
  | "orange"
  | "rose"
  | "red"
  | "emerald"
  | "teal"
  | "sky"
  | "indigo"
  | "violet"
  | "pink";

export type Settings = {
  font_size: number;
  line_height: number;
  theme: "light" | "dark" | "sepia";
  accent: AccentId | string;
};

export type AuthStatus = {
  enabled: boolean;
  authenticated: boolean;
  username: string;
  accent: string;
};

export type DailyStat = {
  day: string;
  time_read_seconds: number;
};

export type Stats = {
  total_time_read_seconds: number;
  completed_books: number;
  streak_days: number;
  this_week_seconds: number;
  daily: DailyStat[];
};

export type MetadataHit = {
  title: string;
  author: string;
  description: string;
  publisher: string;
  pub_date: string;
  isbn: string;
  language: string;
  cover_url: string;
};

export type BookListResponse = {
  books: Book[];
  total: number;
  page?: number;
  limit?: number;
};
