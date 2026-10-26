# CPU 抠图服务

## 2026-09-21 商品区域细化更新

默认推理为 3 线程，原生服务 CPU 额度为 300%，仍单任务串行、关闭内存 arena。4 核服务器给其他服务保留约 1 核额度；此前 4 线程测试数据仅作为对比。已有服务器还需将 threads.conf 中的覆盖配置同步改为 3 线程和 300%。
完整模型先定位所有高置信度部件，保留 12% 上下文；仅当包围区域不足全图 60%、最长边大于 1024 时，再针对该区域运行一次相同模型，回填原尺寸蒙版。区域外低置信度像素保留，不根据商品坐标硬裁。
该策略改善本次龙头图左侧底座倒影残留，但不能保证所有商品都改善。
本机同轮对比：2 线程单次约 23.43 秒；4 线程单次约 12.22 秒、输出像素一致；4 线程加区域细化首次约 27.10 秒、随后约 22.22 秒，峰值提交内存 3.70GiB。旧版下述耗时仅供历史参考。CPU 及 BLAS/Numba 默认线程以当前脚本为准。
不启用实验性 alpha matting：它未可靠去掉底座残留，且增加内存占用。模型文件及校验值不变，已安装完整版的服务器只需更新 app.py 和线程配置。

使用 `rembg[cpu] 2.0.84` 和完整版 `birefnet-general`，明确指定
`CPUExecutionProvider`。模型会话在启动时加载一次；只运行一个 Uvicorn worker。
Go 后端在两个白底图接口之间串行调度请求（含排队最多 15 分钟），最多接收 3 个任务（1 个执行、2 个等待），队列满返回 429。Python 服务也会拒绝重叠推理。
多副本 Go 后端同时调用同一服务时，额外请求可能收到繁忙错误。

## 部署

### 宝塔直接运行 Go 后端（不用 Docker）

在服务器项目根目录，以 root 执行：

必须同时上传安装脚本及 `backend/rembg-service` 目录中的 `app.py`、`requirements.txt`，仅上传脚本无法安装。
另外将离线模型 `birefnet-general-efficient.onnx` 上传到该目录，或直接放入 `/www/wwwroot/haseman-api/rembg-models`。
也支持将脚本放入 `backend` 目录，与 `rembg-service` 文件夹并列后运行。

```sh
bash scripts/install-rembg-native.sh
# 使用宝塔已有的 Python：
bash scripts/install-rembg-native.sh /www/server/pyporject_evn/versions/3.12.13/bin/python3.12
journalctl -u haseman-rembg -f
curl -f http://127.0.0.1:8011/health
```

支持 Python 3.11、3.12，需要 venv。解释器选择顺序：命令行路径、终端已导出的 `AI_AGENT_PYTHON`、`python3`。
宝塔项目或 `.env` 中的变量不会自动进入安装终端；这种情况直接传入上述绝对路径，无需激活环境。
脚本创建独立虚拟环境，不往 AI Agent 的 Python 环境安装依赖。
脚本安装到 `/opt/haseman-rembg`，注册独立 systemd 服务，开机启动、失败重启；
仅监听服务器本机 8011。离线模型加载成功，健康检查返回 `status: ok` 后才可处理。
模型最终放在 `/www/wwwroot/haseman-api/rembg-models`，启动时校验 SHA256；缺失或损坏直接报出路径，不联网重试。
安装依赖需要访问 PyPI。必须同步更新并重新编译 Go 后端：旧二进制只接受 ISNet 的健康检查，即使 Python 正常也会拒绝新模型。
配置宝塔 Go 后端环境变量 `REMBG_SERVICE_URL=http://127.0.0.1:8011` 并重启后端。
更新后的 Go 代码也支持在 `backend/.env` 配置，但旧二进制必须通过宝塔进程环境变量设置。

systemd 使用 `CPUQuota=200%`（最多 2 核）、`MemoryMax=5637144576`（5.25GiB）、`MemorySwapMax=0`、低优先级、单 worker。
执行 `systemctl show haseman-rembg -p CPUQuotaPerSecUSec -p MemoryMax -p MemorySwapMax` 核查配置；
swap 限制需 cgroup v2 支持。实际负载和限制生效情况需要在目标服务器验证。

### Docker 部署

仓库根目录的 `docker-compose.yml` 已添加此服务和后端的 `REMBG_SERVICE_URL`。
构建前把离线模型 `birefnet-general-efficient.onnx` 放进 `backend/rembg-service`。镜像构建时校验模型，不在启动或请求时下载。

```sh
docker compose build rembg-service
docker compose up -d rembg-service
```

然后部署包含本次代码的新 Go 后端镜像及前端构建，并重建后端容器。
Compose 的 backend 使用预构建镜像，单独启动 rembg 不会更新旧后端代码。
Compose 同时发布 `127.0.0.1:8011`，供宿主机 Go/PM2 后端调用；不对公网开放。容器内的后端仍使用 `http://rembg-service:8001`。

开发环境直接运行（Python 3.11 或 3.12；仅线程限制，不提供内存/CPU 硬隔离，生产请用上述 systemd 或 Compose）：

```sh
cd backend/rembg-service
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt
OMP_NUM_THREADS=2 NUMBA_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 MKL_NUM_THREADS=2 U2NET_HOME=./models uvicorn app:app --host 127.0.0.1 --port 8011 --workers 1
```

Windows 使用 `.venv\Scripts\Activate.ps1`，以 `$env:OMP_NUM_THREADS='2'` 等设置环境变量。
Go 后端默认访问 `http://127.0.0.1:8011`，独立主机部署时设置 `REMBG_SERVICE_URL`。
先把离线模型放在 `U2NET_HOME` 指定目录的顶层；`GET /health` 在模型校验和加载成功后才可用。
同步 `/api/white-bg/generate` 的反向代理读取超时应不少于 960 秒；
白底图页面使用已有异步任务接口，不需要保持长连接。

## 连接拒绝排查

若日志含 `cannot cache function '_make_tree': no locator available`，检查 Numba 缓存目录权限。
早期安装脚本未显式创建可写缓存目录，可在服务器执行以下修复，无需重装依赖：

```sh
install -d -m 755 -o haseman-rembg -g haseman-rembg /var/lib/haseman-rembg /var/lib/haseman-rembg/numba
systemctl reset-failed haseman-rembg
systemctl restart haseman-rembg
journalctl -u haseman-rembg -f
```

`connection refused` 表示后端指定地址没有 rembg 监听。不要只更新 Go 二进制：还要启动 Python 服务。
默认本地端口使用 8011，避开本机其他应用占用的 8001。

- Windows 开发：仓库根目录运行 `powershell -ExecutionPolicy Bypass -File scripts/start-rembg.ps1`，自动安装依赖并后台启动（2 个逻辑核心、低优先级）。
- Linux / PM2 部署：运行 `bash scripts/start-rembg.sh`，启动有 2 核 / 5.25GiB 硬限制的容器。
- 修改过 `REMBG_SERVICE_URL` 时，将其改为相应地址并重启 Go 后端。
- 用 `curl http://127.0.0.1:8011/health` 确认 `model` 为 `birefnet-general`，避免连到其他应用。容器后端不要使用 127.0.0.1 指向另一个容器。
- 新建任务前会检查服务身份与就绪状态，不可用时返回 503 简短提示；技术错误保留在后端日志。

## 输出与资源

- 导航中“去背景”页面对应的白底图同步接口和异步任务使用 rembg，无需绘图 API Key。
- 独立透明背景工具 `/remove-bg` 保持原有逻辑；本服务提供的透明/黑底能力用于底层合成和验证。
- 页面上传即提交任务，以文件名标识图片。新页面明确请求透明 PNG，背景切换和下载合成在浏览器完成，不重复推理。支持透明、预设颜色、自定义纯色。收藏保存透明原图。
- 旧调用方未传 `background` 时继续生成白底；历史白底结果不显示可用的换底控件，避免伪装成支持透明。
- 保留 EXIF 方向修正后的原图尺寸和比例，PNG 无损输出；不再根据 2K/4K 或比例重画商品。
- 模型内部推理采用原模型的 1024×1024 输入，再恢复原图大小的蒙版；原尺寸输出不等于每个细节都按原分辨率推理。
- 开启 `decontaminate` 减轻边缘底色；置信度达到 250/255 的前景直接保留原图 RGB 像素，避免商品纹理被前景颜色估计平滑。此阈值只用于选择 RGB，不二值化透明度；软蒙版保持原样。
- 不限制输入像素数，保留原图输出尺寸。已有透明度会与新蒙版相乘保留；运行资源仍受容器上限约束。
- 针对 4 核 / 7.3GB 主机，CPU 保持 2 核，内存上限按用户同意调整为 5.25GiB（5637144576 字节）。Compose 的 `mem_limit` 和 `memswap_limit` 设置同一字节数，不额外使用 swap；`pids_limit: 128`。其他服务需要在剩余资源内运行。
- 推理、Numba、BLAS 线程均设为 2，单 worker；关闭 ONNX 内存 arena 与 memory pattern，减小峰值。超出容器内存预算可能导致该任务失败及服务重启，不会自动切换模型或压缩原图。
- 使用完整版 FP32 权重及 1024×1024 输入；离线图把 20 个可变形卷积改成逐采样点 GridSample + 1×1 卷积累加，避免展开巨大的中间张量；未量化、未训练、未修改权重。
- 4581×3052 龙头实图：ISNet 残留大面积脏边与倒影，Lite 仍保留右侧倒影，完整版去除倒影并保留三个部件。本机 Windows / 2 线程完整 API 连续两次请求约 21.35 / 16.88 秒，累计峰值 RSS 3.43GiB、峰值提交内存 3.57GiB；不代表服务器耗时或 Linux cgroup 峰值。
- 对比原始完整版与优化图（均未应用原像素保留）：该图透明度最大误差 1/255，平均误差约 0.0000042/255。Windows 旧图 RSS 受分页影响，先前单次 5.25GiB 不是固定内存需求；其提交内存达到约 6.99GiB，所以生产使用优化图。

## 离线模型构建与校验

生产不需要 PyTorch 或 ONNX 转换工具。模型不纳入 Git，使用离线包上传。

- 文件：`birefnet-general-efficient.onnx`，934224215 字节。
- SHA256：`b6cc66609e98f76e66c0d7df0eea477d80f08df4cf59e86540b8bd52fe0d7546`。
- 原始 rembg 导出：`BiRefNet-general-epoch_244.onnx`，MD5 `7a35a0141cbbc80de11d9c9a28f52697`。
- 构建机安装 `onnx==1.23.0` 后执行 `python optimize_model.py 原始模型.onnx birefnet-general-efficient.onnx`；脚本拒绝未知源文件。转换不使用用户图片。
- 逐采样点分解思路参考 [CoderViking 的 Lite 导出](https://huggingface.co/CoderViking/birefnet-lite-onnx)，此处转换的是 rembg 完整版，不使用 Lite 权重。

## 验证

```sh
pip install httpx==0.28.1
python -m unittest -v test_app
```

以上测试使用模拟蒙版验证接口、尺寸、原有透明度、纯色合成、繁忙和失败恢复，不能替代模型效果验收。
上线前用真实商品样图检查白色商品、金属反光、细杆、孔洞和透明材料；将透明结果叠在黑色与白色底上，
核对轮廓是否完整、是否有白边，并记录 CPU 耗时和内存。无真实样图时不承诺已达到业务抠图质量。

模型与工具来源：[rembg](https://github.com/danielgatis/rembg)、[BiRefNet](https://github.com/ZhengPeng7/BiRefNet)。
