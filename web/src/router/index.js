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
    children: [
      {
        path: 'access',
        name: 'ClientAccess',
        component: () => import('@/view/client/access.vue'),
        meta: { title: '访问任务', client: true }
      },
      {
        path: 'tasks',
        name: 'ClientTasks',
        component: () => import('@/view/client/tasks.vue'),
        meta: { title: '我的任务', client: true }
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
