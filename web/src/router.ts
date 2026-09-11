import { createRouter, createWebHistory } from "vue-router";
import HomeView from "./views/HomeView.vue";
import LibraryView from "./views/LibraryView.vue";
import SeriesView from "./views/SeriesView.vue";
import SeriesDetailView from "./views/SeriesDetailView.vue";
import StatsView from "./views/StatsView.vue";
import SettingsView from "./views/SettingsView.vue";
import BookView from "./views/BookView.vue";
import ReaderView from "./views/ReaderView.vue";
import LoginView from "./views/LoginView.vue";
import { getAuthStatus, safeRedirect } from "./auth";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "home", component: HomeView },
    { path: "/library", name: "library", component: LibraryView },
    { path: "/series", name: "series", component: SeriesView },
    { path: "/series/:id", name: "series-detail", component: SeriesDetailView, props: true },
    { path: "/stats", name: "stats", component: StatsView },
    { path: "/settings", name: "settings", component: SettingsView },
    { path: "/books/:id", name: "book", component: BookView, props: true },
    { path: "/read/:id", name: "read", component: ReaderView, props: true, meta: { hideChrome: true } },
    { path: "/login", name: "login", component: LoginView, meta: { hideChrome: true } },
  ],
  scrollBehavior() {
    return { top: 0 };
  },
});

router.beforeEach(async (to) => {
  let status;
  try {
    status = await getAuthStatus();
  } catch {
    return true;
  }
  if (to.name === "login") {
    if (!status.enabled || status.authenticated) {
      return safeRedirect(to.query.redirect);
    }
    return true;
  }
  if (status.enabled && !status.authenticated) {
    return { name: "login", query: { redirect: to.fullPath } };
  }
  return true;
});
