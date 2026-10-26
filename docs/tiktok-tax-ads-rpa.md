# TikTok 广告费 RPA 导入

## 请求

`POST /api/internal/tax-data/tiktok/ads/upload`

请求头 `X-RPA-Key` 与 Walmart、TikTok 涉税申报明细导入共用同一个值。
使用 `multipart/form-data`，只需要 `file` 文件字段，不需要 `store_id`。

```powershell
curl.exe -X POST "https://你的系统域名/api/internal/tax-data/tiktok/ads/upload" -H "X-RPA-Key: 与Walmart相同的密钥" -F "file=@D:/RPA/tiktok-ads.xlsx"
```

支持 CSV、XLSX、XLS。广告费文件默认读取第一张工作表，第一行是表头；TikTok 涉税申报明细仍默认读取第二张工作表。最多 20MB、100000 条记录。

| 日期 | 系列名称 | 推广系列ID | 现金消耗 | 信用额度消耗 | 广告费用赠款消耗 | 金额 | 币种 | 类型 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 2026-01-02 | 示例系列 | 00123456789 | 10 | 20 | 5 | 35 | USD | 消耗 |

表头按名称匹配，顺序可调整。推广系列ID建议使用文本单元格，以保留前导零及长编号。币种为 USD，空值按 USD；金额空值按 0，整行空白跳过。末尾无有效日期且标记为“总计”的汇总行整行跳过，不计入明细或季度汇总。日期支持常见文本格式及 Excel 日期单元格。

导入不要求已有汇率：缺少当日汇率时先保存美元原始金额，人民币费用显示“待换算”。之后在汇率管理点击“补齐缺失汇率”，无需重新导入即可换算。系统先检查最早账单日期至今的全部日期及 USD、EUR、CAD、GBP，只请求缺失或无效的日期／币种组合，已有有效汇率不重复获取、不覆盖。汇率获取范围包含 TikTok 广告费和申报明细的最早日期。日期或金额错误时整次导入不入库。重复上传相同文件会成功返回“文件已导入”，不重复增加金额。同一文件内、不同文件之间按“日期＋推广系列ID”去重，保留首次导入记录，后续重复跳过，不覆盖名称或金额。日期先统一为 YYYY-MM-DD，ID 去除前后空格后比较。

返回示例：

```json
{"status": true, "msg": "广告费导入成功，入库 3 条，跳过重复 2 条"}
```

```json
{"status": false, "msg": "第 2 行信用额度消耗无效"}
```

## 展示和计算

- 月份、季度由日期自动生成；费用 = 金额 × 对应日期汇率，显示人民币。
- 广告费表中的现金消耗、信用额度消耗、广告费用赠款消耗、金额均为美元。
- 季度广告费：美元 = 信用额度消耗之和；人民币 = 每条信用额度消耗 × 对应日期汇率，再求和。
- TikTok 报税收入 = 收入费 − 退款费，不扣平台佣金或广告费。
- 季度数据先汇总精确金额，再四舍五入保留两位小数。缺少汇率的人民币字段返回 null、页面显示“待换算”，不会显示部分合计或误显示为 0；美元汇总不受影响。
- 列表和汇总优先读取汇率管理中的当日汇率，缺少时使用导入时保存的汇率。只含广告费的季度也会展示。

查询接口（需要登录，沿用报税数据权限）：

- `GET /api/tax-data/tiktok/ads`
- `GET /api/tax-data/tiktok/quarters?currency=CNY`，美元传 `USD`。

后端启动自动创建 `tik_tok_tax_ad_batches`（原文件和导入批次）、`tik_tok_tax_ads`（广告费明细）两张表。更新前端和后端后生效，已有 TikTok 申报明细无需重新导入。

## TikTok 涉税申报明细去重

`POST /api/internal/tax-data/tiktok/upload` 的涉税申报明细按业务单号（订单ID、调整单ID）去重，汇总表按年份-季度去重。

业务单号去除前后空格，相同业务单号即使业务时间或金额不同也跳过，保留首次导入值。同一文件内、跨文件及并发上传均适用。空业务单号没有可用去重键，按原始行保留。广告费继续按“日期＋推广系列ID”去重。

两个接口仍返回 `status`、`msg`，消息包含入库数和跳过重复数。新文件即使全部是重复记录，也正常返回成功。历史已存在的重复数据不会自动删除。

## 导入日志、下载记录和表头筛选

TikTok 页面支持查看导入日志及下载记录，与 Walmart 日志隔离。申报明细、广告费、季度汇总均支持下载 CSV 报表。

TikTok 导入日志的导入者固定为 `RPA`，无需传入 `importer`；即使传入也不影响记录。

每次已读取文件的导入尝试都会记录报表类型、操作人、时间、原文件、来源条数、入库条数、重复跳过条数及成功／重复／失败状态。历史导入会自动补建日志，已有 TikTok 导入记录的导入者也统一为 `RPA`。

登录后的查询和下载接口（沿用报税数据访问权限）：

明细与广告费列表支持 `page`、`page_size`，默认第 1 页、每页 50 条，单页最多 200 条。返回的 `total` 为筛选后全部记录数。前端支持每页 25、50、100、200 条，两张表分别保留分页和筛选状态；下载报表不受分页限制。

- `GET /api/tax-data/imports?platform=tiktok&page=1&page_size=20`
- `GET /api/tax-data/imports/:id/file?platform=tiktok`
- `GET /api/tax-data/downloads?platform=tiktok&page=1&page_size=20`
- `GET /api/tax-data/downloads/:id/file?platform=tiktok`
- `GET /api/tax-data/tiktok/details?month=2026-01&quarter=2026-Q1`
- `GET /api/tax-data/tiktok/ads?month=2026-01&quarter=2026-Q1`
- `GET /api/tax-data/tiktok/fbt?month=2026-01&quarter=2026-Q1&page=1&page_size=50`
- `GET /api/tax-data/tiktok/filter-options?type=ads&field=month&quarter=2026-Q1`
- `GET /api/tax-data/tiktok/export?type=ads&month=2026-01&quarter=2026-Q1`

`type` 支持 `details`、`ads`、`fbt`、`quarters`。FBT 导入接口为 `/api/internal/tax-data/tiktok/fbt/upload`，月份从文件名开头的 `YYYY-MM` 提取，季度由系统计算。筛选候选值接口支持前三种明细报表，`field` 为 `month` 或 `quarter`；查询候选值时保留另一列筛选，不受当前列选择限制。月份和季度按 AND 组合查询完整数据库数据。

季度汇总导出使用 `currency=USD` 或 `currency=CNY`。明细导出包含当前筛选的所有行，缺少汇率的广告费人民币金额导出为“待换算”。下载日志记录下载用户、时间、报表类型、筛选范围、币种和条数，并保存该次文件快照；再次下载历史记录不重新计算旧文件。
