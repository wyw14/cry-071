import { createRouter, createWebHistory } from 'vue-router'
import SubmitView from './views/SubmitView.vue'
import QueryView from './views/QueryView.vue'
import QueueView from './views/QueueView.vue'
import FeedbackView from './views/FeedbackView.vue'
import MergeView from './views/MergeView.vue'
import AnnouncementView from './views/AnnouncementView.vue'
import ConfigView from './views/ConfigView.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/queue' },
    { path: '/submit', component: SubmitView, meta: { title: '提交反馈' } },
    { path: '/query', component: QueryView, meta: { title: '进度查询' } },
    { path: '/queue', component: QueueView, meta: { title: '处理队列' } },
    { path: '/feedback/:id', component: FeedbackView, meta: { title: '反馈详情' } },
    { path: '/merge', component: MergeView, meta: { title: '合并处理' } },
    { path: '/announcements', component: AnnouncementView, meta: { title: '解决公告' } },
    { path: '/config', component: ConfigView, meta: { title: '业务配置' } }
  ]
})
