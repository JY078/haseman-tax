---
name: write-wayfair-listing
description: Write evidence-led Wayfair retail copy with natural benefits and positioning-aware selling-point hierarchy.
---

# Write Wayfair Listing

消费者字段为 `about_this_product`、`features`（不输出 Title）；完整 JSON 按 Runtime Contract 返回内部 evidence、optimization_reasons 等字段。

## 参考风格与事实边界

学习运营提供的 Oletto/KRAUS 样本的组织方式：产品定位与外观价值建立第一印象，具体结构证明使用价值，再回答空间、维护、安装与保障问题。学习有信息量的卖点标题、自然的功能与利益连接、主次分明的篇幅；不复制原文、品牌、专有技术、规格或宣传承诺。

- `confirmed_facts` 是 SKU 声明依据。图片只支持直接可见的外观、结构和保守场景，不证明隐藏机制、尺寸、测试、认证和性能。
- 参考里的 Reach、DIAMOND Seal、Reflex、Power Clean、Duralock、500,000 次、360°、1.8 GPM、终身保修及比较数据都不是本品事实。只有本 SKU 输入明确支持才可写相应声明。
- 高弧可以表达空间方向，但不能保证任意高锅都放得下；单把手不能自动证明 ADA；陶瓷阀芯不能自动证明防漏；普通随附件不能自动证明快装。减少水渍、防腐、易擦矿物沉积是不同属性，不互相推导。
- 保留已确认的材料归属、质保期限和适用限制。事实不足时缩小声明，不用 `listed as` 或性能口号补齐。
- 允许写本 SKU 已确认的颜色、Finish 和表面处理；不得把参考样本的颜色选项当成本品选项。
- 证据只用于内部 evidence 和写作判断，不能把证据来源写给消费者。消费者文案禁止 `confirmed`、`specified as`、`listed as`、`according to the product information`、`the product is identified as`、`supports the confirmed` 等资料转述腔；直接写产品事实和用户得到的结果。

## 定位与优先级

先读取产品信息中的价格段、产品定位、目标人群和购买任务，包括中文自由文本，以及 `price_band`、`product_positioning`、`target_audience`、`buyer_profile`、`purchase_task`。明确定位优先，价格只辅助，不按价格臆造配置。不在消费者文案中给读者贴低预算或租客标签。

卖点顺序不是模型自由发挥：当输入给出下列定位时，Features 必须按对应的左侧顺序组织。每个位置只在本 SKU 有确认事实时出现；缺少某项就跳过并继续下一个已确认位置，不得用后面的低优先级事实填充前面的空位。About 可以预告前 1–3 个高优先级卖点，但不能改变 Features 的排序逻辑。

| 定位 | 优先回答的购买问题 | 后续支撑 |
| --- | --- | --- |
| 基础替换型 | 是否适配现有安装、是否好操作、是否满足装水清洗 | 简洁外观、清洁维护、已确认结构与保障 |
| 大众性价比型 | 哪些设计和功能改善日常使用、是否覆盖多个任务 | 抽拉/回位、配件准备、维护、安装匹配与保障 |
| 品牌改善型 | 外观和触感改善在哪里、哪些机制支撑体验 | 结构材料、抽拉/回位、空间控制、耐久证据、维护及保障 |

### 三类定位的 Feature 顺序

下面的顺序来自运营卖点表。括号内是写作时应落到的消费者利益和关键词方向，不是要求逐字照抄，也不是允许凭关键词补造事实。

| 定位 | Feature 1 → 后续顺序 |
| --- | --- |
| 基础替换型 | 现代极简外观（适配多种厨房风格） → 高弧形出水口（高锅/大壶的垂直空间） → 双功能喷头（接水与清洗） → 抽拉软管（扩大清洗范围） → 单把手温度控制（调水温和水量） → 90°前向手柄（靠墙或挡水板安装） → 软胶喷孔（擦除矿物沉积） → 陶瓷阀芯（仅写已确认的耐久依据） → 多孔位兼容与底板（安装替换适配） → 集成式进水管（仅写已确认的安装准备） → 安全认证与质保。 |
| 大众性价比型 | 商用开放式弹簧设计（专业厨房观感与适合家庭的高度） → 高弧＋旋转（大型锅具与水槽覆盖） → 强力喷淋（仅有比较或性能依据时写强度） → 三种用水状态 → 自动回位系统 → 下拉软管灵活供水 → 磁吸回位 → 易清洁喷头 → 抗指纹表面 → 抗腐蚀镀层 → 配套皂液器 → 1孔/3孔兼容与底板 → 快速连接 → 终身有限保修。 |
| 品牌改善型 | 顺滑抽拉系统 → 磁吸回位 → 灵活抽拉技术 → 增压技术（仅有流量依据时写增压） → 重型耐用结构 → 1.8 GPM 节水起泡器 → 易清洁软胶喷孔 → 防漏技术（仅有明确密封依据时写） → 抗水渍表面 → 耐腐蚀/耐褪色 → 终身有限保修。 |

“Feature 1”不是固定数量要求，而是优先级锚点。不要把三类定位的卖点混排，也不要因为产品事实很多就把认证、工具、尺寸和包装清单提前。低优先级事实必须合并到相关的安装、维护或保障条目中：工具清单不单独占核心 Feature，精确尺寸只有回答空间/安装问题时才出现，认证和质保合并为一条准确的保障信息，材质与阀芯只有能说明结构价值时才保留。若定位字段和目标人群冲突，以明确的 `product_positioning` 和 `purchase_task` 为主，并在 optimization_reasons 中说明调整原因。

没有定位时按实际差异和证据强度排序。没有某项事实就跳过，不补造；有独立价值的已确认卖点不能为了条数被删除。不设固定条数，也不照抄参考的 16 条数量。定位明确时，宁可少写，也要保持上表顺序。

## 字段分工

- About：连贯的产品介绍，先给产品定位或整体使用价值，用少量有说服力的设计和机制解释为什么，再自然收束到适配或维护。允许以产品为主语，不强制整段写厨房故事或祈使句。不要逐条压缩整个 Feature 清单。
- Features：默认用清晰的功能/利益短标题帮助扫读，标题可保持一致格式；正文提供标题尚未说出的机制、具体用途或购买价值。完整覆盖本品适用的高价值卖点，低价值配件归入安装，认证与质保保持准确，不能为了“安心感”删掉限制条件。
- About 可以预告核心利益，Features 用结构、场景、规格或适配细节展开。术语和主题重复不等于冗余；同一事实、动作和利益仅换词再说才算重复。不要为避免重复臆造第二个收益。

## 自然零售表达

先明确消费者为什么会想要这件产品，再写出自然、有吸引力的零售文案。以读起来顺畅为目标，不追求最短句或最低声明强度。功能、证据、场景、利益是选择信息的方法，不是每条必须拼出的四段句式。具体用途已经体现利益时，不追加 `keeping both controls within easy reach` 一类重复尾句，也不要为了精简删掉能帮助读者想象使用体验的细节。

### 表达可以积极，事实不能升级

- 产品事实：材质、功能、规格、认证、保修和性能必须有输入依据，引用对应 fact ID。
- 日常用途：可把已确认功能转译成常见任务，不要求产品资料逐字写出场景。例如双模式可写接满水壶、冲洗蔬果和餐盘；抽拉可写将水带到餐具旁。场景不能暗含未确认的全水槽覆盖、任意尺寸兼容、更强冲洗或更快清洁。
- 审美与情绪：允许根据已确认或直接可见的设计、颜色表达 `a clean, contemporary look`、`a warm accent`、`a welcoming kitchen`、`a coordinated sink area`。这是设计感受，不需要虚构测试数据来支撑；仍须有对应外观依据，不能保证所有买家的情绪或与任意装修风格完美适配。
- 体验利益：用动作和对象让读者感受到省心、从容、整洁或掌控感，不必直接写情绪词。核心卖点必须让人看懂购买价值，但安装、认证、质保不强行添加生活画面。
- `effortless`、`powerful`、`faster`、`leak-free`、`maintenance-free`、清除顽固污渍及比较级不是免费的情绪修饰，需要对应机制或性能依据。`polished look` 是审美措辞，`polished finish` 是表面处理事实，按实际语义区分。

不要把“有依据”理解为只能复制资料里的措辞。evidence 引用支撑功能或设计的事实；不要给正常的场景转译和审美表达编造新的 fact ID。审核时也按上述边界判断，不因原始资料没有同一句消费者语言而退回。

### 语气与节奏

About 让人先感受到产品的吸引力，再用具体设计和用途兑现这份吸引力。可以从外观更新、准备食材或收拾厨房等本品相关角度切入；不强制每次使用同一个开场。长短句交替，连接相关信息，避免连续使用 `Use...`、`Pull...`、`Turn...` 或 `The faucet...` 罗列操作。

Features 的标题保留可识别的功能词，也可带出利益，如 `Pull-down reach for everyday cleanup`。正文把结构与实际使用连起来，不重复解释标题。情绪来自选中的画面和顺畅节奏，不靠堆叠 `elegant`、`premium`、`perfect`，也不机械要求每条出现情绪形容词。先完成有吸引力的草稿，再核对事实；修正时只收窄越界的承诺，保留好的场景、语气和节奏。

可以用产品主语、功能标题、自然祈使句和从句。`features`、`pairs with`、`directs water` 本身不是禁词；连续罗列部件、空泛的 `streamlined setup`、反复解释同一动作才需要改写。不要为了变化句式破坏整组扫读一致性，也不要用 `premium`、`perfect`、`maximum`、`confidence` 代替证据。

以下仅为有条件的原创风格例，不是可直接复制的 SKU 内容：

| 已确认事实前提 | 标题与正文方向 |
| --- | --- |
| 双模式 Stream/Spray | Two ways to tackle cleanup: Fill a pitcher with the stream, then switch to spray to rinse food from plates |
| 已确认高容器操作空间 | Room for tall cookware: The high-arc spout leaves space to fill a stockpot underneath |
| 已确认单把手调温调流 | Single-handle control: Adjust temperature and flow with one handle |
| 已确认软胶喷孔支持擦除矿物沉积 | Wipe-clean nozzles: Rub the soft nozzles to remove mineral buildup |
| 已确认前向旋转与靠挡水板适配 | Made for tight spaces: The forward-rotating handle needs no clearance behind the faucet |

安装与保障无需强行写使用动作。分号、长句、统一标题不单独作为错误；出现独立卖点堆叠或歧义时拆句。一条只围绕一个购买问题，可用相关结构和事实共同支撑。

### 机械表达改写对照

不要停留在“换一个同义技术词”。先问消费者正在做什么，再把已确认的结构放进这个画面：

| 避免输出 | 推荐方向（事实前提成立时） |
| --- | --- |
| `The spout changes the water direction.` | `Turn the spout toward the side of the sink where you are rinsing a large pan.` |
| `One lever provides centralized faucet operation.` | `Adjust water flow and temperature from one easy-to-reach handle.` |
| `Stream and spray suit general sink tasks.` | `Fill a pitcher with the stream, then switch to spray to rinse produce or plates.` |
| `The faucet supports compatible setups.` | `Fits the confirmed 1-hole or 3-hole deck; the included plate covers unused holes.` |
| `The finish provides a streamlined configuration.` | `The confirmed finish adds a warm, coordinated accent to the sink area.` |
| `The package includes components for installation.` | `Pre-attached lines and the confirmed quick-connect parts keep the listed installation steps together.` |

上表右侧不是固定文案。只有输入确认了对应孔位、预装、快接、调温调流或颜色时才可采用；没有依据就删除该句或换成事实允许的更窄表达。注意示例里的 `confirmed finish` 只用于说明证据边界，最终消费者文案必须改成 `the Brushed Gold finish` 或直接写事实。`compatible setups`、`general sink tasks`、`centralized operation`、`planning water coverage`、`streamlined configuration` 属于应避免的空泛/机械表达。

### 连贯文案示例

仅适用于已确认现代弧形外观、暖金色、抽拉、Stream/Spray、单把手调温调流的假设 SKU；示例不是任何实际 SKU 的事实来源，也不是固定模板：

About: Give your sink a fresh, contemporary look with a gently curved faucet and a warm gold finish. From rinsing produce for dinner to clearing plates afterward, the pull-down sprayer brings water to the task at hand. Switch between stream and spray for filling pitchers and rinsing dishes, with temperature and flow controlled from a single handle.

Features:
- A warm, contemporary accent: The curved profile and gold finish bring a touch of warmth to your sink area.
- Pull-down reach for everyday cleanup: Bring the sprayer to your dishes, from rinsing a mixing bowl during prep to clearing plates after dinner.
- Two modes, from prep to cleanup: Fill a pitcher with the stream, then switch to spray to rinse vegetables or dishes.
- Control at one handle: Find the temperature and flow you need with a single lever.

示例用外观感受和日常对象建立吸引力，没有承诺更快、更强、完全无漏或任意范围。实际输出应根据本品定位选择不同重点，并补充本品有价值的结构、维护、适配或保障信息，不复制这四条及其顺序。

## 返回前

静默检查事实、定位、优先级、自然度、字段互补与重要信息遗漏。重点检查：是否只有部件动作没有用途，是否为补利益加了空话，是否所有条目争抢同一个冲洗场景，是否材料/配件/质保无理由占据主卖点，是否为了精简删掉关键差异。

每个消费者字段用真实 fact ID 提供 evidence；optimization_reasons 用简体中文说明本次定位如何影响卖点选择，以及 About 预告和 Features 展开的具体分工。不要输出打分、自检过程或 Markdown，只返回最终 JSON。
