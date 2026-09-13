<template>
  <button
    type="button"
    class="action theme-toggle"
    :aria-label="t('buttons.toggleTheme')"
    :title="t('buttons.toggleTheme')"
    @click="toggle"
  >
    <svg
      v-if="isDark"
      class="theme-toggle__icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="4" fill="currentColor" stroke="none" />
      <path
        d="M12 2v2m0 16v2M4.93 4.93l1.41 1.41m11.32 11.32l1.41 1.41M2 12h2m16 0h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"
      />
    </svg>
    <svg
      v-else
      class="theme-toggle__icon"
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M21 12.79A9 9 0 1 1 11.21 3a7 7 0 0 0 9.79 9.79z" />
    </svg>
  </button>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const THEME_KEY = "fan-files-theme";
type Theme = "dark" | "light";

const isDark = ref<boolean>(currentTheme() === "dark");

function getStoredTheme(): Theme | null {
  try {
    const value = localStorage.getItem(THEME_KEY);
    return value === "light" || value === "dark" ? value : null;
  } catch {
    return null;
  }
}

function getSystemTheme(): Theme {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function currentTheme(): Theme {
  const active = document.documentElement.dataset.theme;
  return active === "light" || active === "dark"
    ? active
    : getStoredTheme() || getSystemTheme();
}

function persist(theme: Theme) {
  try {
    localStorage.setItem(THEME_KEY, theme);
  } catch {
    /* storage may be unavailable */
  }
}

function applyTheme(theme: Theme) {
  const root = document.documentElement;
  root.dataset.theme = theme;
  if (theme === "dark") {
    root.classList.add("dark");
    root.classList.remove("light");
  } else {
    root.classList.add("light");
    root.classList.remove("dark");
  }
  persist(theme);
  syncMeta(theme);
  isDark.value = theme === "dark";
}

function syncMeta(theme: Theme) {
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) {
    meta.setAttribute("content", theme === "dark" ? "#141d24" : "#fafafa");
  }
}

function toggle() {
  const next: Theme = currentTheme() === "dark" ? "light" : "dark";
  const root = document.documentElement;

  // Temporary class that crossfades themed colors (WOW.js / Sal.js
  // entrance animations use opacity/transform and are left untouched).
  root.classList.add("theme-transition");
  applyTheme(next);
  window.setTimeout(() => {
    root.classList.remove("theme-transition");
  }, 350);
}

// Keep data-theme in sync when the theme is changed from the settings
// page (which only toggles the .dark/.light class on the root element).
let observer: MutationObserver | null = null;
onMounted(() => {
  observer = new MutationObserver(() => {
    const root = document.documentElement;
    if (root.classList.contains("dark")) {
      syncFromClass("dark");
    } else if (root.classList.contains("light")) {
      syncFromClass("light");
    }
  });
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["class"],
  });
});

function syncFromClass(theme: Theme) {
  const root = document.documentElement;
  if (root.dataset.theme === theme) return;
  root.dataset.theme = theme;
  persist(theme);
  syncMeta(theme);
  isDark.value = theme === "dark";
}

onBeforeUnmount(() => {
  observer?.disconnect();
});
</script>

<style scoped>
.theme-toggle {
  width: 44px;
  min-width: 44px;
  height: 44px;
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  padding: 0;
  color: var(--action);
  transition:
    background-color 0.25s ease,
    color 0.25s ease;
}

.theme-toggle__icon {
  width: 24px;
  height: 24px;
  display: block;
  transition: transform 0.25s ease;
}

.theme-toggle:focus-visible {
  outline: 2px solid var(--blue);
  outline-offset: 2px;
}

@media (hover: hover) {
  .theme-toggle:hover .theme-toggle__icon {
    transform: rotate(20deg) scale(1.1);
  }
}

@media (hover: none) {
  .theme-toggle:hover .theme-toggle__icon,
  .theme-toggle:active .theme-toggle__icon {
    transform: none;
  }
}
</style>
