<template>
  <section class="contract-generation" data-testid="contract-generation">
    <div class="contract-generation__content">
      <span class="eyebrow">{{ t('contractGeneration.eyebrow') }}</span>
      <h1>{{ t('contractGeneration.title') }}</h1>
      <p class="contract-generation__subtitle">{{ t('contractGeneration.subtitle') }}</p>

      <article class="contract-generation__card">
        <div class="contract-generation__icon" aria-hidden="true">
          <t-icon name="file-add" size="28px" />
        </div>
        <div>
          <span class="contract-generation__status">{{ t('contractGeneration.status') }}</span>
          <h2>{{ t('contractGeneration.cardTitle') }}</h2>
          <p>{{ t('contractGeneration.cardDescription') }}</p>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<style scoped lang="less">
.contract-generation {
  width: 100%;
  height: 100%;
  overflow: auto;
  box-sizing: border-box;
  padding: 56px 42px;
  color: var(--legal-text-primary);
  background: var(--legal-bg-page);
}

.contract-generation__content {
  width: min(760px, 100%);
  margin: 0 auto;
}

.eyebrow {
  color: var(--legal-text-secondary);
  font-size: 10px;
  letter-spacing: .12em;
  text-transform: uppercase;
}

h1 {
  margin: 8px 0 6px;
  font-size: 28px;
  letter-spacing: -.03em;
}

p {
  margin: 0;
  color: var(--legal-text-secondary);
  font-size: 13px;
  line-height: 1.65;
}

.contract-generation__subtitle {
  max-width: 580px;
}

.contract-generation__card {
  margin-top: 28px;
  padding: 24px;
  display: flex;
  align-items: flex-start;
  gap: 16px;
  border: 1px solid var(--legal-border);
  border-radius: 8px;
  background: var(--legal-bg-surface);
}

.contract-generation__icon {
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--legal-ai-strong);
  background: var(--legal-ai-soft);
}

.contract-generation__status {
  display: inline-flex;
  padding: 4px 8px;
  border-radius: 12px;
  color: var(--legal-warning-strong);
  background: var(--legal-warning-soft);
  font-size: 10px;
  font-weight: 650;
}

h2 {
  margin: 10px 0 6px;
  font-size: 17px;
}

@media (max-width: 620px) {
  .contract-generation { padding: 32px 20px; }
  .contract-generation__card { padding: 18px; }
}
</style>
