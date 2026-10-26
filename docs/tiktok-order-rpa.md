# TikTok 订单表与结算单匹配

订单表位于结算单和季度汇总之间，月份、季度按 `Created Time` 生成并固定在表格左侧。其余 63 列使用 `英文(中文)` 表头；长订单号、SKU、电话、邮编按字符串保存。

## RPA 导入

`POST /api/internal/tax-data/tiktok/orders/upload`

- 请求头 `X-RPA-Key`：沿用现有 Walmart / TikTok RPA 密钥。
- `multipart/form-data` 文件字段 `file`，不需要 `store_id`。
- 支持 CSV、XLSX、XLS；Excel 默认第一张工作表，最大 20MB、100000 行。
- 按英文表头映射，忽略表头首尾空白（包含 Virtual Bundle Seller SKU 前的空格）。
- 必需表头：Order ID、Created Time、Normal or Pre-order、Delivery Option Type、Delivery Option。其他字段可缺省或为空。
- 同文件重复导入跳过；每个 SKU 明细行均保留，不用 Order ID 单独去重。
- 导入日志包含原文件、时间、入库／跳过条数及状态，导入者为 RPA。
- 返回 `{ "status": true, "msg": "订单导入成功，入库 N 条" }`，失败时返回 `{ "status": false, "msg": "具体失败原因" }`。

```bash
curl -X POST "https://你的API域名/api/internal/tax-data/tiktok/orders/upload" \
  -H "X-RPA-Key: 现有RPA密钥" \
  -F "file=@订单.xlsx"
```

## 结算单匹配

仅结算单 `Type` 去除首尾空白后为 `Order` 时参与订单匹配。其他类型（含空值）物流方式统一为“杂费”、订单类型为 `Other`，不参与季度配送费的物流分项及待匹配计数。

结算单 `Order/adjustment ID` 精确匹配订单 `Order ID`（去除首尾空白），不将长 ID 转为数字。空 ID 不参与匹配。

| Delivery Option Type | Delivery Option | 结算单物流方式 |
| --- | --- | --- |
| 商家物流 | 标准运输 | MCF |
| TikTok 物流 | 标准运输 | 海外仓 |
| TikTok 物流 | Fulfilled by TikTok Shop 标准 | FBT |

结算单订单类型直接使用 `Normal or Pre-order` 的原值。未匹配订单时两项留空；匹配订单但不符合规则时物流方式留空，订单类型仍正常显示。

已有与后续结算单均在后端查询、导出时自动匹配，无须重新导入结算单。同一 Order ID 有多条 SKU 或多次导入时，仅选最新导入的一条订单记录作为订单级匹配来源，不增加结算单条数、不重复计算结算金额。原始文件和历史已下载报表快照保持原样。

## 页面接口

- `GET /api/tax-data/tiktok/orders?month=2026-01&quarter=2026-Q1&page=1&page_size=50`
- `GET /api/tax-data/tiktok/filter-options?type=orders&field=month&quarter=2026-Q1`
- `GET /api/tax-data/tiktok/export?type=orders&quarter=2026-Q1`

沿用报税数据访问权限。分页支持每页 25、50、100、200 条；筛选作用于完整数据库数据；导出当前筛选的全部记录并保存下载日志。导入／下载记录通过已有日志接口的 `platform=tiktok` 查询。

部署前后端后，后端启动自动创建订单和导入批次表。订单明细不直接加入季度汇总，避免与涉税明细重复计算。


## 季度汇总配送费

配送费总额始终来自 FBT 表的 FBT Fulfillment Fee，按 FBT 表月份归属季度；人民币使用该月数据库内最后一个有效 USD 汇率，美元直接求和。先汇总未舍入金额，最终四舍五入到两位。其他收入、分仓费、仓储费及报税收入口径不变。

页面仅显示配送费总额，点击配送费金额及箭头在单元格内展开结算单物流费用，再次点击收起；支持键盘 Enter／空格操作。分项按声明日期归属季度，按对应日期汇率换算，匹配规则为：

| 物流方式 | 结算单取值 |
| --- | --- |
| MCF | Seller shipping fee |
| FBT | FBT fulfillment fee |
| 海外仓 | TikTok Shop shipping fee |

结算单分项用于参考，不相加替代 FBT 表配送费总额。未匹配订单仅在明细内提示条数，订单补录后自动重新匹配。结算单分项缺少当日汇率时显示“待换算”，不会影响 FBT 表总额；FBT 表自身缺少月内汇率时总额仍为“待换算”。下载报表保留结算单分项，总额列标明来自 FBT 表。
