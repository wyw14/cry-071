<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { ClipboardList, FileSearch, GitMerge, Megaphone, PlusCircle, Settings, MapPinned } from 'lucide-vue-next'

const route = useRoute()
const title = computed(() => String(route.meta.title ?? '公共空间反馈'))
const navigation = [
  { to: '/queue', label: '处理队列', icon: ClipboardList },
  { to: '/submit', label: '提交反馈', icon: PlusCircle },
  { to: '/query', label: '进度查询', icon: FileSearch },
  { to: '/merge', label: '合并处理', icon: GitMerge },
  { to: '/announcements', label: '解决公告', icon: Megaphone },
  { to: '/config', label: '业务配置', icon: Settings }
]
</script>

<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand"><MapPinned :size="22" /><span>公共空间反馈</span></div>
      <nav aria-label="主导航">
        <RouterLink v-for="item in navigation" :key="item.to" :to="item.to">
          <component :is="item.icon" :size="18" /><span>{{ item.label }}</span>
        </RouterLink>
      </nav>
    </aside>
    <main>
      <header class="topbar"><h1>{{ title }}</h1><span class="environment">本地离线环境</span></header>
      <section class="content"><RouterView /></section>
    </main>
  </div>
</template>
