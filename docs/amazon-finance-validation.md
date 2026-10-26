# Amazon 财务明细与 PDF 联合导入

POST /api/internal/tax-data/amazon/finance/upload，沿用 X-RPA-Key。

multipart 字段 file 为 CSV/XLSX 明细，summary_file 为汇总 PDF，各不超过 20MB。

从 PDF 内容识别日期范围和 USD 金额。导入只检查 PDF 开始、结束日期是否与已导入 PDF 相同；重复范围拒绝导入。金额差异、明细日期不阻止导入。明细按原规则跳过重复行；即使所有明细重复，未重复的 PDF 仍保存到 finance_summary 导入日志。PDF 和新增明细在同一事务中保存。

季度汇总按每份 PDF 的完整日期范围查询数据库明细，并使用季度汇总公式（美元）校验：税费对应 Amazon Obligated Tax and Regulatory Fee Withheld；仓储费对应 FBA inventory and inbound services fees；配送费对应 FBA transaction fees + FBA transaction fee refunds。差异显示在对应季度，列出明细值、PDF 值和差额，不覆盖季度金额。

税费汇总所有 marketplace withheld tax，不筛选 type。配送费统计 Order、Refund 的 fba fees 加 FBA Transaction fees / FBA Inbound Placement Service Fee 的 fba fees；销售费统计 Order、Refund 的 selling fees。

PDF 解析复用 CUSTOMS_PYTHON（备用 STP_WORKER_PYTHON、默认 python3）及 pdfplumber，Docker 已包含依赖。支持可提取文本的官方 USD 报告。
