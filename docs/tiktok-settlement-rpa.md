# TikTok 结算单 RPA 导入

接口：`POST /api/internal/tax-data/tiktok/settlements/upload`

- 请求头：`X-RPA-Key`，与 Walmart / TK 其他导入接口使用相同密钥。
- 请求体：`multipart/form-data`，文件字段名 `file`，无需 `store_id`。
- 支持 CSV、XLSX、XLS，Excel 仅读取第一张工作表，最大 20MB、100000 行。
- 按英文表头名称映射，不依赖列顺序。支持用户提供的 81 个字段；至少需要 Statement Date、Statement ID、Currency、Type、Order/adjustment ID、Total settlement amount 表头，其余缺失或空白字段保留为空。
- 月份、季度按 Statement Date 计算。费用 = Total settlement amount × 声明日期当天对应币种兑人民币汇率，四舍五入到两位小数；CNY 直接使用原金额。
- 没有汇率也能导入，费用显示“待换算”；补齐汇率后查询自动使用数据库最新值。汇率补齐日期范围包含结算单最早日期。
- 物流方式、订单类型由订单表匹配：Order/adjustment ID 对应 Order ID；订单类型取 Normal or Pre-order，物流方式按 [订单匹配规则](tiktok-order-rpa.md) 生成。已有结算单自动生效。
- 同一文件内容重复上传（包括并发上传）不重复入库，改名不影响此判断。跨文件按 Statement Date + Statement ID + Order/adjustment ID 联合去重，详见下方规则。
- 每次已读取文件的导入尝试保存原文件、导入时间、入库／跳过条数、状态；导入者固定 RPA。

```bash
curl -X POST "https://你的API域名/api/internal/tax-data/tiktok/settlements/upload" \
  -H "X-RPA-Key: 现有RPA密钥" \
  -F "file=@结算单.xlsx"
```

响应保持 `{ "status": true, "msg": "结算单导入成功，入库 100 条，跳过重复 5 条" }` 或 `{ "status": false, "msg": "具体失败原因" }`。

登录后的页面接口沿用报税数据访问权限：

- `GET /api/tax-data/tiktok/settlements?month=2026-01&quarter=2026-Q1&page=1&page_size=50`
- `GET /api/tax-data/tiktok/filter-options?type=settlements&field=month&quarter=2026-Q1`
- `GET /api/tax-data/tiktok/export?type=settlements&month=2026-01`
- 导入和下载日志沿用 `/api/tax-data/imports`、`/api/tax-data/downloads`，传 `platform=tiktok`。

列表按声明日期、记录 ID 稳定排序，返回 `items`、筛选后的 `total`、包含英文键和中文名称的 `columns`。页面及导出表头使用“英文(中文)”格式，例如 `Statement Date(声明日期)`。月份／季度筛选作用于全部数据，导出不受当前分页限制，下载记录保存当次报表快照。

81 列按英文语义逐项翻译，映射见 `backend/internal/handler/tiktok_settlement_columns.json`。原提供的中文列表与英文数量、顺序不同，因此未按位置硬套；没有英文对应项的“相关订单ID”“额外信息”等不额外创建。结算单仅提供季度汇总的物流费用明细，不改变取自 FBT 表的配送费总额：MCF 取 Seller shipping fee，FBT 取 FBT fulfillment fee，海外仓取 TikTok Shop shipping fee。其他结算字段不重复计入涉税收入。

## 导入去重

按 `Statement Date + Statement ID + Order/adjustment ID` 三项联合去重。声明日期统一为 YYYY-MM-DD，两个 ID 去掉首尾空白，保留前导零；空 ID 也作为组合的一部分参与比较。任一字段不同就视为新记录。金额、SKU、Type 不参与去重。

同一文件内保留首条，跨文件与已有记录重复时跳过、不覆盖原金额。相同文件仍可直接跳过；不同文件的并发导入通过数据库事务锁串行去重。日志记录原始行数、实际导入数和跳过数，接口消息包含“入库 N 条，跳过重复 M 条”。历史已存在的重复记录不自动删除。
