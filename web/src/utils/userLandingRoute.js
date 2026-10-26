export function isCustomerServiceUser(user) {
  return [user?.sub_department, user?.department].some(value => {
    const name = String(value || '').trim().replace(/部$/, '')
    return name === '海外客服' || name === '客服'
  })
}

export function userLandingRoute(user) {
  if (isFinanceUser(user)) return '/tax-data'
  return isCustomerServiceUser(user) ? '/daily-call-registration' : '/generate'
}

export function isTaxDataUser(user) {
  const isAdmin = (user?.roles || []).some(role => role?.name === 'admin' || role === 'admin')
  const departmentText = [user?.department, user?.sub_department]
    .filter(Boolean).join(' ')
  return isAdmin || departmentText.includes('财务')
}

export function canViewOperationsTools(user) {
  const departmentText = [user?.department, user?.sub_department]
    .filter(Boolean).join(' ')
  return !departmentText.includes('设计') && !departmentText.includes('财务')
}

export function isFinanceUser(user) {
  return [user?.department, user?.sub_department]
    .filter(Boolean).join(' ').includes('财务')
}

export function isDesignUser(user) {
  return [user?.department, user?.sub_department]
    .filter(Boolean).join(' ').includes('设计')
}
