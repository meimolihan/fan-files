<template>
  <div v-show="active" @click="closeHovers" class="overlay"></div>
  <nav :class="{ active }">
    <template v-if="isLoggedIn">
      <button @click="toAccountSettings" class="action">
        <i class="material-icons">person</i>
        <span>{{ user.username }}</span>
      </button>
      <button
        class="action"
        @click="toRoot"
        :aria-label="$t('sidebar.myFiles')"
        :title="$t('sidebar.myFiles')"
      >
        <i class="material-icons">folder</i>
        <span>{{ $t("sidebar.myFiles") }}</span>
      </button>

      <button
        class="action"
        @click="toFavorites"
        :aria-label="$t('sidebar.myFavorites')"
        :title="$t('sidebar.myFavorites')"
      >
        <i class="material-icons">star</i>
        <span>{{ $t("sidebar.myFavorites") }}</span>
      </button>

      <div v-if="user.perm.create">
        <button
          @click="showHover('newDir')"
          class="action"
          :aria-label="$t('sidebar.newFolder')"
          :title="$t('sidebar.newFolder')"
        >
          <i class="material-icons">create_new_folder</i>
          <span>{{ $t("sidebar.newFolder") }}</span>
        </button>

        <button
          @click="showHover('newFile')"
          class="action"
          :aria-label="$t('sidebar.newFile')"
          :title="$t('sidebar.newFile')"
        >
          <i class="material-icons">note_add</i>
          <span>{{ $t("sidebar.newFile") }}</span>
        </button>
      </div>

      <div v-if="user.perm.admin">
        <button
          class="action"
          @click="toGlobalSettings"
          :aria-label="$t('sidebar.settings')"
          :title="$t('sidebar.settings')"
        >
          <i class="material-icons">settings_applications</i>
          <span>{{ $t("sidebar.settings") }}</span>
        </button>
      </div>
      <button
        v-if="canLogout"
        @click="logout"
        class="action"
        id="logout"
        :aria-label="$t('sidebar.logout')"
        :title="$t('sidebar.logout')"
      >
        <i class="material-icons">exit_to_app</i>
        <span>{{ $t("sidebar.logout") }}</span>
      </button>
    </template>
    <template v-else>
      <router-link
        v-if="!hideLoginButton"
        class="action"
        to="/login"
        :aria-label="$t('sidebar.login')"
        :title="$t('sidebar.login')"
      >
        <i class="material-icons">exit_to_app</i>
        <span>{{ $t("sidebar.login") }}</span>
      </router-link>

      <router-link
        v-if="signup"
        class="action"
        to="/login"
        :aria-label="$t('sidebar.signup')"
        :title="$t('sidebar.signup')"
      >
        <i class="material-icons">person_add</i>
        <span>{{ $t("sidebar.signup") }}</span>
      </router-link>
    </template>

    <div class="credits storage-usage" v-if="!disableUsedPercentage">
      <div class="storage-usage__item" v-for="s in usage" :key="s.label">
        <p v-if="usage.length > 1" class="storage-usage__label">
          {{ s.label }}
        </p>
        <progress-bar
          class="storage-usage__bar"
          :val="s.usedPercentage"
          size="small"
        ></progress-bar>
        <p class="storage-usage__amount">
          {{ $t("sidebar.diskUsed", { used: s.used, total: s.total }) }}
        </p>
      </div>
    </div>

    <p class="credits">
      <span>
        <span v-if="disableExternal">fan-files</span>
        <a
          v-else
          rel="noopener noreferrer"
          target="_blank"
          href="https://github.com/filebrowser/filebrowser"
          >fan-files</a
        >
        <span> {{ " " }} {{ version }}</span>
      </span>
      <span>
        <a @click="help">{{ $t("sidebar.help") }}</a>
      </span>
    </p>
  </nav>
</template>

<script>
import { reactive } from "vue";
import { mapActions, mapState } from "pinia";
import { useAuthStore } from "@/stores/auth";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";

import * as auth from "@/utils/auth";
import {
  version,
  signup,
  hideLoginButton,
  disableExternal,
  disableUsedPercentage,
  noAuth,
  logoutPage,
  loginPage,
} from "@/utils/constants";
import { files as api } from "@/api";
import ProgressBar from "@/components/ProgressBar.vue";
import prettyBytes from "pretty-bytes";

export default {
  name: "sidebar",
  setup() {
    const usage = reactive([]);
    return { usage, usageAbortController: new AbortController() };
  },
  components: {
    ProgressBar,
  },
  inject: ["$showError"],
  computed: {
    ...mapState(useAuthStore, ["user", "isLoggedIn"]),
    ...mapState(useFileStore, ["reload"]),
    ...mapState(useLayoutStore, ["currentPromptName"]),
    active() {
      return this.currentPromptName === "sidebar";
    },
    signup: () => signup,
    hideLoginButton: () => hideLoginButton,
    version: () => version,
    disableExternal: () => disableExternal,
    disableUsedPercentage: () => disableUsedPercentage,
    canLogout: () => !noAuth && (loginPage || logoutPage !== "/login"),
  },
  methods: {
    ...mapActions(useLayoutStore, ["closeHovers", "showHover"]),
    abortOngoingFetchUsage() {
      this.usageAbortController.abort();
    },
    async fetchUsage() {
      if (this.disableUsedPercentage) {
        this.usage.splice(0, this.usage.length);
        return;
      }
      this.abortOngoingFetchUsage();
      const controller = new AbortController();
      this.usageAbortController = controller;
      try {
        const res = await api.storagesUsage(controller.signal);
        if (controller !== this.usageAbortController) return;
        const usageStats = (res || []).map((s) => ({
          label: s.label,
          used: prettyBytes(s.used, { binary: true }),
          total: prettyBytes(s.total, { binary: true }),
          usedPercentage:
            s.total > 0 ? Math.round((s.used / s.total) * 100) : 0,
        }));
        this.usage.splice(0, this.usage.length, ...usageStats);
      } catch {
        // A transient failure (e.g. a refused connection while browsing)
        // must not wipe the storage usage that is already shown in the
        // sidebar; keep the previous snapshot and retry on next navigation.
      }
    },
    toRoot() {
      this.$router.push({ path: "/files" });
      this.closeHovers();
    },
    toFavorites() {
      this.$router.push({ path: "/favorites" });
      this.closeHovers();
    },
    toAccountSettings() {
      this.$router.push({ path: "/settings/profile" });
      this.closeHovers();
    },
    toGlobalSettings() {
      this.$router.push({ path: "/settings/global" });
      this.closeHovers();
    },
    help() {
      this.showHover("help");
    },
    logout: auth.logout,
  },
  watch: {
    $route: {
      handler() {
        this.fetchUsage();
      },
      immediate: true,
    },
  },
  unmounted() {
    this.abortOngoingFetchUsage();
  },
};
</script>

<style scoped>
.storage-usage {
  width: calc(100% - 5em);
  margin: 2em 2.5em 3em;
}

.storage-usage__item + .storage-usage__item {
  margin-top: 1em;
  padding-top: 0.75em;
  border-top: 1px dashed var(--borderPrimary);
}

.storage-usage__label {
  margin: 0 0 0.45em;
  font-size: 1.1em;
  font-weight: 600;
  color: var(--textPrimary);
}

.storage-usage__bar {
  display: block;
  width: 100%;
}

.storage-usage__amount {
  margin: 0.45em 0 0;
  color: var(--textSecondary);
}
</style>
