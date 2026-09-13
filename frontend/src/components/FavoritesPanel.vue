<template>
  <div class="favorites-overlay" v-if="active" @click="closeHovers"></div>
  <div class="favorites-panel" v-if="active">
    <div class="favorites-panel__header">
      <i class="material-icons">star</i>
      <span>{{ t("sidebar.myFavorites") }}</span>
      <button class="action favorites-panel__close" @click="closeHovers">
        <i class="material-icons">close</i>
      </button>
    </div>

    <div class="favorites-panel__list">
      <p v-if="!favoritesStore.favorites.length" class="favorites-panel__empty">
        {{ t("files.noFavorites") }}
      </p>
      <div v-else class="favorites-panel__scroll">
        <div
          v-for="fav in favoritesStore.favorites"
          :key="fav.path"
          class="favorites-panel__item"
        >
          <button class="favorites-panel__item-btn" @click="openFav(fav.path)">
            <i class="material-icons">folder_open</i>
            <div class="favorites-panel__item-info">
              <span class="favorites-panel__item-name">{{ label(fav) }}</span>
              <span class="favorites-panel__item-path">{{ fav.realPath }}</span>
            </div>
          </button>
          <button
            class="action favorites-panel__item-star"
            :title="t('buttons.unfavorite')"
            @click.stop="favoritesStore.remove(fav.path)"
          >
            <i class="material-icons">star</i>
          </button>
        </div>
      </div>
    </div>

    <button
      v-if="canAddCurrentDir"
      class="favorites-panel__add"
      @click="addCurrentDir"
    >
      <i class="material-icons">add</i>
      {{ t("buttons.addCurrentFolder") }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useLayoutStore } from "@/stores/layout";
import { useFileStore } from "@/stores/file";
import { useFavoritesStore } from "@/stores/favorites";
import type { Favorite } from "@/api/favorites";

const { t } = useI18n();
const router = useRouter();
const layoutStore = useLayoutStore();
const fileStore = useFileStore();
const favoritesStore = useFavoritesStore();

const active = computed(() => layoutStore.currentPromptName === "favorites");

const canAddCurrentDir = computed(
  () => fileStore.isListing && fileStore.req?.isDir && fileStore.req?.path
);

const closeHovers = () => layoutStore.closeHovers();

const label = (fav: Favorite) => {
  const parts = fav.path.replace(/\/+$/, "").split("/");
  return parts[parts.length - 1] || "/";
};

const openFav = (path: string) => {
  router.push(`/files${path === "/" ? "/" : path}`);
  closeHovers();
};

const addCurrentDir = () => {
  if (!fileStore.req?.path) return;
  favoritesStore.add(fileStore.req.path);
};
</script>

<style scoped>
.favorites-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
  background: rgba(0, 0, 0, 0.3);
}

.favorites-panel {
  position: fixed;
  z-index: 10001;
  background: var(--surfacePrimary);
  border: 1px solid var(--borderPrimary);
  border-radius: 8px;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  max-height: 70vh;
  width: 20em;
  top: 5em;
  right: 8em;
}

@media (max-width: 736px) {
  .favorites-panel {
    top: auto;
    right: 0;
    bottom: 0;
    left: 0;
    width: 100%;
    max-height: 60vh;
    border-radius: 12px 12px 0 0;
  }
}

.favorites-panel__header {
  display: flex;
  align-items: center;
  gap: 0.5em;
  padding: 0.75em 1em;
  border-bottom: 1px solid var(--divider);
  color: var(--textPrimary);
  font-weight: 600;
}

.favorites-panel__header i {
  color: #f7b500;
}

.favorites-panel__close {
  margin-left: auto;
}

.favorites-panel__list {
  overflow-y: auto;
  flex: 1;
  min-height: 2em;
}

.favorites-panel__empty {
  padding: 1.5em 1em;
  text-align: center;
  color: var(--textSecondary);
}

.favorites-panel__scroll {
  display: flex;
  flex-direction: column;
}

.favorites-panel__item {
  display: flex;
  align-items: center;
}

.favorites-panel__item-btn {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.6em;
  padding: 0.65em 1em;
  color: var(--textPrimary);
  text-align: left;
  border: none;
  background: none;
  cursor: pointer;
  min-width: 0;
}

.favorites-panel__item-btn:hover {
  background: var(--hover);
}

.favorites-panel__item-btn i {
  flex-shrink: 0;
  color: var(--iconSecondary);
}

.favorites-panel__item-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.favorites-panel__item-name {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.favorites-panel__item-path {
  font-size: 0.85em;
  color: var(--textSecondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.favorites-panel__item-star {
  padding: 0.5em;
}

.favorites-panel__item-star i {
  color: #f7b500;
}

.favorites-panel__add {
  display: flex;
  align-items: center;
  gap: 0.5em;
  padding: 0.75em 1em;
  border-top: 1px solid var(--divider);
  background: none;
  color: var(--blue);
  font-weight: 500;
  cursor: pointer;
  border: none;
  border-radius: 0 0 8px 8px;
}

.favorites-panel__add:hover {
  background: var(--hover);
}

.favorites-panel__add i {
  font-size: 1.1em;
}
</style>
