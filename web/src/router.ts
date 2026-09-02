import { createRouter, createWebHistory } from "vue-router";
import HomeView from "./views/HomeView.vue";
import LibraryView from "./views/LibraryView.vue";
import SeriesView from "./views/SeriesView.vue";
import SeriesDetailView from "./views/SeriesDetailView.vue";
import StatsView from "./views/StatsView.vue";
import BookView from "./views/BookView.vue";
import ReaderView from "./views/ReaderView.vue";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "home", component: HomeView },
    { path: "/library", name: "library", component: LibraryView },
    { path: "/series", name: "series", component: SeriesView },
    { path: "/series/:id", name: "series-detail", component: SeriesDetailView, props: true },
    { path: "/stats", name: "stats", component: StatsView },
    { path: "/books/:id", name: "book", component: BookView, props: true },
    { path: "/read/:id", name: "read", component: ReaderView, props: true, meta: { hideChrome: true } },
  ],
  scrollBehavior() {
    return { top: 0 };
  },
});
