<template>
  <header>
    <img v-if="showLogo" :src="logoURL" alt="fan-files" />
    <Action
      v-if="showMenu"
      class="menu-button"
      icon="menu"
      :label="t('buttons.toggleSidebar')"
      @action="layoutStore.showHover('sidebar')"
    />

    <slot />

    <div
      id="dropdown"
      :class="{ active: layoutStore.currentPromptName === 'more' }"
    >
      <slot name="actions" />
    </div>

    <Action
      v-if="isLoggedIn"
      class="favorites-button"
      icon="star"
      :label="t('sidebar.myFavorites')"
      @action="toggleFavorites"
    />

    <ThemeToggle />

    <Action
      v-if="ifActionsSlot"
      id="more"
      icon="more_vert"
      :label="t('buttons.more')"
      @action="layoutStore.showHover('more')"
    />

    <div
      class="overlay"
      v-show="layoutStore.currentPromptName == 'more'"
      @click="layoutStore.closeHovers"
    />
  </header>
</template>

<script setup lang="ts">
import { useLayoutStore } from "@/stores/layout";
import { useAuthStore } from "@/stores/auth";
import { useFavoritesStore } from "@/stores/favorites";

import { logoURL } from "@/utils/constants";

import Action from "@/components/header/Action.vue";
import ThemeToggle from "@/components/header/ThemeToggle.vue";
import { computed, onMounted, useSlots } from "vue";
import { useI18n } from "vue-i18n";

defineProps<{
  showLogo?: boolean;
  showMenu?: boolean;
}>();

const layoutStore = useLayoutStore();
const authStore = useAuthStore();
const favoritesStore = useFavoritesStore();
const slots = useSlots();

const { t } = useI18n();

const ifActionsSlot = computed(() => (slots.actions ? true : false));

const isLoggedIn = computed(() => authStore.isLoggedIn);

onMounted(() => {
  if (authStore.isLoggedIn) {
    favoritesStore.init();
  }
});

const toggleFavorites = () => {
  if (layoutStore.currentPromptName === "favorites") {
    layoutStore.closeHovers();
  } else {
    layoutStore.showHover("favorites");
    favoritesStore.init();
  }
};
</script>

<style></style>
