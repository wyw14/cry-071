<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
const route=useRoute();const note=ref('');const publicReply=ref(true)
</script>
<template><section class="detail-header"><div><span class="mono">{{ route.params.id }}</span><h2>反馈处理详情</h2></div><div class="actions"><button>分派</button><button>变更状态</button></div></section><div class="split"><section><h3>处理时间线</h3><ol class="timeline"><li><time>等待加载</time><strong>请从处理队列进入真实记录</strong></li></ol></section><aside class="inspector"><h3>添加处理记录</h3><textarea v-model="note" rows="6" placeholder="填写回复或内部备注"/><label class="check"><input v-model="publicReply" type="checkbox" />提交者可见</label><button class="primary">保存记录</button></aside></div></template>
