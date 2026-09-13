<template>
  <div class="card floating">
    <div class="card-content">
      <div v-if="names.length" class="delete-target">
        <span v-if="names.length === 1" class="delete-highlight">{{
          names[0]
        }}</span>
        <template v-else>
          <ul class="delete-list">
            <li v-for="(name, index) in names.slice(0, maxNames)" :key="index">
              <span class="delete-highlight">{{ name }}</span>
            </li>
          </ul>
          <p v-if="names.length > maxNames" class="delete-more">
            {{ $t("prompts.deleteMore", { count: names.length - maxNames }) }}
          </p>
        </template>
      </div>
      <p v-if="!this.isListing || selectedCount === 1">
        {{ $t("prompts.deleteMessageSingle") }}
      </p>
      <p v-else>
        {{ $t("prompts.deleteMessageMultiple", { count: selectedCount }) }}
      </p>
    </div>
    <div class="card-action">
      <button
        @click="closeHovers"
        class="button button--flat button--grey"
        :aria-label="$t('buttons.cancel')"
        :title="$t('buttons.cancel')"
        tabindex="2"
      >
        {{ $t("buttons.cancel") }}
      </button>
      <button
        id="focus-prompt"
        @click="submit"
        class="button button--flat button--red"
        :aria-label="$t('buttons.delete')"
        :title="$t('buttons.delete')"
        tabindex="1"
      >
        {{ $t("buttons.delete") }}
      </button>
    </div>
  </div>
</template>

<script>
import { mapActions, mapState, mapWritableState } from "pinia";
import { files as api } from "@/api";
import buttons from "@/utils/buttons";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";

export default {
  name: "delete",
  inject: ["$showError"],
  computed: {
    ...mapState(useFileStore, [
      "isListing",
      "selectedCount",
      "req",
      "selected",
    ]),
    ...mapState(useLayoutStore, ["currentPrompt"]),
    ...mapWritableState(useFileStore, ["reload", "preselect"]),
    names: function () {
      if (!this.isListing) {
        return this.req?.name ? [this.req.name] : [];
      }

      return this.selected
        .map((index) => this.req?.items[index]?.name)
        .filter((name) => name);
    },
  },
  data() {
    return {
      maxNames: 5,
    };
  },
  methods: {
    ...mapActions(useLayoutStore, ["closeHovers"]),
    submit: async function () {
      buttons.loading("delete");

      try {
        if (!this.isListing) {
          await api.remove(this.$route.path);
          buttons.success("delete");

          this.currentPrompt?.confirm();
          this.closeHovers();
          return;
        }

        this.closeHovers();

        if (this.selectedCount === 0) {
          return;
        }

        const promises = [];
        for (const index of this.selected) {
          promises.push(api.remove(this.req.items[index].url));
        }

        await Promise.all(promises);
        buttons.success("delete");

        const nearbyItem =
          this.req.items[Math.max(0, Math.min(this.selected) - 1)];

        this.preselect = nearbyItem?.path;

        this.reload = true;
      } catch (e) {
        buttons.done("delete");
        this.$showError(e);
        if (this.isListing) this.reload = true;
      }
    },
  },
};
</script>

<style scoped>
.delete-target {
  margin-bottom: 1em;
}

.delete-highlight {
  color: var(--icon-yellow);
  font-weight: 600;
  word-break: break-all;
}

.delete-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.delete-list li {
  padding: 0.1em 0;
  border-left: 3px solid var(--icon-yellow);
  padding-left: 0.6em;
}

.delete-more {
  margin: 0.5em 0 0;
}
</style>
