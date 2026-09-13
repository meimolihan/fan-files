import { defineStore } from "pinia";
import { favorites as api } from "@/api";
import type { Favorite } from "@/api/favorites";

export const useFavoritesStore = defineStore("favorites", {
  state: (): {
    favorites: Favorite[];
  } => ({
    favorites: [],
  }),
  getters: {
    isFavorite: (state) => (path: string) =>
      state.favorites.some((f) => f.path === path),
  },
  actions: {
    async init() {
      try {
        this.favorites = await api.getFavorites();
      } catch {
        // A transient failure must not clear the favorites that are already
        // loaded; keep the previous snapshot and retry on next use.
      }
    },
    async add(path: string) {
      this.favorites = await api.addFavorite(path);
    },
    async remove(path: string) {
      this.favorites = await api.removeFavorite(path);
    },
    async toggle(path: string) {
      if (this.isFavorite(path)) {
        await this.remove(path);
      } else {
        await this.add(path);
      }
    },
  },
});
