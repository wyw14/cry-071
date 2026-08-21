<script setup lang="ts">
import { reactive } from 'vue'
const form = reactive({ title: '', area_id: 'area-central-park', content: '', feedback_ids: '' })
</script>
<template><form class="form-grid"><label class="span-2">公告标题<input v-model="form.title" /></label><label>区域<select v-model="form.area_id"><option value="area-central-park">中心公园</option><option value="area-riverside">滨河步道</option></select></label><label>关联反馈<input v-model="form.feedback_ids" placeholder="受理反馈 ID" /></label><label class="span-2">解决情况<textarea v-model="form.content" rows="6" /></label><div class="actions span-2"><button>保存草稿</button><button class="primary">发布公告</button></div></form></template>
