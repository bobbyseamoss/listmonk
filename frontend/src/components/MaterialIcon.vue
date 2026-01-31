<template>
  <span class="material-symbols-outlined" :class="sizeClass">{{ mappedIcon }}</span>
</template>

<script>
/* eslint-disable quote-props */
// Map MDI icon names to Google Material Symbol names
const iconMap = {
  // Navigation & Dashboard
  'view-dashboard-variant-outline': 'dashboard',
  'format-list-bulleted-square': 'list',
  'newspaper-variant-outline': 'article',
  'account-multiple': 'group',
  'file-upload-outline': 'upload_file',
  'rocket-launch-outline': 'rocket_launch',
  'plus': 'add',
  'image-outline': 'image',
  'file-image-outline': 'image',
  'cog-outline': 'settings',
  'tag-outline': 'label',
  'calendar-clock': 'schedule',
  'email-outline': 'mail',
  'text': 'text_fields',
  'alarm': 'alarm',
  'pause-circle-outline': 'pause_circle',
  'file-find-outline': 'find_in_page',
  'clock-start': 'schedule',
  'file-multiple-outline': 'content_copy',
  'file-multiple': 'file_copy',
  'trash-can-outline': 'delete',
  'pencil-outline': 'edit',
  'arrow-top-right': 'arrow_outward',
  'link-variant': 'link',
  'cloud-download-outline': 'cloud_download',
  'account-search-outline': 'person_search',
  'check-circle-outline': 'check_circle',
  'account-check-outline': 'how_to_reg',
  'account-off-outline': 'person_off',
  'chevron-right': 'chevron_right',
  'chevron-left': 'chevron_left',
  'chevron-down': 'expand_more',
  'content-save-outline': 'save',
  'minus': 'remove',
  'arrow-up': 'arrow_upward',
  'arrow-down': 'arrow_downward',
  'cancel': 'cancel',
  'magnify': 'search',
  'chart-bar': 'bar_chart',
  'email-bounce': 'mark_email_unread',
  'speedometer': 'speed',
  'warning-empty': 'warning',
  'account-outline': 'person',
  'logout-variant': 'logout',
  'code': 'code',
  'play': 'play_arrow',
  'pause': 'pause',
  'eye-outline': 'visibility',
  'filter-outline': 'filter_list',
  'refresh': 'refresh',
  'dots-vertical': 'more_vert',
  'account': 'person',
  'email': 'email',
  'flash': 'bolt',
  'clock-outline': 'schedule',
  'cursor-default-click': 'ads_click',
  'information': 'info',
  'alert': 'warning',
  'content-copy': 'content_copy',
  'sync': 'sync',
  'form-select': 'checklist',
  'shopify': 'shopping_cart',
  'help': 'help',
  'wrench-outline': 'build',
  'tray-full': 'inbox',
  'arrow-right': 'arrow_forward',
  'cart': 'shopping_cart',
  'microsoft-azure': 'cloud',
  'playlist-plus': 'playlist_add',
  'playlist-minus': 'playlist_remove',
  'account-edit-outline': 'manage_accounts',
  'code-tags': 'code',
  'lightning-bolt': 'bolt',
  'calendar-today': 'calendar_today',
};
/* eslint-enable quote-props */

export default {
  name: 'MaterialIcon',
  props: {
    icon: {
      type: [String, Array],
      required: true,
    },
    size: {
      type: String,
      default: '',
    },
  },
  computed: {
    // Extract the icon name from the prop (Buefy passes [pack, icon] array)
    iconName() {
      if (Array.isArray(this.icon)) {
        // Buefy passes [pack, iconName] - we want the iconName (second element)
        return this.icon[1] || this.icon[0];
      }
      return this.icon;
    },
    mappedIcon() {
      // Return mapped icon or the original if no mapping exists
      return iconMap[this.iconName] || this.iconName;
    },
    sizeClass() {
      if (this.size === 'is-small') return 'is-size-small';
      if (this.size === 'is-medium') return 'is-size-medium';
      if (this.size === 'is-large') return 'is-size-large';
      return '';
    },
  },
};
</script>

<style scoped>
.material-symbols-outlined {
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 20;
}
.is-size-small {
  font-size: 16px;
}
.is-size-medium {
  font-size: 28px;
}
.is-size-large {
  font-size: 36px;
}
</style>
