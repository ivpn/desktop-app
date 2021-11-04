<template>
  <div>
    <svg
      v-if="platform == PlatformEnum.Linux"
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M16 8.5H1.95312L8.10156 14.6484L7.39844 15.3516L0.046875 8L7.39844 0.648438L8.10156 1.35156L1.95312 7.5H16V8.5Z"
        fill="var(--text-color)"
      />
    </svg>
    <svg
      v-else-if="platform == PlatformEnum.Windows"
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M16.001 8.5H1.9541L8.10254 14.6484L7.39941 15.3516L0.0478516 8L7.39941 0.648438L8.10254 1.35156L1.9541 7.5H16.001V8.5Z"
        fill="var(--text-color)"
      />
    </svg>
    <svg
      v-else
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M12.6666 8H3.33331"
        stroke="var(--text-color)"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        d="M7.99998 12.6663L3.33331 7.99968L7.99998 3.33301"
        stroke="var(--text-color)"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  </div>
</template>

<script>
import { Platform, PlatformEnum } from "@/platform/platform";

export default {
  computed: {
    PlatformEnum: function () {
      return PlatformEnum;
    },
    platform: function () {
      return Platform();
    },
  },
};
</script>

<style scoped lang="scss">
svg {
  vertical-align: middle;
}
</style>
