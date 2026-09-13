<template>
  <div>
    <header-bar showMenu showLogo />
    <div class="favorites-view">
      <div class="card">
        <div class="card-title">
          <h2>
            <i class="material-icons favorites-view__title-icon">star</i>
            {{ t("sidebar.myFavorites") }}
          </h2>
        </div>
        <div class="card-content">
          <p
            v-if="!favoritesStore.favorites.length"
            class="favorites-view__empty"
          >
            {{ t("files.noFavorites") }}
          </p>
          <div v-else class="favorites-view__list">
            <div
              v-for="fav in favoritesStore.favorites"
              :key="fav.path"
              class="favorites-view__item"
            >
              <button class="favorites-view__open" @click="openFav(fav.path)">
                <i class="material-icons">folder_open</i>
                <div class="favorites-view__info">
                  <span class="favorites-view__name">{{ label(fav) }}</span>
                  <span class="favorites-view__path">{{ fav.realPath }}</span>
                </div>
              </button>
              <button
                class="action favorites-view__remove"
                :title="t('buttons.unfavorite')"
                @click.stop="toggleFav(fav.path)"
              >
                <i class="material-icons">star</i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useFavoritesStore } from "@/stores/favorites";
import type { Favorite } from "@/api/favorites";

import HeaderBar from "@/components/header/HeaderBar.vue";

const { t } = useI18n();
const router = useRouter();
const favoritesStore = useFavoritesStore();

onMounted(() => {
  favoritesStore.init();
});

const label = (fav: Favorite) => {
  const parts = fav.path.replace(/\/+$/, "").split("/");
  return parts[parts.length - 1] || "/";
};

const openFav = (path: string) => {
  router.push(`/files${path === "/" ? "/" : path}`);
};

const toggleFav = async (path: string) => {
  await favoritesStore.toggle(path);
};
</script>

<style scoped>
.favorites-view {
  padding: 1.5em;
  max-width: 48em;
  margin: 0 auto;
}

.favorites-view__title-icon {
  vertical-align: -0.3em;
  color: #f7b500;
}

.favorites-view__empty {
  margin: 0;
  padding: 1em 0;
  color: var(--textSecondary);
}

.favorites-view__list {
  display: flex;
  flex-direction: column;
}

.favorites-view__item {
  display: flex;
  align-items: center;
  border-bottom: 1px solid var(--divider);
}

.favorites-view__item:last-child {
  border-bottom: none;
}

.favorites-view__open {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.75em;
  padding: 0.75em 0.25em;
  background: none;
  border: none;
  color: var(--textPrimary);
  text-align: left;
  cursor: pointer;
  min-width: 0;
}

.favorites-view__open:hover {
  background: transparent;
}

.favorites-view__open i {
  flex-shrink: 0;
  color: var(--iconSecondary);
}

.favorites-view__info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.favorites-view__name {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.favorites-view__path {
  font-size: 0.85em;
  color: var(--textSecondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.favorites-view__remove {
  padding: 0.5em;
  flex-shrink: 0;
}

.favorites-view__remove i {
  color: #f7b500;
}

@media (max-width: 736px) {
  .favorites-view {
    padding: 1em;
  }
}
</style>
