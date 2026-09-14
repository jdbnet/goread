import { createApp } from "vue";
import App from "./App.vue";
import { router } from "./router";
import { prefetchCore, registerServiceWorker, startOffline } from "./offline/status";
import "./style.css";

async function boot(): Promise<void> {
  await startOffline();
  await registerServiceWorker();
  createApp(App).use(router).mount("#app");
  void prefetchCore();
}

void boot();
