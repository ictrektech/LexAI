import type { RouteLocationRaw } from 'vue-router'

import {
  LEGAL_ASSISTANT_CHAT_ROUTE,
  LEGAL_ASSISTANT_HOME_ROUTE,
  LEGAL_CONTRACT_REVIEW_ROUTE,
  LEGAL_CONTRACT_REVIEW_DETAIL_ROUTE,
  LEGAL_SMART_ARCHIVE_ROUTE,
} from '../router/paths'

export type LegalWorkspaceNavSection = 'primary' | 'tools' | 'resources'

export interface LegalWorkspaceNavItem {
  id: string
  labelKey: string
  icon: string
  section: LegalWorkspaceNavSection
  destination?: RouteLocationRaw
  activeRouteNames?: string[]
  disabled?: boolean
  badgeKey?: string
}

export const LEGAL_WORKSPACE_NAV_ITEMS: readonly LegalWorkspaceNavItem[] = [
  {
    id: 'new-chat',
    labelKey: 'legalWorkspace.newChat',
    icon: 'add',
    section: 'primary',
    destination: { name: LEGAL_ASSISTANT_HOME_ROUTE },
  },
  {
    id: 'ai-assistant',
    labelKey: 'legalWorkspace.aiAssistant',
    icon: 'chat',
    section: 'tools',
    destination: { name: LEGAL_ASSISTANT_HOME_ROUTE },
    activeRouteNames: [LEGAL_ASSISTANT_HOME_ROUTE, LEGAL_ASSISTANT_CHAT_ROUTE],
  },
  {
    id: 'contract-review',
    labelKey: 'legalWorkspace.contractReview',
    icon: 'file-paste',
    section: 'tools',
    destination: { name: LEGAL_CONTRACT_REVIEW_ROUTE },
    activeRouteNames: [LEGAL_CONTRACT_REVIEW_ROUTE, LEGAL_CONTRACT_REVIEW_DETAIL_ROUTE],
  },
  {
    id: 'smart-archive',
    labelKey: 'legalWorkspace.smartArchive',
    icon: 'folder-open',
    section: 'tools',
    destination: { name: LEGAL_SMART_ARCHIVE_ROUTE },
    activeRouteNames: [LEGAL_SMART_ARCHIVE_ROUTE],
  },
  {
    id: 'legal-research',
    labelKey: 'legalWorkspace.legalResearch',
    icon: 'search',
    section: 'tools',
    disabled: true,
    badgeKey: 'legalWorkspace.comingSoon',
  },
  {
    id: 'drafting',
    labelKey: 'legalWorkspace.drafting',
    icon: 'edit-1',
    section: 'tools',
    disabled: true,
    badgeKey: 'legalWorkspace.comingSoon',
  },
  {
    id: 'due-diligence',
    labelKey: 'legalWorkspace.dueDiligence',
    icon: 'task-checked',
    section: 'tools',
    disabled: true,
    badgeKey: 'legalWorkspace.comingSoon',
  },
  {
    id: 'platform-console',
    labelKey: 'legalWorkspace.platformConsole',
    icon: 'view-module',
    section: 'resources',
    destination: '/platform/knowledge-bases',
  },
  {
    id: 'knowledge-bases',
    labelKey: 'legalWorkspace.knowledgeBases',
    icon: 'book',
    section: 'resources',
    destination: '/platform/knowledge-bases',
  },
  {
    id: 'agents',
    labelKey: 'legalWorkspace.agents',
    icon: 'usergroup',
    section: 'resources',
    destination: '/platform/agents',
  },
  {
    id: 'settings',
    labelKey: 'legalWorkspace.settings',
    icon: 'setting',
    section: 'resources',
    destination: '/platform/settings',
  },
]

export function legalWorkspaceItemsFor(section: LegalWorkspaceNavSection): readonly LegalWorkspaceNavItem[] {
  return LEGAL_WORKSPACE_NAV_ITEMS.filter((item) => item.section === section)
}

export function isLegalWorkspaceItemActive(item: LegalWorkspaceNavItem, routeName: unknown): boolean {
  if (!item.activeRouteNames || typeof routeName !== 'string') return false
  return item.activeRouteNames.includes(routeName)
}
