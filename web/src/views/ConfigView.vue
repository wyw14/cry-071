<script setup lang="ts">
import { reactive } from 'vue'
const settings = reactive({ normal: 48, high: 12, critical: 4, maxFile: 10, anonymous: true })
</script>
<template><form class="settings"><section><h2>超期规则</h2><label>一般反馈（小时）<input v-model.number="settings.normal" type="number" min="1" /></label><label>高优先级（小时）<input v-model.number="settings.high" type="number" min="1" /></label><label>紧急反馈（小时）<input v-model.number="settings.critical" type="number" min="1" /></label></section><section><h2>提交与附件</h2><label>单个附件上限（MB）<input v-model.number="settings.maxFile" type="number" min="1" max="10" /></label><label class="check"><input v-model="settings.anonymous" type="checkbox" />允许匿名提交</label></section><div class="actions"><button class="primary">保存配置</button></div></form></template>
