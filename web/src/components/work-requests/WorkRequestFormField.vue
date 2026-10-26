<template>
  <div class="form-field">
    <div class="field-label">
      <label :for="fieldId || undefined">{{ label }}<span v-if="required">*</span></label>
      <small v-if="hint">{{ hint }}</small>
    </div>
    <slot />
    <p v-if="error" class="field-error">{{ error }}</p>
  </div>
</template>

<script setup>
defineProps({ label: { type: String, required: true }, fieldId: { type: String, default: '' }, required: { type: Boolean, default: false }, hint: { type: String, default: '' }, error: { type: String, default: '' } })
</script>

<style scoped>
.form-field { display: flex; min-width: 0; flex-direction: column; gap: 7px; }
.field-label { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; }
.field-label label { color: #404040; font-size: 13px; font-weight: 700; }
.field-label label span { margin-left: 3px; color: #dc2626; }
.field-label small { color: #a3a3a3; font-size: 11px; text-align: right; }
.field-error { margin: 0; color: #dc2626; font-size: 12px; }
</style>
