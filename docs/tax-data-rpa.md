# 报税数据：RPA 对接与部署

## 部署配置

后端启动时自动创建税务专用表并导入 53 条费用类型规则（用户原表的完全重复行合并）。不需要在 web 页面上传文件。

RPA 上传密钥按用户要求固定在后端 `internal/handler/tax_data.go` 的 `taxDataRPAKey` 常量中，不读取 `.env`。重新编译并重启后端后生效，更换密钥需修改代码并重新部署。

按用户要求，CurrencyBeacon Key 固定于后端 `internal/handler/tax_data.go` 的常量中，不从 `.env` 读取。RPA 和前端不需要传递；更换 Key 需修改后端代码并重新部署。

## RPA 上传

`POST /api/internal/tax-data/raw/upload`

请求头：`X-RPA-Key: <TAX_DATA_RPA_KEY>`

请求类型：`multipart/form-data`

| 字段 | 要求 |
| --- | --- |
| store_id | 必填，稳定的店铺标识；不同店铺必须不同，重试时不能变 |
| file | 必填，UTF-8 CSV、XLSX 或 XLS；最大 20MB、100000 数据行；XLSX 解压后最大 512MB，单个 XML 最大 128MB |
| importer | 可选，仅 RPA 密钥导入使用：RPA 上报的操作人姓名/账号，最长 140 字符；未传显示 RPA，此字段不代表已认证的用户身份。网页登录导入时忽略该字段，直接显示登录用户名 |

Windows 命令行测试（将域名、密钥、店铺和本地文件路径替换为实际值）：

```powershell
curl.exe -X POST "https://你的系统域名/api/internal/tax-data/raw/upload" -H "X-RPA-Key: 你的TAX_DATA_RPA_KEY" -F "store_id=walmart-shop-001" -F "file=@D:/RPA/bill.xlsx" --max-time 600
```

本机直接访问后端可用 `http://localhost:8080/api/internal/tax-data/raw/upload`。不要手动设置 Content-Type，curl/RPA 文件上传组件会自动生成 multipart boundary。不传日期、季度或汇率。`TAX_DATA_RPA_KEY` 是系统上传鉴权密钥，不是 CurrencyBeacon 汇率密钥。

Python RPA 示例（依赖 requests）：

```python
import os
import requests

with open(local_bill_path, 'rb') as bill:
    response = requests.post(
        os.environ['SYSTEM_BASE_URL'].rstrip('/') + '/api/internal/tax-data/raw/upload',
        headers={'X-RPA-Key': os.environ['TAX_DATA_RPA_KEY']},
        data={'store_id': walmart_store_id, 'importer': '财务操作人姓名'},
        files={'file': (os.path.basename(local_bill_path), bill)},
        timeout=(15, 600),
    )
response.raise_for_status()
result = response.json()
if not result.get('status'):
    raise RuntimeError(result.get('msg', '导入失败'))
# status=true 表示事务已完成；日志中的待核对状态由网页查看。
```

接口同步完成事务后返回统一结构：成功为 `{ "status": true, "msg": "成功" }`，失败为 `{ "status": false, "msg": "具体失败原因" }`。HTTP 状态码仍用于网络/鉴权判断；RPA 以 JSON 的 status 为最终业务结果。

- 400/422：格式、表头、日期、金额错误；整份未入库，修改文件后再提交。
- 401：密钥无效。
- 重复行不报 409：跳过重复行，返回 imported_rows、skipped_rows 和 import_log_id。整份都是重复数据时返回 200。
- 503：汇率获取失败；未入库，修复配置/网络后重试。
- 500：存储失败，事务回滚；可重新提交相同文件。

连接超时不意味着失败，RPA 应重传同一文件确认；文件哈希与店铺标识提供幂等性。不要为重试生成不同 store_id。

XLSX 读取第一张工作表，CSV 使用逗号分隔；可在前 50 行中定位表头。旧二进制 XLS 不支持，RPA 应先转换为 XLSX。CSV 中有逗号的金额或描述必须正常加引号。

## 后端处理口径

1. 原始数据：保存源文件二进制和每一行所有表头字段（包含用户列出的 42 个平台字段；额外字段也保存），不把月份/季度/费用类型/费用写回原始行。
2. 校验必需字段：Transaction Posted Timestamp、Transaction Type、Transaction Description、Amount Type、Amount、Currency、Transaction Key。空的类型字段允许参与精确匹配，Amount 和 Currency 允许为空，原始值保留；空 Amount 在计算时按 0，空 Currency 按 USD。非空金额、币种和交易时间仍须有效。
3. 费用类型：以 Transaction Type、Transaction Description、Amount Type 三元组匹配；规则表为空或 MatchKey 未初始化时，后端自动使用内置规则恢复匹配；去除首尾空白与外围引号，保留大小写。空白字段不是通配符。金额明细保存命中的规则快照；以后规则变化不修改历史数据。
4. “分摊”仅是对应表辅助标记，无额外金额分配作用。
5. 月份/季度：逐行按 Transaction Posted Timestamp，跨季度文件分开归属。带偏移的时间保留所给日期口径，不换成服务器时区；Unix 秒/毫秒时间按 UTC；支持 ISO、常用美国月/日/年及 Excel 序列日期。
6. 费用：Amount × 当次导入的币种兑 CNY 汇率，用十进制计算并逐行四舍五入到两位小数后汇总。原币 Amount、汇率均另行保存。
7. 汇率：每次新文件导入时由后端获取；同批次同币种只获取一次。调用 CurrencyBeacon `/v1/convert`，固定 `from=USD&to=CNY&amount=1`，不传 `date` 参数，使用接口最新汇率，用户无需填写日期。`response.value` 即 1 美元对应的人民币汇率；保存来源、响应日期、服务端 timestamp 和获取时间。CNY 为 1，其余币种暂不支持。请求失败或无效响应拒绝入库，不使用假值或 0 替代。
8. 季度“报税收入” = 销售额 + 带正负号的退款净额（负退款减少收入，正冲销增加收入）；总计是 10 个费用类型之和。PaymentSummary 结算汇总整行跳过；已入库的历史结算行也不在原始账单列表、季度汇总和新导出报表中显示。原文件下载仍保留原始内容。
9. 未匹配行仍保存原始与处理记录，整个批次标为 needs_review，季度页面和导出按已保存的处理明细实时汇总：已分类数据正常展示，未匹配费用单独列出并标记条数，不计入总计和报税收入。当前不提供规则编辑或自动历史重算接口。
10. 重复保护：同一店铺内，以 Transaction Key、Transaction Type、Transaction Description、Amount Type 四个字段（去首尾空白、区分大小写）组合去重；空值也参与组合。文件内和历史已入库的重复行都跳过，仅新行处理并汇总。即使 Amount 或日期不同，四字段相同仍判为重复，保留首次入库记录。不同店铺分别去重。相同文件重传不刷新汇率。旧数据参与新规则去重，但不会自动删除已有历史重复行。
11. 通过密钥鉴权、读取文件后的每次导入会保存日志及原文件；记录导入者、时间、源记录数、入库数、重复跳过数、状态和失败原因。导入者口径：RPA 密钥导入显示 RPA（或 RPA 上报的 importer 身份）；网页登录导入显示登录用户名。无法读取文件或未通过鉴权的请求不生成文件日志。旧批次补建日志，身份标为“历史导入（身份未记录）”。
12. 报表下载按当前季度筛选导出全部匹配记录（不受页面分页限制）。下载日志记录登录用户、类型、季度、行数、文件名和服务端返回时间；保存文件快照以供再次下载。该记录不能证明客户端已保存到磁盘。原文件下载按原始字节返回；生成 CSV 时对可执行公式文本进行转义。


这套汇率是平台参考汇率，并非另外认定的申报汇率来源；当前按用户指定的“本次获取并固化”口径实现。
接口地址：https://api.currencybeacon.com/v1/convert

## Web 读取接口

以下均需要已有登录 Bearer token，后端实时查询用户是否为 admin 或 department/sub_department 包含“财务”。其他用户返回 403。

- GET /api/tax-data?page=1&page_size=50&store_id=...：分页批次，包含每批次汇率快照。
- PUT /api/tax-data/quarters/:quarter/rate：修改指定季度 USD/CNY 汇率，JSON `{ "rate": "7.18", "currency": "USD" }`，只重算该季度。
- GET /api/tax-data/quarters?store_id=...&quarter=2026-Q1：基于处理明细的季度汇总；available_quarters 返回所有已入库季度（含待核对批次），用于筛选。
- GET /api/tax-data/raw?page=1&page_size=50&quarter=2026-Q1：跨批次原始账单，先按交易所属季度筛选再分页。
- GET /api/tax-data/batches/:id/raw?page=1&page_size=50：原始行、总行数和表头。
- GET /api/tax-data/batches/:id/processed?page=1&page_size=50：处理行、规则快照和总行数。
- GET /api/tax-data/rules：完整费用对应表。
- GET /api/tax-data/imports?page=1&page_size=20：导入日志。
- GET /api/tax-data/imports/:id/file：下载导入时的原始文件，并记下载日志。
- GET /api/tax-data/export?type=raw&quarter=2026-Q1：导出原始账单与四个处理字段、汇率；type=quarters 导出季度汇总。省略 quarter 为全部季度。
- GET /api/tax-data/downloads?page=1&page_size=20：下载记录。
- GET /api/tax-data/downloads/:id/file：重新下载历史文件快照，并新增一次下载记录。

page_size 最大 200；省略 store_id 表示全部店铺。当前财务/管理员查看所有店铺，不设店铺隔离权限。

## 验证

单元测试与隔离数据库集成测试在 backend 执行：

```powershell
# 指向专用本地测试 PostgreSQL；测试创建事务内隔离 schema，结束回滚。
$env:TAX_TEST_DSN = 'host=127.0.0.1 port=55439 user=tax_test dbname=postgres sslmode=disable'
go test ./internal/handler -run '^TestTax' -count=1 -v
```

测试覆盖 CSV/XLSX、53 条对应规则、跨季度时间、费用精度、退款口径、结算排除、幂等重试、四字段去重、汇率快照不变、未匹配批次、分页、原文件保存及权限撤销。测试汇率响应通过隔离的 HTTP 测试替身控制，不写入业务数据库。

CurrencyBeacon Key 按用户要求固定在后端代码中，不在前端和 `.env` 保存。2026-10-05 先前使用该密钥带 date 参数真实请求得到 HTTP 403，尚未取得真实汇率。部署环境需确认网络/账户可调用 convert 请求；失败时不入库。
