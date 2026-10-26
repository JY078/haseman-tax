export function dailyCallLocale(user) {
  for (const value of [user?.sub_department, user?.department]) {
    const name = String(value || '').trim().replace(/部$/, '')
    if (name === '海外客服') return 'en'
    if (name === '客服') return 'zh'
  }
  return 'zh'
}

const english = {
  '页码': 'Page number', '跳转': 'Go', '每页': 'Per page', '每页条数': 'Rows per page',
  '清除搜索': 'Clear search', '导出全部筛选结果': 'Export all matching records',
  '导出报表': 'Export report', '正在导出...': 'Exporting...', '没有可导出的记录': 'No records to export', '报表已导出': 'Report exported', '导出失败，请重试': 'Export failed. Please try again.', '需要客户提供的信息': 'Information requested from customers', '外呼结果': 'Result',
  '某天': 'Day', '某月': 'Month', '某年': 'Year', '时间范围': 'Time range', '选择时间': 'Select time', '清除时间筛选': 'Clear time filter',
  '问题类型': 'Issue type',
  '复制全文': 'Copy full text', '全文已复制': 'Full text copied', '复制失败，请选择文字后复制': 'Unable to copy. Select the text to copy it.',
  '查看全文': 'Read full issue', '问题描述全文': 'Full issue description',
  '请选择': 'Select an option',
  '每日来电登记': 'Daily Call Registration',
  '新增登记': 'New registration', '质检状态': 'Review status', '搜索登记': 'Search registrations',
  '搜索姓名、订单、邮箱或电话': 'Search name, order, email or phone', '搜索': 'Search', '刷新': 'Refresh',
  '暂无访问权限': 'Access denied', '暂时无法加载登记': 'Unable to load registrations', '重新加载': 'Reload',
  '来电记录': 'Call records', '最新登记优先': 'Newest first', '来电登记表': 'Inbound call log', '主动外呼表': 'Proactive outbound calls', '表类型': 'Table type', '时间范围': 'Time range', '某天': 'Day', '某月': 'Month', '某年': 'Year', '选择时间': 'Choose date', '清除时间筛选': 'Clear date filter',
  '查看填写人员': 'View contributors', '待质检': 'Pending review', '没有符合条件的登记': 'No matching registrations',
  '暂无来电登记': 'No call registrations yet', '上一页': 'Previous page', '下一页': 'Next page',
  '新登记': 'NEW REGISTRATION', '来电': 'CALL', '新增来电登记': 'New call registration', '来电详情': 'Call details',
  '关闭': 'Close', '登记详情': 'Registration details', '填写人员': 'Contributors', '正在加载登记...': 'Loading registration...',
  '重试': 'Retry', '首次登记': 'Registered by', '登记信息 · 最近填写': 'Registration · Last updated by',
  '质检信息 · 最近填写': 'Review · Last updated by', '待客服填写': 'Awaiting customer service review',
  '来电登记': 'Call registration', '编辑登记信息': 'Edit registration', '由海外客服填写': 'Filled in by overseas customer service',
  '保存时自动记录': 'Recorded automatically on save', '取消': 'Cancel', '保存中...': 'Saving...',
  '创建登记': 'Create registration', '保存登记': 'Save registration', '质检记录': 'Quality review',
  '填写质检结果': 'Edit quality review', '由客服填写': 'Filled in by customer service', '保存质检': 'Save review',
  '全部记录': 'All records', '已质检': 'Reviewed', '放弃尚未保存的修改？': 'Discard unsaved changes?',
  '质检已保存': 'Review saved', '登记已保存': 'Registration saved', '保存失败，请重试': 'Unable to save. Please try again.',
  '请检查网络连接后重试': 'Check your connection and try again.', '加载登记详情失败': 'Unable to load registration details.',
  '日期': 'Date', '姓名': 'Name', '订单号': 'Order Number', '邮箱': 'Email', '电话号码': 'Phone no',
  '购买渠道': 'Bought in', '国家': 'Country', '问题简述': 'Brief of the issue', '质检结果': 'Quality result', '理由': 'Reason',
  '请填写此字段': 'Please fill out this field.', '请输入有效的邮箱地址': 'Please enter a valid email address.',
  '请检查填写内容': 'Please check this value.',
  '请先登录': 'Please sign in.', '当前账号不可用': 'This account is unavailable.',
  '仅海外客服、客服及管理员可访问来电登记': 'Only overseas customer service, customer service and administrators can access call registrations.',
  '请求包含无效字段或格式': 'The request contains invalid fields or formatting.',
  '获取来电登记失败': 'Unable to load call registrations.', '无效的登记 ID': 'Invalid registration ID.',
  '登记不存在': 'Registration not found.', '获取登记失败': 'Unable to load registration.', '新增登记失败': 'Unable to create registration.',
  '请填写质检结果和理由，并检查内容长度': 'Enter a quality result and reason, and check the text length.',
  '请填写 Name 和 Brief of the issue，并检查邮箱格式和内容长度': 'Enter a name and issue summary, and check the email format and text length.',
  '当前部门无权修改这部分信息': 'Your department cannot edit this section.', '保存登记失败': 'Unable to save registration.',
  '内部工作台': 'Workspace', '图像创作': 'Image tools', '运营工具': 'Operations', '产品库': 'Product library',
  '个人中心': 'Profile', '历史': 'History', '作品': 'My work', '意见反馈': 'Feedback', '退出登录': 'Sign out',
  '登录': 'Sign in', '关闭导航': 'Close navigation', '打开导航': 'Open navigation',
  '核心生成': 'Generation', '图像处理': 'Image editing', '图像资源': 'Resources', '功能': 'Tools',
  '专业模式': 'Professional mode', '场景图生成': 'Scene generation', 'AI 图像生成': 'AI image generation',
  '智能生成': 'Smart generation', '多角度生成': 'Multiple angles', '景深图生成': 'Depth of field',
  '产品融合': 'Product blending', '图片文字翻译': 'Image translation', '图片转提示词': 'Image to prompt',
  '去背景': 'Remove background', '资产库': 'Asset library', 'AI图像标注': 'AI image metadata',
  '订单转 MCF': 'Order to MCF', '订单汇总': 'Order overview', 'Listing 关键词库': 'Listing keywords',
  '使用专业画布完成复杂出图流程': 'Create images with the professional canvas', '生成产品场景图': 'Generate product scenes',
  '通过描述或参考图自由生图': 'Create images from text or references', '智能生成listing和套图': 'Generate listings and image sets',
  '基于现有产品图补全更多展示角度': 'Generate additional product views', '生成主体清晰、背景虚化的景深图': 'Create a sharp subject with a soft background',
  '产品融合到场景中，光影自然真实': 'Blend products into scenes with natural lighting', '将图片中的文字翻译为目标语言': 'Translate text within images',
  '根据图片反推可复用提示词': 'Extract reusable prompts from images', '智能移除背景': 'Remove image backgrounds',
  '管理和复用图片与设计素材': 'Manage images and design assets', '批量处理 AI 生成的人物图像元数据': 'Process AI portrait metadata in batches',
  '将平台订单转换为 Amazon MCF 模板': 'Convert platform orders to Amazon MCF templates',
  '查看平台订单并回填物流追踪号': 'View orders and enter tracking numbers', '维护拓展、过滤、标记与入库流程': 'Manage keyword discovery, filtering and tagging',
}

export function dailyCallText(locale, text) {
  if (text === 'Daily Call Registration') return locale === 'en' ? text : '每日来电登记'
  return locale === 'en' ? english[text] || text : text
}

export function dailyCallError(locale, error, fallback) {
  const message = error?.response?.data?.error
  if (!message) return dailyCallText(locale, fallback)
  if (locale === 'en') return english[message] || (/[^\x00-\x7F]/.test(message) ? dailyCallText(locale, fallback) : message)
  if (message === '请填写 Name 和 Brief of the issue，并检查邮箱格式和内容长度') return '请填写姓名和问题简述，并检查邮箱格式和内容长度'
  return /[\u4e00-\u9fff]/.test(message) ? message : fallback
}
