# 商品价格台账

入口：顶部导航「运营工具 → 商品价格」，路径 `/product-prices`。

- Amazon 原价、可选折扣价和价格更新时间保存在产品库 SKU 上。
- 商品范围与产品库一致：仅包含通过 `spu_id` 关联有效 `product_spus.id` 的 SKU，不显示未归属产品库的历史 SKU。
- 价格页按美国站范围展示：排除产品库 SPU 以 `CA`、`EU`、`UK` 开头的记录（忽略大小写及首尾空格），其余保留。该规则仅用于价格页，产品库继续展示全部站点。
- TikTok 售价、PID 和记录更新时间按平台 SKU 保存，通过订单转 MCF 的当前有效映射关联。更换当前 Amazon SKU 后保留 TK 价格与 PID；历史、停用映射不显示。没有 TK 映射的产品仍可维护 Amazon 价格。
- 一个 Amazon SKU 对应多个当前 TK SKU 时会显示多行，Amazon 价格共用。记录数按关联行计算。
- 已有当前有效 TK 映射的商品优先展示，组内按 Amazon SKU、TK SKU 排序；排序在服务端分页前执行。
- 支持手工记录美元价格，以及点击「获取亚马逊价格」通过 Firecrawl 回填，不保留历史价格。保存时间由服务器生成，界面按浏览器时区显示。未填写价格为 NULL，零价格为有效金额。PID 使用字符串，支持多个 TK SKU 使用相同 PID。
- 点击 ASIN 打开 `https://www.amazon.com/dp/{ASIN}`；点击 PID 打开 `https://www.tiktok.com/view/product/{PID}`。链接根据当前编号生成，不重复存储。
- 登录用户可查看。Amazon 编辑权限与现有产品库保持一致；TK 编辑仅限管理员或具有 TikTok 平台权限的用户。

## 维护交互

- 原价、折扣价与 TK 售价独立成列，桌面固定表头、产品列及操作列；支持每页 20 / 50 / 100 条。
- 点击金额或「维护」打开右侧面板，点击「补充 PID」直接定位到 PID 输入框。平台 SKU 旁提供复制按钮。
- 面板支持切换平台、查看有效售价、上下条切换及「保存并下一条」。连续维护限当前页，Amazon 按 SKU 去重，避免重复修改同一产品。
- 保存后原地更新相关行及时间；关闭面板后重新校准筛选结果。失败时保留输入内容，未保存时关闭、切换或离开页面会提示。
- 支持 Ctrl / Command + Enter 保存及 Esc 关闭；未改动的记录禁用保存，避免无意义地刷新更新时间。
- 仅未关联 TK 的行可「添加映射」。已映射商品在价格页不可新增或切换映射，已有 TK SKU 的映射须在订单转 MCF 中维护。复用同一映射接口和权限，并通过 `only_unmapped` 在服务端检查，避免旧页面覆盖已有映射。价格和 PID 仍可维护。

## 接口

- 「获取亚马逊价格」处理全部美国站当前已关联 TK 的 SKU，不受分页或筛选影响。同一 SKU 的多个映射只处理一次。沿用系统设置中的 `firecrawl_api_key` 和 `firecrawl_proxy`，密钥不发送至浏览器。
- 页面依次请求并显示进度、成功数量及未更新明细；停止后完成当前商品，不再请求后续商品。获取期间需保持页面打开。
- 代理地址留空时直接连接 Firecrawl，不使用系统环境代理；已有代理设置需确保后端服务器可达。建立连接最多等待 10 秒。网络连接失败时停止本批次，并提示代理、DNS 或响应超时原因，保留未更新价格。
- 仅采用选定 ASIN 的公开美元一次性购买售价；页面只有一个有效金额时填入原价并清空折扣价，页面有两个有效金额时按金额高低识别原价和折扣价。价格不再要求模型证据逐字匹配 Markdown；缺失 ASIN、抓取失败或币种不明时保留原值。抓取期间手动修改过的价格不会被覆盖。
- 对 Amazon 显示“加入购物车后查看价格”“当前不可用”等页面，自动改读当前 ASIN 的 Offers／Used from 价格；仅在该 ASIN 没有主商品公开售价时启用，不读取推荐商品或其他 ASIN 的金额。
- Firecrawl 价格抓取只请求页面 Markdown 和原始 HTML，不再请求易失败的 LLM JSON 提取；解析范围限定为当前 ASIN 的主价格区域及其 Offers 入口，排除标题上方的 Alternative items 推荐卡。
- `GET /api/product-prices/amazon-fetch-targets` 返回全部符合范围的去重 SKU。
- `POST /api/product-prices/amazon/{product_sku_id}/fetch` 抓取、校验并回填单条 Amazon 价格，权限与现有 Amazon 价格维护一致。

- `GET /api/product-prices?q=&filter=all&page=1&page_size=20`，筛选支持 `all`、`missing`、`sale`、`unmapped`，最多每页 100 条。
- `PUT /api/product-prices/amazon/{product_sku_id}`，完整提交 `{ "price": "39.99", "sale_price": null }`。
- `PUT /api/product-prices/tiktok/{current_mapping_id}`，完整提交 `{ "price": "29.99", "pid": "1729384756123456789" }`。

价格字段使用十进制字符串或 null，最多两位小数。折扣价须有原价且不高于原价。PID 为最多 32 位数字字符串或空字符串。

## 启用与验证

数据库迁移为 `000067_add_product_prices`，同时纳入现有启动时 AutoMigrate。部署新版后端并重启，再部署 web 构建结果。

后端验证：在 backend 目录运行 `go test ./...`。集成测试须设置 `PRODUCT_PRICE_TEST_ENV_FILE` 指向本地 PostgreSQL 的环境文件，再运行 `go test ./internal/handler -run TestProductPrice -v`。集成测试使用独立 schema 和回滚事务，不保留测试数据。

前端验证：在 web 目录运行 `npm run build`。
