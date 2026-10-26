# 财务 / 报关单

页面：`/finance/customs-declarations`。沿用报税数据的权限，仅管理员及财务部门可访问。

页面采用全宽紧凑布局。顶部 BI 看板展示报关单数、商品项数、件数、毛重、净重、待核对数量，以及分币种商品金额、运费、保费、月度申报趋势和指运港分布。申报日期使用单入口双月历范围选择器，支持跨月、跨年、同日起止和快捷期间；指运港使用下拉筛选。选择后同时更新看板与明细。看板统计全部匹配记录，不受列表分页影响。金额采用十进制汇总，不混加不同币种；运费、保费仅计入明确标记“总价”的费用，且每份报关单计一次。缺失、无效及非总价费用明确提示未计入；没有有效值时展示“—”。

列表覆盖海关编号、合同号、进出口日期、申报日期、提运单号、贸易方式、指运港、运保费、件数和重量，以及商品名称、法定数量、申报数量、单价、总价、币制；勾选“显示全部字段”可查看完整识别信息。表格采用报税数据页面的紧凑样式，固定表头、海关编号和操作列，支持每页 20 / 50 / 100 份。新上传文件会保留贸易方式标题中的四位代码；历史记录未提取的代码不会推测补填。导入先检查文件哈希，再按识别到的海关编号去重；不同文件名或不同 PDF 内容只要海关编号相同均跳过并返回原记录编号，数据库唯一约束防止并发重复写入。

上传 PDF 后，由服务端提取字段、商品明细和原文，保存到 PostgreSQL `customs_declarations` 表。原 PDF 一并保存，列表刷新后仍可查看和下载。无需配置第三方 AI 服务，文件不发送至外部识别服务。

“批量导入报关单”支持一次选择多个 PDF，按选择顺序逐份上传识别，沿用单文件接口和海关编号去重机制。页面显示已处理进度以及各文件成功、重复跳过、失败的结果；单个文件失败不会中断剩余文件。每个文件独立限制为非空 PDF、最多 20 MB / 20 页。处理期间请保持页面打开，完成后自动刷新看板和列表并保留当前筛选条件。

一张完整报关单对应一条记录，最多 20 MB / 20 页。优先按文字、表格和相对列位置读取，扫描页面使用本地 Tesseract 中文 OCR。主要适配带“海关编号”、表头及商品列的海关报关单；非标准版式可能仅提取部分字段。缺失字段和 OCR 结果会标记“需核对”，无法取得唯一 18 位海关编号时拒绝导入。同一文件 SHA-256 或同一海关编号重复上传均保留原记录，不覆盖。当前不提供修改或删除功能。

## 服务部署

后端启动自动建表。Dockerfile 运行阶段采用 Python Debian slim 镜像以支持 PDFium 二进制依赖，并加入 Python 虚拟环境、pdfplumber、Tesseract 中英文依赖，需重新构建后端镜像并发布前端。

非容器部署时：

Python 路径优先使用 `CUSTOMS_PYTHON`，未设置时复用 `STP_WORKER_PYTHON`，均支持系统环境变量和后端 `.env`；都未设置则使用 `python3`。修改配置后重启后端。已有 Miniconda 环境无需重新创建环境，在后端目录执行：

```sh
/opt/miniconda3/envs/py311/bin/python -m pip install -r internal/customs/requirements.txt
/opt/miniconda3/envs/py311/bin/python -c "import sys, pdfplumber; print(sys.executable); print(pdfplumber.__version__)"
```

仅识别文字 PDF 时无需 Tesseract；扫描件额外需要安装下面的 Tesseract 中英文识别依赖。发布时需同时包含 `internal/customs/parse.py`，不能只更新 Go 可执行文件。新建独立环境可使用：

```sh
python3 -m venv .venv-customs
.venv-customs/bin/pip install -r internal/customs/requirements.txt
export CUSTOMS_PYTHON=/absolute/path/.venv-customs/bin/python
# Debian / Ubuntu 扫描识别依赖
sudo apt-get install tesseract-ocr tesseract-ocr-chi-sim tesseract-ocr-eng
```

Windows 可将 `CUSTOMS_PYTHON` 指向安装了依赖的 `python.exe`，并将 Tesseract 加入 PATH。后端工作目录须为 `backend`（容器内为 `/app`），以定位 `internal/customs/parse.py`。反向代理应允许至少 21 MB 请求体和 150 秒超时；接口限制 PDF 实际大小为 20 MB。

## 接口

页面右上角“下载报表”导出 Excel（`.xlsx`），包含当前日期、指运港筛选下的全部匹配报关单，不受分页限制。列顺序跟随当前显示字段，“显示全部字段”勾选时导出完整列；单据信息合并展示，商品逐项列出。另附“导出说明”工作表记录筛选条件、导出时间及统计口径。

以下接口均需要登录令牌及财务权限：

- `GET /api/finance/customs-declarations/export`：使用与列表一致的筛选参数，返回全部匹配记录用于生成报表，不包含 PDF 内容及提取原文。

- `POST /api/finance/customs-declarations`：multipart 字段 `file`；新建返回 201，重复返回 200 和 `duplicate: true`，识别失败返回 422。
- `GET /api/finance/customs-declarations?page=1&page_size=20&keyword=...`：分页列表；关键词匹配海关编号、合同号、收发货人和文件名。
- `GET /api/finance/customs-declarations/summary`：全部匹配报关单的 BI 汇总，附带全量 `destination_options` 指运港选项。此接口与列表均支持 `destination_port`（去首尾空格后精确匹配）、`date_from`、`date_to`（申报日期，YYYY-MM-DD，含起止日）。保留 `keyword` 接口兼容性，页面不再提供关键词搜索。
- `GET /api/finance/customs-declarations/:id`：字段、商品明细、识别警告和提取原文。
- `GET /api/finance/customs-declarations/:id/file`：下载原始 PDF。

金额和编号以原始字符串保存，避免浮点金额误差及前导零丢失。费用保留币种、数值、计价方式。识别数据仅用于整理，业务使用前应核对原 PDF。

## 验证

在 `backend/internal/customs` 执行 `python -m unittest -v`。设置 `CUSTOMS_SAMPLE_PDF` 后额外验证用户提供的真实样例，样例原文件不加入仓库。后端执行 `go test ./internal/handler ./internal/model ./internal/router`。前端执行 `npm test` 和 `npm run build`。
