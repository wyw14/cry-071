<script setup lang="ts">
import { reactive, ref } from 'vue'
import { Send } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Feedback } from '@/api/types'
import { useWorkspaceStore } from '@/stores/workspace'

const store = useWorkspaceStore()
const form = reactive({ area_id: 'area-central-park', facility_category_id: 'lighting', subject_code: 'damaged', priority: 'normal', title: '', description: '', location: '', anonymous: false, submitter_name: '', submitter_contact: '' })
const result = ref<{ feedback: Feedback; query_token: string; duplicate_candidates: Array<{ feedback_id: string; title: string; similarity_score: number }> }>()
const pending = ref(false)
async function submit() {
  pending.value = true
  try {
    const submitted = await api<{ feedback: Feedback; query_token: string; duplicate_candidates: Array<{ feedback_id: string; title: string; similarity_score: number }> }>('/feedback', { method: 'POST', body: JSON.stringify(form) })
    result.value = submitted
    store.rememberToken(submitted.query_token)
  }
  finally { pending.value = false }
}
</script>
<template>
  <form class="form-grid" @submit.prevent="submit">
    <label>公共区域<select v-model="form.area_id"><option value="area-central-park">中心公园</option><option value="area-riverside">滨河步道</option></select></label>
    <label>设施类别<select v-model="form.facility_category_id"><option value="lighting">照明设施</option><option value="sanitation">环卫设施</option><option value="accessibility">无障碍设施</option></select></label>
    <label>反馈主题<select v-model="form.subject_code"><option value="damaged">设施损坏</option><option value="cleanliness">环境卫生</option><option value="safety">安全隐患</option></select></label>
    <label>优先级<select v-model="form.priority"><option value="low">低</option><option value="normal">一般</option><option value="high">高</option><option value="critical">紧急</option></select></label>
    <label class="span-2">标题<input v-model="form.title" required minlength="5" maxlength="120" /></label>
    <label class="span-2">位置说明<input v-model="form.location" required maxlength="300" /></label>
    <label class="span-2">详细描述<textarea v-model="form.description" required minlength="10" maxlength="4000" rows="5" /></label>
    <label>姓名<input v-model="form.submitter_name" :disabled="form.anonymous" /></label>
    <label>联系方式<input v-model="form.submitter_contact" /></label>
    <label class="check span-2"><input v-model="form.anonymous" type="checkbox" />匿名提交</label>
    <div class="actions span-2"><button class="primary" :disabled="pending"><Send :size="17" />{{ pending ? '正在提交' : '提交反馈' }}</button></div>
  </form>
  <section v-if="result" class="result-band"><strong>受理编号：{{ result.feedback.AcceptanceNumber }}</strong><span>查询令牌已保存在本机浏览器。</span><p v-if="result.duplicate_candidates.length">发现 {{ result.duplicate_candidates.length }} 条可能相似反馈，请在重复提交前核对。</p></section>
</template>
