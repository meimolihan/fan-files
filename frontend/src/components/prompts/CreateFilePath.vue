<template>
  <div>
    <div class="path-container" ref="container">
      <template v-for="(item, index) in path" :key="index">
        /
        <span class="path-item">
          <span
            v-if="isDir === true || index < path.length - 1"
            class="material-icons"
            >folder
          </span>
          <span v-else class="material-icons">insert_drive_file</span>
          {{ item }}
        </span>
      </template>
      <span
        v-if="storage"
        class="path-storage"
        :title="t('prompts.targetStorage')"
      >
        <i class="material-icons">sd_storage</i>
        {{ storage }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from "vue";
import { useRoute } from "vue-router";
import { useFileStore } from "@/stores/file";
import url from "@/utils/url";
import { useI18n } from "vue-i18n";

const fileStore = useFileStore();
const route = useRoute();
const { t } = useI18n();

const props = defineProps({
  name: {
    type: String,
    required: true,
  },
  isDir: {
    type: Boolean,
    default: false,
  },
  path: {
    type: String,
    default: null,
  },
  storage: {
    type: String,
    default: null,
  },
});

const container = ref<HTMLElement | null>(null);

const path = computed(() => {
  const routePath = props.path || route.path;
  let basePath = fileStore.isFiles ? routePath : url.removeLastDir(routePath);
  if (!basePath.endsWith("/")) {
    basePath += "/";
  }
  basePath += props.name;
  return basePath.split("/").filter(Boolean).splice(1);
});

watch(path, () => {
  nextTick(() => {
    const lastItem = container.value?.lastElementChild;
    lastItem?.scrollIntoView({ behavior: "auto", inline: "end" });
  });
});
</script>

<style scoped>
.path-container {
  display: flex;
  align-items: center;
  margin: 0.2em 0;
  gap: 0.25em;
  overflow-x: auto;
  max-width: 100%;
  scrollbar-width: none;
  opacity: 0.5;
}

.path-container::-webkit-scrollbar {
  display: none;
}

.path-item {
  display: flex;
  align-items: center;
  margin: 0.2em 0;
  gap: 0.25em;
  white-space: nowrap;
}

.path-item > span {
  font-size: 0.9em;
}

.path-storage {
  display: inline-flex;
  align-items: center;
  gap: 0.25em;
  margin-left: 0.5em;
  padding: 0.1em 0.5em;
  border-radius: 0.3em;
  background: var(--surfaceSecondary);
  color: var(--textPrimary);
  white-space: nowrap;
  font-size: 0.85em;
}

.path-storage > i {
  font-size: 1em;
}
</style>
