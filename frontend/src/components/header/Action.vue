<template>
  <button
    @click="action"
    :aria-label="label"
    :title="label"
    class="action"
    :disabled="disabled"
  >
    <i class="material-icons">{{ icon }}</i>
    <span>{{ label }}</span>
    <span v-if="counter && counter > 0" class="counter">{{ counter }}</span>
  </button>
</template>

<script setup lang="ts">
import { useLayoutStore } from "@/stores/layout";

const props = defineProps<{
  icon?: string;
  label?: string;
  counter?: number;
  show?: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (e: "action"): any;
}>();

const layoutStore = useLayoutStore();

const action = () => {
  if (props.disabled) return;
  if (props.show) {
    layoutStore.showHover(props.show);
  }

  emit("action");
};
</script>
