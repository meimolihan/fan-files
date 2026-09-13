<template>
  <div class="card floating">
    <div class="card-title">
      <h2>{{ t("prompts.newFile") }}</h2>
    </div>

    <div class="card-content">
      <p>{{ t("prompts.newFileMessage") }}</p>
      <input
        id="focus-prompt"
        class="input input--block"
        type="text"
        @keyup.enter="submit"
        v-model.trim="name"
        tabindex="1"
      />
      <label v-if="storageOptions.length > 1" class="storage-label">
        <span>{{ t("prompts.selectStorage") }}</span>
        <select
          class="input input--block"
          v-model="selectedStorage"
          :aria-label="t('prompts.selectStorage')"
          tabindex="2"
        >
          <option
            v-for="storage in storageOptions"
            :key="storage"
            :value="storage"
          >
            {{ storage }}
          </option>
        </select>
      </label>
      <CreateFilePath
        :name="name"
        :storage="storageOptions.length > 1 ? selectedStorage : undefined"
      />
    </div>

    <div class="card-action">
      <button
        class="button button--flat button--grey"
        @click="layoutStore.closeHovers"
        :aria-label="t('buttons.cancel')"
        :title="t('buttons.cancel')"
        tabindex="4"
      >
        {{ t("buttons.cancel") }}
      </button>
      <button
        class="button button--flat"
        @click="submit"
        :aria-label="t('buttons.create')"
        :title="t('buttons.create')"
        tabindex="3"
      >
        {{ t("buttons.create") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { inject, onActivated, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";
import CreateFilePath from "@/components/prompts/CreateFilePath.vue";

import { files as api } from "@/api";
import url from "@/utils/url";

const $showError = inject<IToastError>("$showError")!;

const fileStore = useFileStore();
const layoutStore = useLayoutStore();

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const name = ref<string>("");
const storageOptions = ref<string[]>([]);
const selectedStorage = ref<string>("");

const loadStorages = async () => {
  try {
    storageOptions.value = await api.storages();
  } catch {
    storageOptions.value = [];
  }

  // Prefer the storage holding the current directory, falling back to the
  // primary volume.
  const current = fileStore.req?.storage;
  if (current && storageOptions.value.includes(current)) {
    selectedStorage.value = current;
  } else {
    selectedStorage.value = storageOptions.value[0] || "";
  }
};

onActivated(() => {
  name.value = "";
  loadStorages();
});

const submit = async (event: Event) => {
  event.preventDefault();
  if (name.value === "") return;

  // Build the path of the new directory.
  let uri = fileStore.isFiles ? route.path + "/" : "/";

  if (!fileStore.isListing) {
    uri = url.removeLastDir(uri) + "/";
  }

  uri += encodeURIComponent(name.value);
  uri = uri.replace("//", "/");

  try {
    await api.post(
      uri,
      "",
      false,
      () => {},
      selectedStorage.value || undefined
    );
    router.push({ path: uri });
  } catch (e) {
    if (e instanceof Error) {
      $showError(e);
    }
  }

  layoutStore.closeHovers();
};
</script>

<style scoped>
.storage-label {
  display: block;
  margin: 0.5em 0 0.2em;
}

.storage-label > span {
  display: inline-block;
  margin-bottom: 0.3em;
  font-size: 0.9em;
  opacity: 0.7;
}
</style>
