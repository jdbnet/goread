declare module "epubjs" {
  export interface Location {
    start?: { cfi?: string };
    end?: { cfi?: string };
  }

  export interface Rendition {
    display(target?: string): Promise<unknown>;
    next(): Promise<unknown>;
    prev(): Promise<unknown>;
    resize(width: number, height: number): void;
    on(event: string, fn: (location: Location) => void): void;
    themes: {
      default(obj: Record<string, Record<string, string>>): void;
      fontSize(size: string): void;
      override(name: string, value: string, important?: boolean): void;
    };
    destroy(): void;
  }

  export interface Book {
    ready: Promise<unknown>;
    renderTo(el: HTMLElement | string, options?: Record<string, unknown>): Rendition;
    locations: {
      generate(chars: number): Promise<unknown>;
      percentageFromCfi(cfi: string): number;
    };
    destroy(): void;
  }

  export default function ePub(url: string | ArrayBuffer, options?: { openAs?: string }): Book;
}
