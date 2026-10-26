# TikTok 涉税申报汇总表

涉税明细导入接口只提交一个工作簿文件：

```text
POST /api/internal/tax-data/tiktok/upload
file          同一工作簿文件
```

XLSX/XLS 按工作表名称识别：名称包含“汇总”或 `summary` 的工作表为汇总表，名称包含“明细”或 `detail` 的工作表为涉税明细表，不依赖工作表顺序。找不到这两类工作表会拒绝导入。汇总表需要包含 `年份-季度` 和 `交易笔数` 表头。系统按年份-季度去重保存交易笔数；季度汇总优先展示该值，并在下方展示系统计算的销量和样品数。CSV 继续支持旧的单表明细导入，此时交易笔数显示“待导入”。

季度汇总的 FBT 费取 FBT 表：

```text
外层 FBT 费 = -(Total Logistics Cost - FBT Fulfillment Fee - FBT Fulfillment Fee Reimbursement)
```

FBT 费展开明细为仓储费、分仓费和其他；其他为 `Inbound Shipping Fee`、`Inbound Incidents Fee`、`Inbound Booking Non-Compliance Fee`、`Routing Non-Compliance Fee`、`Outbound Appointment No-Show Fee`、`Delayed Response Fee`、`Disposal Fee`、`Return to Seller Shipping Fee`、`Return to Seller Handling Fee`、`Inbound Return Operation Fee` 之和。
