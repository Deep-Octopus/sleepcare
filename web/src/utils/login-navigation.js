const routeTreeContainsName = (routes, routeName) => routes.some((route) => {
  if (route?.name === routeName) {
    return true
  }

  return routeTreeContainsName(route?.children || [], routeName)
})

export const resolvePostLoginRoute = ({ defaultRouter, routes = [] } = {}) => {
  const routeName = typeof defaultRouter === 'string' ? defaultRouter.trim() : ''
  if (!routeName || !routeTreeContainsName(routes, routeName)) {
    return null
  }

  return { name: routeName }
}
