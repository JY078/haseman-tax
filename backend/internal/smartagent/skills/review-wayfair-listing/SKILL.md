---
name: review-wayfair-listing
description: Review a generated Wayfair listing against confirmed facts and the current natural-copy rules.
---

# Wayfair Listing Reviewer

审核 Writer 实际输出的 `about_this_product`、`features`，并核对 supplied fingerprint；reviewed_title 返回空字符串。不要改写 Listing，只返回 Runtime Contract 要求的审核 JSON。

## 审核规则

下文的“场景与收益”指清楚的实际用途或购买价值，不要求每条同时写出动作、结果和机制；适配、维护及准确保障信息按各自购买问题判断。此解释适用于以下所有价值检查。

按共享写作标准区分产品事实、日常用途、审美情绪和性能承诺。允许由已确认功能转译常见任务，允许有外观依据的温暖、协调、现代感；不要求资料逐字出现相同场景或情绪词。指出越界时必须引用具体新增的规格、能力或承诺，不能只用“缺乏证据”删除正常零售表达。修复建议只收窄有问题的承诺，保留原句有效的画面与节奏；不得把自然文案统一改为更短的部件说明。

事实正确只是底线。检查 About 是否体现本品的吸引力，核心 Feature 是否让消费者看见用途或购买价值；若整段连续发操作指令、反复陈述部件动作，应指出具体句子并要求自然衔接。不要因没有情绪形容词扣分，也不要把加入形容词本身当成质量提升。

重点拦截以下机械表达：`general sink tasks`、`compatible setups`、`centralized faucet operation`、`planning water coverage`、`streamlined configuration`。要求改成具体对象、动作和结果；但不得因为文案使用 `warm accent`、`welcoming`、`coordinated` 等有外观依据的审美表达而扣分。

- 两个字段必须存在，产品声明必须与 `confirmed_facts` 一致，并且有真实 fact ID 依据。
- 若输入提供价格段、产品定位、目标人群或购买任务，检查文案是否服务于该策略：基础替换型应回答替换适配与安装顾虑，大众性价比型应突出核心任务和维护价值，品牌改善型应突出体验、空间控制、耐用细节和可信保障。定位只能改变排序与叙事，不能成为无事实依据的性能声明。
- 若输入明确定位，按写作技能中的定位顺序审核 Features：基础替换型为外观→高弧空间→双模式→抽拉范围→单把手→前向手柄→易清洁→阀芯→孔位/底板→进水管→认证质保；大众性价比型为商用外观→高弧旋转→喷淋/用水状态→回位/抽拉→易清洁→表面→皂液器→安装→快接→质保；品牌改善型为顺滑抽拉→磁吸→触达→增压→重型结构→节水→易清洁→防漏→表面→质保。只检查已确认事实对应的位置，缺失事实可以跳过。
- 不审核或要求 Title；允许本 SKU 有依据的颜色和 Finish 描述。
- About This Product 应连贯介绍产品定位、设计和重点购买价值，不强制从操作任务或厨房故事开头，不得出现内部审核、平台决策或元话术。
- Features 可以是短语或完整句子，数量、长度和结尾标点不设固定限制；检查每条是否提供具体功能及使用场景或消费者收益。
- 对核心功能检查可理解的购买价值：只有部件动作、同义解释或孤立参数而让读者无法理解用途时指出问题；日常任务、调节对象、适配答案已经清楚时不强制再补收益从句。不要把所有功能都归结为“冲洗餐具”；相邻条目应有各自重点。
- 检查正文是否具有自然美式口语可读性：使用常见词、主动语态和有变化的句式，不按参数表顺序罗列，不连续堆叠名词。
- 证据来源不能出现在消费者文案中：`confirmed`、`specified as`、`listed as`、`according to the product information`、`the product is identified as`、`supports the confirmed` 命中时，要求改成直接事实或用户结果，而不是继续保留资料转述腔。
- 允许整组使用一致的功能标题和产品主语；审核正文是否将事实转成具体用途或购买价值，不因标签、祈使句或句型形式判错。About 介绍定位和重点利益，Features 增加证据与细节；同一主题可以预告再展开。
- 场景必须是消费者能想象的具体动作或对象；`different tasks`、`routine use`、`where it is needed`、`around the sink` 等泛化占位词不能代替实际场景。
- 参数本身不等于消费者价值。每个重点卖点应形成 `确认事实 -> 使用情境/动作 -> 具体结果`；只有裸参数、功能名称或抽象形容词时应判为未完成价值表达。
- 拒绝用 `supports/provides/helps/offers + 抽象名词` 冒充 Benefit，以及只写 `convenient`、`versatile`、`comfortable`、`efficient` 而没有具体结果的文案。
- 拒绝 `fit information`、`purchase decision`、`helps shoppers decide`、`search recognition`、`switch tasks`、`useful working room`、`where needed` 等生硬或内部表达。
- 检查重点卖点是否形成 `可搜索功能/结构名称 -> 已确认机制或证据 -> 使用动作/场景 -> 具体收益`。不强制每条包含四段，但只有功能名称、配件清单、认证编号或泛泛收益时不能通过质量审核。
- confirmed facts 中存在回缩方式、旋转接头、配重、快接、预装部件、操作角度、测试次数或认证等可验证细节时，应优先用最相关细节支撑卖点；不得借用竞品技术名或数据。
- 检查 Features 是否按消费者影响、证据强度和差异化排序。条数不设限制，但不能为覆盖全部输入事实而把低价值配件、认证或参数拆成独立条目；标签后的正文不能只是同义重复标签。
- 低优先级事实应合并：工具清单不应独占核心 Feature；尺寸只在回答空间/安装顾虑时保留；认证和质保应合并为准确的保障信息；材质与阀芯需要连接到已确认的结构价值，否则删除或并入相关条目。
- 拒绝把“包含 deck plate、supply line 或 mounting hardware”直接推导为安装简单；只有已确认的快接、预装或其他安装机制才能支撑减少安装步骤等表述。
- 拒绝由事实名称补造机制或性能：`stain resistant` 不能自行改写为 `treated surface`，`ceramic disc cartridge` 不能自行增加测试次数或完全防漏，普通 included components 不能自行变成 quick installation。
- 拒绝 `without relying only on...` 等负向绕行，以及 `supports a straightforward setup`、`provides key parts for installing and using`、`to match the faucet assembly`、无证据的 `premium components` 或 `reliable performance` 等弱结论。包装清单和认证信息只有连接到具体安装、使用、维护、安全或适配收益时才可保留。
- About This Product 应建立整体生活价值并连贯组织少量重点；Features 应展开不同机制和收益，不能逐句复述 About。
- 以高质量文案标准复核：指出不增加信息的开场、结尾和重复 Feature；About 应连贯，Features 应有独立购买价值。具体动作已说明用途时不必追加收益从句，适配、维护和准确质保信息也不强制套用动作模板。`routine use`、`everyday use`、`makes it easier` 等泛化短语不能单独充当收益。
- 审核只返回问题与可执行的修复方向，不生成替代 Listing；引用原句，说明事实或表达问题，由调用方将建议交给 Writer 修复。
- 按写作技能的同一优先级审核：默认设计定位、结构与空间适配、操作、维护、安装、保障，再按本品定位和购买任务调整。不得强制操作任务总排第一，也不得把有证据的结构品质一律降为补充信息。
- 当 high-arc、材料/无铅或质保等事实已确认且能回答空间、信任或保障顾虑时，检查其是否被不当地全部删除。无需机械要求固定条目，但不得为了缩短文案而只留下操作功能。
- 只审核实际输出，不推测 Writer 的内部思考或保证调用次数；拒绝把表单字段顺序、认证清单或内部字段状态直接当成消费者文案。
- 不得编造隐藏规格、尺寸、性能、配件、认证或其他产品事实。

不要使用字符数、词数、句数、Feature 条数、短语形式或结尾标点作为通过或失败依据。

## 表达一致性检查

- 专业度：功能与收益的因果归属正确；不把高弧高度归因于旋转，不从材料名称推导防漏、稳定控水、认证或安全，不用 `is listed as` 掩盖属性归属不明。
- 本土化：指出 `steady filling`、模糊的 `adjust the faucet` 和说明书式部件罗列，要求用已确认的具体任务或调节对象表达。
- 逻辑性：About 应连贯介绍产品，Features 展开不同购买理由；允许 About 预告同一主题，Feature 补充机制或细节，拒绝没有新增信息的整条换词复述。
- 易懂性：检查 `broader working area`、`where you need it` 等是否只是空泛收益；已有具体动作时不因出现某个常用词组机械判错。分号或长句只有造成卖点堆叠、歧义或阅读负担时才作为表达问题，不能仅按标点扣分。
- 机械英文：`brings a sprayer to the sink`、`change the water direction`、`by moving the spout`、`at a single control point` 等仅解释装置的句子需要改为直接的零售消费者表达；例如已确认单手柄调温调流时，应说明调节对象而不是重复“一个控制点”。
- 把语法正确但仍像产品说明书的文案判为可读性问题；优先要求它改成自然动作与结果，不要只将一个技术动词替换成另一个技术动词。
