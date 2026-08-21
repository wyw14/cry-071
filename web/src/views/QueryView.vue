<script setup lang="ts">
import { ref } from 'vue'
import { Search } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { PublicFeedbackView } from '@/api/types'
import { useWorkspaceStore } from '@/stores/workspace'
const store=useWorkspaceStore();const token=ref(store.queryToken);const result=ref<PublicFeedbackView>();const error=ref('')
async function query(){error.value='';try{result.value=await api(`/feedback/query/${encodeURIComponent(token.value)}`);store.rememberToken(token.value)}catch(err){error.value=err instanceof Error?err.message:'查询失败'}}
</script>
<template>
  <div class="query-bar"><input v-model="token" placeholder="输入查询令牌" /><button class="primary" @click="query"><Search :size="17" />查询</button></div>
  <p v-if="error" class="error">{{ error }}</p>
  <template v-if="result">
    <section class="detail-header"><div><span class="mono">{{ result.feedback.AcceptanceNumber }}</span><h2>{{ result.feedback.Title }}</h2></div><span class="status">{{ result.feedback.Status }}</span></section>
    <dl class="facts"><div><dt>位置</dt><dd>{{ result.feedback.Location }}</dd></div><div><dt>优先级</dt><dd>{{ result.feedback.Priority }}</dd></div><div><dt>处理人</dt><dd>{{ result.feedback.AssigneeID || '待分派' }}</dd></div></dl>
    <ol class="timeline"><li v-for="event in result.timeline" :key="event.ID"><time>{{ new Date(event.OccurredAt).toLocaleString('zh-CN') }}</time><strong>{{ event.Summary }}</strong></li></ol>
  </template>
</template>
