import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/init',
    name: 'Init',
    component: () => import('@/view/init/index.vue')
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/view/login/index.vue')
  },
  {
    path: '/scanUpload',
    name: 'ScanUpload',
    meta: {
      title: '扫码上传',
      client: true
    },
    component: () => import('@/view/media/scanUpload.vue')
  },
  {
    path: '/client',
    component: () => import('@/view/client/layout.vue'),
    meta: {
      client: true
    },
    redirect: '/client/login',
    children: [
      {
        path: 'login',
        name: 'ClientLogin',
        component: () => import('@/view/client/login.vue'),
        meta: { title: '康养用户登录', client: true, clientChrome: false }
      },
      {
        path: 'access',
        name: 'ClientAccess',
        component: () => import('@/view/client/access.vue'),
        meta: { title: '访问任务', client: true, clientChrome: false }
      },
      {
        path: 'home',
        name: 'ClientHome',
        component: () => import('@/view/client/home.vue'),
        meta: { title: '我的康养服务', client: true, clientNav: 'home' }
      },
      {
        path: 'tasks',
        name: 'ClientTasks',
        component: () => import('@/view/client/tasks.vue'),
        meta: { title: '我的随访', client: true, clientNav: 'tasks' }
      },
      {
        path: 'tasks/:taskId',
        name: 'ClientTask',
        component: () => import('@/view/client/task.vue'),
        meta: { title: '任务说明', client: true }
      },
      {
        path: 'tasks/:taskId/fill',
        name: 'ClientTaskForm',
        component: () => import('@/view/client/form.vue'),
        meta: { title: '填写任务', client: true }
      },
      {
        path: 'tasks/:taskId/confirm',
        name: 'ClientTaskConfirm',
        component: () => import('@/view/client/confirm.vue'),
        meta: { title: '确认提交', client: true }
      },
      {
        path: 'tasks/:taskId/success',
        name: 'ClientTaskSuccess',
        component: () => import('@/view/client/success.vue'),
        meta: { title: '提交成功', client: true }
      },
      {
        path: 'consultations',
        name: 'ClientConsultations',
        component: () => import('@/view/client/consultations.vue'),
        meta: { title: '联系服务', client: true, clientNav: 'consultations' }
      },
      {
        path: 'consultations/new',
        name: 'ClientConsultationNew',
        component: () => import('@/view/client/consultation-new.vue'),
        meta: { title: '发起咨询', client: true }
      },
      {
        path: 'consultations/:id',
        name: 'ClientConsultationDetail',
        component: () => import('@/view/client/consultation-detail.vue'),
        meta: { title: '咨询详情', client: true }
      },
      {
        path: 'satisfaction',
        name: 'ClientSatisfaction',
        component: () => import('@/view/client/satisfaction.vue'),
        meta: { title: '服务评价', client: true, clientNav: 'satisfaction' }
      },
      {
        path: 'satisfaction/:id',
        name: 'ClientSatisfactionDetail',
        component: () => import('@/view/client/satisfaction-detail.vue'),
        meta: { title: '评价详情', client: true }
      }
    ]
  },
  {
    path: '/forceChangePassword',
    name: 'ForceChangePassword',
    component: () => import('@/view/system/security/forceChangePassword.vue'),
    meta: { title: '修改密码' }
  },
  {
    path: '/:catchAll(.*)',
    meta: {
      closeTab: true
    },
    component: () => import('@/view/error/index.vue')
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
