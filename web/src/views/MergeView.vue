<script setup lang="ts">
import { ref } from 'vue'
const primary = ref('')
const members = ref('')
const reason = ref('')
</script>
<template><form class="form-grid"><label>主反馈<input v-model="primary" placeholder="反馈 ID" /></label><label>待合并反馈<input v-model="members" placeholder="多个 ID 用逗号分隔" /></label><label class="span-2">合并原因<textarea v-model="reason" rows="4" /></label><div class="actions span-2"><button class="primary">确认合并</button></div></form><section class="notice">合并后，成员反馈保留原受理编号和时间线，并链接到主反馈统一处理。</section></template>
