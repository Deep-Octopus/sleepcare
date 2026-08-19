import assert from 'node:assert/strict'
import test from 'node:test'
import { resolvePostLoginRoute } from './login-navigation.js'

test('登录后始终进入当前账号有权访问的默认首页', () => {
  const routes = [
    {
      name: 'layout',
      children: [
        {
          name: 'CareContent',
          children: [
            { name: 'CareQuestionnaires' }
          ]
        }
      ]
    }
  ]

  const result = resolvePostLoginRoute({
    defaultRouter: 'CareQuestionnaires',
    previousPath: '/layout/sleep-care/care-clients',
    routes
  })

  assert.deepEqual(result, { name: 'CareQuestionnaires' })
})

test('默认首页不在当前账号菜单时不沿用旧页面', () => {
  const result = resolvePostLoginRoute({
    defaultRouter: 'CareQuestionnaires',
    previousPath: '/layout/sleep-care/care-clients',
    routes: [{ name: 'layout', children: [{ name: 'CarePlans' }] }]
  })

  assert.equal(result, null)
})
