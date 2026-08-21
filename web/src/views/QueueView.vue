<script setup lang="ts">
import { onMounted } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Feedback } from '@/api/types'
import { useWorkspaceStore } from '@/stores/workspace'
const store=useWorkspaceStore()
async function load(){store.loading=true;try{const query=new URLSearchParams();Object.entries(store.filters).forEach(([key,value])=>{if(value)query.set(key,String(value))});const page=await api<{Items:Feedback[]}>(`/admin/queue?${query}`);store.queue=page.Items??[]}finally{store.loading=false}}
onMounted(load)
</script>
<template>
  <div class="toolbar"><select v-model="store.filters.status" @change="load"><option value="">全部状态</option><option value="pending_acceptance">待受理</option><option value="needs_information">待补充</option><option value="in_progress">处理中</option><option value="pending_confirmation">待确认</option></select><select v-model="store.filters.priority" @change="load"><option value="">全部优先级</option><option value="critical">紧急</option><option value="high">高</option><option value="normal">一般</option><option value="low">低</option></select><label class="check"><input v-model="store.filters.overdue" type="checkbox" @change="load" />只看超期</label><button class="icon" title="刷新队列" @click="load"><RefreshCw :size="18" /></button></div>
  <div class="table-wrap"><table><thead><tr><th></th><th>受理编号</th><th>问题</th><th>状态</th><th>优先级</th><th>负责人</th><th>截止时间</th></tr></thead><tbody><tr v-for="item in store.queue" :key="item.ID"><td><input type="checkbox" :checked="store.selected.has(item.ID)" @change="store.toggle(item.ID)" /></td><td class="mono"><RouterLink :to="`/feedback/${item.ID}`">{{ item.AcceptanceNumber }}</RouterLink></td><td>{{ item.Title }}<small>{{ item.Location }}</small></td><td><span class="status">{{ item.Status }}</span></td><td>{{ item.Priority }}</td><td>{{ item.AssigneeID || '未分派' }}</td><td>{{ new Date(item.DueAt).toLocaleString('zh-CN') }}</td></tr></tbody></table></div>
  <p v-if="!store.loading && !store.queue.length" class="empty">当前筛选条件下没有反馈。</p>
</template>
