<template>
  <aside class="legal-sidebar" :class="{ 'legal-sidebar--collapsed': sidebarProps.collapsed }">
    <div class="legal-sidebar__header">
      <button class="legal-sidebar__brand" type="button" aria-label="LexAI" @click="goHome">
        <img
          v-if="!sidebarProps.collapsed"
          class="legal-sidebar__brand-logo"
          :src="lexaiLogo"
          alt="LexAI"
        />
        <span v-else class="legal-sidebar__brand-mark" aria-hidden="true">L</span>
      </button>
      <button
        v-if="!sidebarProps.collapsed"
        class="legal-sidebar__collapse"
        type="button"
        data-testid="legal-sidebar-collapse"
        :aria-label="t('legalWorkspace.collapseSidebar')"
        :title="t('legalWorkspace.collapseSidebar')"
        @click="$emit('toggle')"
      >
        <t-icon name="chevron-left-double" size="17px" />
      </button>
    </div>

    <nav class="legal-sidebar__navigation" :aria-label="t('legalWorkspace.navigationLabel')">
      <Transition :name="navigationTransition" mode="out-in">
        <div :key="navigationLevelKey" class="legal-sidebar__navigation-panel">
          <template v-if="isDrilledDown">
            <div class="legal-sidebar__subnav-header">
              <button
                class="legal-sidebar__back"
                type="button"
                data-testid="legal-nav-back"
                :aria-label="t('legalWorkspace.backToNavigation')"
                :title="t('legalWorkspace.backToNavigation')"
                @click="goBack"
              >
                <t-icon name="chevron-left" size="17px" />
                <span v-if="!sidebarProps.collapsed">{{ currentLevelTitle }}</span>
              </button>
            </div>
            <div class="legal-sidebar__subnav-items">
              <NavButton v-for="item in currentLevelItems" :key="item.id" :item="item" />
            </div>
          </template>

          <template v-else>
            <div v-if="primaryItems.length" class="legal-sidebar__primary">
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
          </template>
        </div>
      </Transition>
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
import { computed, defineComponent, h, ref, type PropType, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { Icon as TIcon, Tooltip as TTooltip } from 'tdesign-vue-next'
import lexaiLogo from '@/assets/img/LexAI_logo_exact.svg'

import {
  LEGAL_WORKSPACE_NAV_ITEMS,
  isLegalWorkspaceItemActive,
  legalWorkspaceNavPath,
  legalWorkspaceItemsFor,
  type LegalWorkspaceNavItem,
} from '@/config/legalWorkspace'
import { LEGAL_ASSISTANT_HOME_ROUTE } from '@/router/paths'

const sidebarProps = defineProps<{ collapsed: boolean }>()
const emit = defineEmits<{ toggle: []; expand: [] }>()

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const primaryItems = legalWorkspaceItemsFor('primary')
const toolItems = legalWorkspaceItemsFor('tools')
const resourceItems = legalWorkspaceItemsFor('resources')
const navigationStack = ref<string[]>([])
const navigationTransition = ref<'nav-forward' | 'nav-back'>('nav-forward')

const navigationLevelKey = computed(() => navigationStack.value.length ? navigationStack.value.join('/') : 'root')
const isDrilledDown = computed(() => navigationStack.value.length > 0)

function resolveNavigationLevel(stack: readonly string[]) {
  let items: readonly LegalWorkspaceNavItem[] = LEGAL_WORKSPACE_NAV_ITEMS
  let item: LegalWorkspaceNavItem | undefined

  for (const id of stack) {
    item = items.find((candidate) => candidate.id === id)
    if (!item) return { item: undefined, items: [] as readonly LegalWorkspaceNavItem[] }
    items = item.children || []
  }

  return { item, items }
}

const currentNavigationLevel = computed(() => resolveNavigationLevel(navigationStack.value))
const currentLevelItems = computed(() => currentNavigationLevel.value.items)
const currentLevelTitle = computed(() => currentNavigationLevel.value.item ? t(currentNavigationLevel.value.item.labelKey) : '')

function stackForRoute(routeName: unknown): string[] {
  const path = legalWorkspaceNavPath(routeName)
  return path.length > 1 ? path.slice(0, -1).map((item) => item.id) : []
}

function sameStack(left: readonly string[], right: readonly string[]) {
  return left.length === right.length && left.every((value, index) => value === right[index])
}

function setNavigationStack(nextStack: readonly string[], transition: 'nav-forward' | 'nav-back') {
  if (sameStack(navigationStack.value, nextStack)) return
  navigationTransition.value = transition
  navigationStack.value = [...nextStack]
}

function pushNavigation(id: string) {
  setNavigationStack([...navigationStack.value, id], 'nav-forward')
}

function popNavigation() {
  if (!navigationStack.value.length) return
  setNavigationStack(navigationStack.value.slice(0, -1), 'nav-back')
}

function resetNavigation() {
  setNavigationStack([], 'nav-back')
}

function syncNavigationFromRoute(routeName: unknown) {
  const nextStack = stackForRoute(routeName)
  if (!nextStack.length) resetNavigation()
  else setNavigationStack(nextStack, nextStack.length >= navigationStack.value.length ? 'nav-forward' : 'nav-back')
  if (nextStack.length && sidebarProps.collapsed) emit('expand')
}

watch(() => route.name, syncNavigationFromRoute, { immediate: true })

const goHome = () => router.push({ name: LEGAL_ASSISTANT_HOME_ROUTE })

function handleNavigation(item: LegalWorkspaceNavItem) {
  if (item.disabled) return
  if (item.children?.length) {
    pushNavigation(item.id)
    if (sidebarProps.collapsed) emit('expand')
    return
  }
  if (item.destination) void router.push(item.destination)
}

function goBack() {
  popNavigation()
}

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

    return () => {
      const button = h('button', {
        type: 'button',
        'data-testid': `legal-nav-${props.item.id}`,
        class: [
          'legal-nav-item',
          props.primary && 'legal-nav-item--primary',
          active.value && 'legal-nav-item--active',
          props.item.disabled && 'legal-nav-item--disabled',
        ],
        disabled: props.item.disabled,
        'aria-current': active.value ? 'page' : undefined,
        'aria-label': label.value,
        'aria-haspopup': props.item.children?.length ? 'menu' : undefined,
        'aria-expanded': props.item.children?.length ? navigationStack.value.includes(props.item.id) : undefined,
        onClick: () => handleNavigation(props.item),
      }, [
        h(TIcon, { class: 'legal-nav-item__icon', name: props.item.icon, size: '19px' }),
        !sidebarProps.collapsed
          ? h('span', { class: 'legal-nav-item__content' }, [
              h('span', { class: 'legal-nav-item__label' }, label.value),
              badge.value ? h('span', { class: 'legal-nav-item__badge' }, badge.value) : null,
              props.item.children?.length ? h(TIcon, { class: 'legal-nav-item__chevron', name: 'chevron-right', size: '15px' }) : null,
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
  color: var(--legal-text-primary);
  background: var(--legal-bg-surface);
  border-right: 1px solid var(--legal-border);
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

  &:focus-visible {
    outline: 2px solid var(--legal-ai);
    outline-offset: 2px;
  }
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
  justify-content: center;
  gap: 10px;
  color: var(--legal-brand);
  background: transparent;
  cursor: pointer;
  overflow: hidden;
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
  background: var(--legal-brand);
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.04em;
}

.legal-sidebar__brand-logo {
  display: block;
  width: 106px;
  height: 38px;
  object-fit: contain;
  // The supplied SVG includes transparent canvas on both sides and slightly
  // more space below the artwork. Reposition and enlarge the image inside the
  // existing 38px header without changing the source asset.
  object-position: 25% center;
  position: relative;
  top: -1px;
  transform: scale(1.2);
  transform-origin: center;
}

.legal-sidebar__collapse,
.legal-sidebar__expand {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--legal-text-secondary);
  background: transparent;
  cursor: pointer;

  &:hover {
    color: var(--legal-brand);
    background: var(--legal-bg-hover);
  }
}

.legal-sidebar__navigation {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.legal-sidebar__navigation-panel {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.legal-sidebar__subnav-header {
  min-height: 40px;
  margin-bottom: 14px;
  display: flex;
  align-items: center;
}

.legal-sidebar__back {
  min-width: 0;
  min-height: 36px;
  padding: 0 9px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 0;
  border-radius: 8px;
  color: var(--legal-text-primary);
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  font-weight: 650;

  &:hover {
    background: var(--legal-bg-hover);
  }
}

.legal-sidebar__subnav-items {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.nav-forward-enter-active,
.nav-forward-leave-active,
.nav-back-enter-active,
.nav-back-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}

.nav-forward-enter-from { opacity: 0; transform: translateX(14px); }
.nav-forward-leave-to { opacity: 0; transform: translateX(-14px); }
.nav-back-enter-from { opacity: 0; transform: translateX(-14px); }
.nav-back-leave-to { opacity: 0; transform: translateX(14px); }

@media (prefers-reduced-motion: reduce) {
  .nav-forward-enter-active,
  .nav-forward-leave-active,
  .nav-back-enter-active,
  .nav-back-leave-active {
    transition-duration: 0ms;
  }
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
    border-top: 1px solid var(--legal-border);
  }
}

.legal-sidebar__section-label {
  margin: 0 10px 8px;
  color: var(--legal-text-secondary);
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
  color: var(--legal-text-secondary);
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: color 140ms ease, background 140ms ease;

  &:hover:not(:disabled) {
    color: var(--legal-text-primary);
    background: var(--legal-bg-hover);
  }
}

:deep(.legal-nav-item--primary) {
  color: #fff;
  background: var(--legal-brand);

  &:hover:not(:disabled) {
    color: #fff;
    background: var(--legal-brand-hover);
  }
}

:deep(.legal-nav-item--active:not(.legal-nav-item--primary)) {
  color: var(--legal-ai-strong);
  background: var(--legal-ai-soft);
  font-weight: 650;
}

:deep(.legal-nav-item--disabled) {
  color: var(--legal-text-disabled);
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
  color: var(--legal-text-disabled);
  font-size: 9px;
  font-weight: 650;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

:deep(.legal-nav-item__chevron) {
  flex: 0 0 15px;
  color: var(--legal-text-disabled);
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
