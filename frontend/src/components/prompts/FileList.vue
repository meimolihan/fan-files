<template>
  <div>
    <div class="file-list-crumbs">
      <span
        class="crumb"
        tabindex="0"
        role="button"
        :aria-label="$t('files.home')"
        :title="$t('files.home')"
        @click="goHome"
      >
        <i class="material-icons">home</i>
      </span>
      <template v-for="(crumb, index) in crumbs" :key="crumb.url">
        <i class="material-icons chevron">keyboard_arrow_right</i>
        <span
          :class="['crumb', { 'crumb--current': index === crumbs.length - 1 }]"
          tabindex="0"
          role="button"
          @click="goAt(crumb.url, index)"
        >
          {{ crumb.name }}
        </span>
      </template>
      <span
        v-if="crumbs.length > 0"
        class="crumb up-btn"
        tabindex="0"
        role="button"
        :aria-label="$t('prompts.upOneLevel')"
        :title="$t('prompts.upOneLevel')"
        @click="goUp"
      >
        <i class="material-icons">arrow_upward</i>
        <span class="up-btn__label">{{ $t("prompts.upOneLevel") }}</span>
      </span>
    </div>

    <ul class="file-list">
      <li
        @click="itemClick"
        @touchstart="touchstart"
        @dblclick="next"
        role="button"
        tabindex="0"
        :aria-label="item.name"
        :aria-selected="selected == item.url"
        :key="item.name"
        v-for="item in items"
        :data-url="item.url"
      >
        {{ item.name }}
      </li>
    </ul>

    <p>
      {{ $t("prompts.currentlyNavigating") }} <code>{{ nav }}</code
      >.
    </p>
  </div>
</template>

<script>
import { mapState, mapActions } from "pinia";
import { useAuthStore } from "@/stores/auth";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";

import url from "@/utils/url";
import { files } from "@/api";
import { StatusError } from "@/api/utils.js";

export default {
  name: "file-list",
  props: {
    exclude: {
      type: Array,
      default: () => [],
    },
  },
  data: function () {
    return {
      items: [],
      touches: {
        id: "",
        count: 0,
      },
      selected: null,
      current: window.location.pathname,
      nextAbortController: new AbortController(),
    };
  },
  inject: ["$showError"],
  computed: {
    ...mapState(useAuthStore, ["user"]),
    ...mapState(useFileStore, ["req"]),
    nav() {
      return decodeURIComponent(this.current);
    },
    crumbs() {
      const parts = this.current.split("/").filter((s) => s !== "");
      const segments = parts.slice(1);

      let acc = "/" + parts[0] + "/";
      const out = [];
      for (const segment of segments) {
        acc += segment + "/";
        let name = segment;
        try {
          name = decodeURIComponent(segment);
        } catch {
          // keep the raw segment if it isn't valid percent-encoding
        }
        out.push({ name, url: acc });
      }
      return out;
    },
    homeUrl() {
      const parts = this.current.split("/").filter((s) => s !== "");
      return "/" + parts[0] + "/";
    },
  },
  mounted() {
    this.fillOptions(this.req);
  },
  unmounted() {
    this.abortOngoingNext();
  },
  methods: {
    ...mapActions(useLayoutStore, ["showHover"]),
    abortOngoingNext() {
      this.nextAbortController.abort();
    },
    fillOptions(req) {
      // Sets the current path and resets
      // the current items.
      this.current = req.url;
      this.items = [];

      this.$emit("update:selected", this.current);

      // If the path isn't the root path,
      // show a button to navigate to the previous
      // directory.
      if (req.url !== "/files/") {
        this.items.push({
          name: "..",
          url: url.removeLastDir(req.url) + "/",
        });
      }

      // If this folder is empty, finish here.
      if (req.items === null) return;

      // Otherwise we add every directory to the
      // move options.
      for (const item of req.items) {
        if (!item.isDir) continue;
        if (this.exclude?.includes(item.url)) continue;

        this.items.push({
          name: item.name,
          url: item.url,
        });
      }
    },
    next: function (event) {
      // Retrieves the URL of the directory the user
      // just clicked in and fill the options with its
      // content.
      this.load(event.currentTarget.dataset.url);
    },
    load: function (uri) {
      this.abortOngoingNext();
      this.nextAbortController = new AbortController();
      files
        .fetch(uri, this.nextAbortController.signal)
        .then(this.fillOptions)
        .catch((e) => {
          if (e instanceof StatusError && e.is_canceled) {
            return;
          }
          this.$showError(e);
        });
    },
    goAt: function (url, index) {
      // Clicking the current (last) crumb is a no-op.
      if (index === this.crumbs.length - 1) return;
      this.load(url);
    },
    goHome: function () {
      if (this.current === this.homeUrl) return;
      this.load(this.homeUrl);
    },
    goUp: function () {
      this.load(url.removeLastDir(this.current) + "/");
    },
    touchstart(event) {
      const url = event.currentTarget.dataset.url;

      // In 300 milliseconds, we shall reset the count.
      setTimeout(() => {
        this.touches.count = 0;
      }, 300);

      // If the element the user is touching
      // is different from the last one he touched,
      // reset the count.
      if (this.touches.id !== url) {
        this.touches.id = url;
        this.touches.count = 1;
        return;
      }

      this.touches.count++;

      // If there is more than one touch already,
      // open the next screen.
      if (this.touches.count > 1) {
        this.next(event);
      }
    },
    itemClick: function (event) {
      if (this.user.singleClick) this.next(event);
      else this.select(event);
    },
    select: function (event) {
      // If the element is already selected, unselect it.
      if (this.selected === event.currentTarget.dataset.url) {
        this.selected = null;
        this.$emit("update:selected", this.current);
        return;
      }

      // Otherwise select the element.
      this.selected = event.currentTarget.dataset.url;
      this.$emit("update:selected", this.selected);
    },
    createDir: async function () {
      this.showHover({
        prompt: "newDir",
        action: null,
        confirm: (url) => {
          const paths = url.split("/");
          this.items.push({
            name: paths[paths.length - 2],
            url: url,
          });
        },
        props: {
          redirect: false,
          base: this.current === this.$route.path ? null : this.current,
        },
      });
    },
  },
};
</script>

<style scoped>
.file-list-crumbs {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  overflow-x: auto;
  scrollbar-width: none;
  padding: 0.3em 0.2em 0.4em;
  border-bottom: 1px solid var(--divider);
  margin-bottom: 0.4em;
  user-select: none;
}

.file-list-crumbs::-webkit-scrollbar {
  display: none;
}

.file-list-crumbs .crumb {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 0.2em;
  padding: 0.25em 0.5em;
  border-radius: 0.4em;
  color: var(--textPrimary);
  white-space: nowrap;
  cursor: pointer;
}

.file-list-crumbs .crumb:hover,
.file-list-crumbs .crumb:focus-visible {
  background: var(--divider);
  outline: none;
}

.file-list-crumbs .crumb .material-icons {
  font-size: 1em;
}

.file-list-crumbs .crumb--current {
  color: var(--textSecondary);
  font-weight: 600;
  cursor: default;
}

.file-list-crumbs .crumb--current:hover {
  background: transparent;
}

.file-list-crumbs .chevron {
  font-size: 1em;
  opacity: 0.5;
  flex: 0 0 auto;
}

.file-list-crumbs .up-btn {
  margin-left: auto;
  color: var(--blue);
}

.file-list-crumbs .up-btn .up-btn__label {
  font-size: 0.85em;
}
</style>
