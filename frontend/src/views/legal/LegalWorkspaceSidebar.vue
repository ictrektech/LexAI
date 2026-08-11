<template>
  <aside class="legal-sidebar" :class="{ 'legal-sidebar--collapsed': sidebarProps.collapsed }">
    <div class="legal-sidebar__header">
      <button class="legal-sidebar__brand" type="button" aria-label="LexAI" @click="goHome">
        <span class="legal-sidebar__brand-mark">L</span>
        <span v-if="!sidebarProps.collapsed" class="legal-sidebar__brand-name">LexAI</span>
      </button>
      <button
        v-if="!sidebarProps.collapsed"
        class="legal-sidebar__collapse"
        type="button"
        :aria-label="t('legalWorkspace.collapseSidebar')"
        :title="t('legalWorkspace.collapseSidebar')"
        @click="$emit('toggle')"
      >
        <t-icon name="chevron-left-double" size="17px" />
      </button>
    </div>

    <nav class="legal-sidebar__navigation" :aria-label="t('legalWorkspace.navigationLabel')">
      <div class="legal-sidebar__primary">
        <NavButton v-for="item in primaryItems" :key="item.id" :item="item" primary />
      </div>

      <div class="legal-sidebar__section">
        <div v-if="!sidebarProps.collapsed" class="legal-sidebar__section-label">{{ t('legalWorkspace.tools') }}</div>
        <NavButton v-for="item in toolItems" :key="item.id" :item="item" />
      </div>

      <div class="legal-sidebar__section legal-sidebar__section--resources">
        <div v-if="!sidebarProps.collapsed" class="legal-sidebar__section-label">{{ t('legalWorkspace.resources') }}</div>
        <NavButton v-for="item in resourceItems" :key="item.id" :item="item" />
      </div>
    </nav>

    <t-tooltip v-if="sidebarProps.collapsed" :content="t('legalWorkspace.expandSidebar')" placement="right">
      <button
        class="legal-sidebar__expand"
        type="button"
        :aria-label="t('legalWorkspace.expandSidebar')"
        @click="$emit('toggle')"
      >
        <t-icon name="chevron-right-double" size="17px" />
      </button>
    </t-tooltip>
  </aside>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { Icon as TIcon, Tooltip as TTooltip } from 'tdesign-vue-next'

import {
  isLegalWorkspaceItemActive,
  legalWorkspaceItemsFor,
  type LegalWorkspaceNavItem,
} from '@/config/legalWorkspace'
import { LEGAL_ASSISTANT_HOME_ROUTE } from '@/router/paths'

const sidebarProps = defineProps<{ collapsed: boolean }>()
defineEmits<{ toggle: [] }>()

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const primaryItems = legalWorkspaceItemsFor('primary')
const toolItems = legalWorkspaceItemsFor('tools')
const resourceItems = legalWorkspaceItemsFor('resources')

const goHome = () => router.push({ name: LEGAL_ASSISTANT_HOME_ROUTE })

const NavButton = defineComponent({
  name: 'LegalWorkspaceNavButton',
  props: {
    item: { type: Object as PropType<LegalWorkspaceNavItem>, required: true },
    primary: { type: Boolean, default: false },
  },
  setup(props) {
    const active = computed(() => isLegalWorkspaceItemActive(props.item, route.name))
    const label = computed(() => t(props.item.labelKey))
    const badge = computed(() => props.item.badgeKey ? t(props.item.badgeKey) : '')

    const navigate = () => {
      if (props.item.disabled || !props.item.destination) return
      router.push(props.item.destination)
    }

    return () => {
      const button = h('button', {
        type: 'button',
        class: [
          'legal-nav-item',
          props.primary && 'legal-nav-item--primary',
          active.value && 'legal-nav-item--active',
          props.item.disabled && 'legal-nav-item--disabled',
        ],
        disabled: props.item.disabled,
        'aria-current': active.value ? 'page' : undefined,
        'aria-label': label.value,
        onClick: navigate,
      }, [
        h(TIcon, { class: 'legal-nav-item__icon', name: props.item.icon, size: '19px' }),
        !sidebarProps.collapsed
          ? h('span', { class: 'legal-nav-item__content' }, [
              h('span', { class: 'legal-nav-item__label' }, label.value),
              badge.value ? h('span', { class: 'legal-nav-item__badge' }, badge.value) : null,
            ])
          : null,
      ])

      if (!sidebarProps.collapsed) return button
      return h(TTooltip, { content: badge.value ? `${label.value} · ${badge.value}` : label.value, placement: 'right' }, {
        default: () => h('span', { class: 'legal-nav-tooltip-anchor' }, [button]),
      })
    }
  },
})
</script>

<style scoped lang="less">
.legal-sidebar {
  width: 240px;
  min-width: 240px;
  height: 100%;
  padding: 18px 12px 14px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  color: #171715;
  background: #fff;
  border-right: 1px solid #e8e8e5;
  transition: width 180ms ease, min-width 180ms ease, padding 180ms ease;

  &--collapsed {
    width: 64px;
    min-width: 64px;
    padding-inline: 8px;
  }
}

.legal-sidebar__header {
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  margin-bottom: 22px;
}

.legal-sidebar__brand,
.legal-sidebar__collapse,
.legal-sidebar__expand,
:deep(.legal-nav-item) {
  border: 0;
  font: inherit;
}

:deep(.legal-nav-tooltip-anchor) {
  width: 100%;
  display: block;
}

.legal-sidebar__brand {
  min-width: 0;
  padding: 0 7px;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: #111;
  background: transparent;
  cursor: pointer;
}

.legal-sidebar__brand-mark {
  width: 28px;
  height: 28px;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 28px;
  color: #fff;
  background: #111;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.04em;
}

.legal-sidebar__brand-name {
  font-size: 20px;
  line-height: 1;
  font-weight: 700;
  letter-spacing: -0.04em;
}

.legal-sidebar__collapse,
.legal-sidebar__expand {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #686863;
  background: transparent;
  cursor: pointer;

  &:hover {
    color: #111;
    background: #f2f2ef;
  }
}

.legal-sidebar__navigation {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.legal-sidebar__primary {
  margin-bottom: 22px;
}

.legal-sidebar__section {
  display: flex;
  flex-direction: column;
  gap: 3px;

  &--resources {
    margin-top: auto;
    padding-top: 20px;
    border-top: 1px solid #efefec;
  }
}

.legal-sidebar__section-label {
  margin: 0 10px 8px;
  color: #92928b;
  font-size: 11px;
  font-weight: 650;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

:deep(.legal-nav-item) {
  width: 100%;
  min-height: 40px;
  padding: 0 11px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  gap: 11px;
  color: #494945;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: color 140ms ease, background 140ms ease;

  &:hover:not(:disabled) {
    color: #111;
    background: #f3f3f0;
  }
}

:deep(.legal-nav-item--primary) {
  color: #fff;
  background: #171715;

  &:hover:not(:disabled) {
    color: #fff;
    background: #30302d;
  }
}

:deep(.legal-nav-item--active:not(.legal-nav-item--primary)) {
  color: #111;
  background: #ecece8;
  font-weight: 650;
}

:deep(.legal-nav-item--disabled) {
  color: #aaa9a3;
  cursor: not-allowed;
}

:deep(.legal-nav-item__icon) {
  flex: 0 0 19px;
}

:deep(.legal-nav-item__content) {
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

:deep(.legal-nav-item__label) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 550;
}

:deep(.legal-nav-item__badge) {
  flex-shrink: 0;
  color: #a2a19b;
  font-size: 9px;
  font-weight: 650;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

.legal-sidebar--collapsed {
  .legal-sidebar__header {
    justify-content: center;
  }

  .legal-sidebar__brand {
    padding: 0;
  }

  :deep(.legal-nav-item) {
    justify-content: center;
    padding: 0;
  }

  .legal-sidebar__section--resources {
    align-items: stretch;
  }
}

.legal-sidebar__expand {
  margin: 12px auto 0;
  flex-shrink: 0;
}
</style>
