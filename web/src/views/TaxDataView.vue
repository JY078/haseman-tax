<template>
  <MainLayout>
    <div class="tax-page">
      <section class="panel">
        <div class="platform-toolbar">
          <div class="platform-picker">
            <Listbox :model-value="platform" @update:model-value="switchPlatform" as="template" v-slot="{ open }">
              <ListboxButton class="platform-trigger" :class="{ open }" aria-label="切换报税平台">
                <span class="platform-label">平台</span><strong>{{platformOptions.find(item => item.value ===
                  platform)?.label }}</strong>
                <svg class="platform-chevron" :class="{ open }" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                  <path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"
                    stroke-linejoin="round" />
                </svg>
              </ListboxButton>
              <div class="platform-menu" v-if="open">
                <ListboxOptions class="platform-options" static>
                    <ListboxOption v-for="item in availablePlatformOptions" :key="item.value" :value="item.value" as="template"
                    v-slot="{ active, selected }">
                    <li class="platform-option" :class="{ highlighted: active, selected }"><span>{{ item.label
                        }}</span><svg class="platform-selected-icon" :class="{ visible: selected }" viewBox="0 0 16 16"
                        fill="none" aria-hidden="true">
                        <path d="m3.5 8 3 3 6-6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
                          stroke-linejoin="round" />
                      </svg></li>
                  </ListboxOption>
                </ListboxOptions>
              </div>
            </Listbox>
            <Listbox v-model="countryId" @update:model-value="switchCountry" as="template" v-slot="{ open }">
              <div class="store-picker site-picker"><span class="store-label">国家</span>
                <ListboxButton class="store-trigger" :class="{ open }"><strong>{{ countryLabel }}</strong><svg class="platform-chevron" :class="{ open }" viewBox="0 0 16 16"
                    fill="none" aria-hidden="true">
                    <path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"
                      stroke-linejoin="round" />
                  </svg></ListboxButton>
                <div v-if="open" class="store-menu">
                  <ListboxOptions class="store-options" static>
                    <ListboxOption value="" as="template" v-slot="{ active, selected }">
                      <li class="store-option" :class="{ highlighted: active, selected }"><span>全部</span><svg class="store-selected-icon" :class="{ visible: selected }" viewBox="0 0 16 16"
                          fill="none" aria-hidden="true">
                          <path d="m3.5 8 3 3 6-6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
                            stroke-linejoin="round" />
                        </svg></li>
                    </ListboxOption>
                    <ListboxOption v-for="site in siteOptions" :key="site.value" :value="site.value" as="template"
                      v-slot="{ active, selected }">
                      <li class="store-option" :class="{ highlighted: active, selected }"><span>{{ site.label
                          }}</span><svg class="store-selected-icon" :class="{ visible: selected }" viewBox="0 0 16 16"
                          fill="none" aria-hidden="true">
                          <path d="m3.5 8 3 3 6-6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
                            stroke-linejoin="round" />
                        </svg></li>
                    </ListboxOption>
                  </ListboxOptions>
                </div>
              </div>
            </Listbox>
            <Listbox v-model="storeId" @update:model-value="switchStore" as="template" v-slot="{ open }">
              <div class="store-picker"><span class="store-label">店铺</span>
                <ListboxButton class="store-trigger" :class="{ open }"><strong>{{ storeLabel }}</strong><svg class="platform-chevron" :class="{ open }" viewBox="0 0 16 16"
                    fill="none" aria-hidden="true">
                    <path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"
                      stroke-linejoin="round" />
                  </svg></ListboxButton>
                <div v-if="open" class="store-menu">
                  <ListboxOptions class="store-options" static>
                    <ListboxOption value="" as="template" v-slot="{ active, selected }">
                      <li class="store-option" :class="{ highlighted: active, selected }"><span>全部</span><svg class="store-selected-icon" :class="{ visible: selected }" viewBox="0 0 16 16"
                          fill="none" aria-hidden="true">
                          <path d="m3.5 8 3 3 6-6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
                            stroke-linejoin="round" />
                        </svg></li>
                    </ListboxOption>
                    <ListboxOption v-for="store in storeOptions" :key="store.value" :value="store.value" as="template"
                      v-slot="{ active, selected }">
                      <li class="store-option" :class="{ highlighted: active, selected }"><span>{{ store.label
                          }}</span><svg class="store-selected-icon" :class="{ visible: selected }" viewBox="0 0 16 16"
                          fill="none" aria-hidden="true">
                          <path d="m3.5 8 3 3 6-6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
                            stroke-linejoin="round" />
                        </svg></li>
                    </ListboxOption>
                  </ListboxOptions>
                </div>
              </div>
            </Listbox>
            <TaxDateRangePicker v-model:from="dateFrom" v-model:to="dateTo" label="范围" @change="switchPeriod" />
            <button v-if="canFinance('create', platform)" class="catalog-add-button" type="button" @click="openCatalogDialog()">添加店铺/站点</button>
            <button v-if="canFinance('import', platform)" class="import-trigger" type="button" @click="openImportDialog">导入数据</button>
          </div>
          <div class="history-actions"><button @click="openHistory('imports')">导入日志</button><button
              @click="openHistory('downloads')">下载记录</button></div>
        </div>
        <AmazonTaxTables v-if="platform === 'amazon'" :store-id="storeId" :site-id="countryId" :country-options="siteOptions"
          :site-currency-code="selectedSiteCurrency" :download-busy="downloadBusy" :download-error="downloadError"
          :download-file="downloadFile" :date-from="dateFrom" :date-to="dateTo" :can-export="canFinance('export', 'amazon')" />
        <template v-else-if="platform === 'tiktok'">
          <div class="toolbar tiktok-toolbar">
            <div class="tabs" role="tablist" aria-label="TikTok 报税数据表">
              <button id="tiktok-details-tab" role="tab" aria-controls="tiktok-table"
                :aria-selected="tiktokTab === 'details'" :class="{ active: tiktokTab === 'details' }"
                @click="switchTikTokTab('details')">涉税申报明细</button>
              <button id="tiktok-ads-tab" role="tab" aria-controls="tiktok-table" :aria-selected="tiktokTab === 'ads'"
                :class="{ active: tiktokTab === 'ads' }" @click="switchTikTokTab('ads')">广告费</button>
              <button id="tiktok-fbt-tab" role="tab" aria-controls="tiktok-table" :aria-selected="tiktokTab === 'fbt'"
                :class="{ active: tiktokTab === 'fbt' }" @click="switchTikTokTab('fbt')">FBT</button>
              <button id="tiktok-settlements-tab" role="tab" aria-controls="tiktok-table"
                :aria-selected="tiktokTab === 'settlements'" :class="{ active: tiktokTab === 'settlements' }"
                @click="switchTikTokTab('settlements')">结算单</button>
              <button id="tiktok-orders-tab" role="tab" aria-controls="tiktok-table"
                :aria-selected="tiktokTab === 'orders'" :class="{ active: tiktokTab === 'orders' }"
                @click="switchTikTokTab('orders')">订单</button>
              <button id="tiktok-quarters-tab" role="tab" aria-controls="tiktok-table"
                :aria-selected="tiktokTab === 'quarters'" :class="{ active: tiktokTab === 'quarters' }"
                @click="switchTikTokTab('quarters')">季度汇总</button>
            </div>
            <div class="filters"><button v-if="['settlements', 'orders'].includes(tiktokTab)"
                class="column-config-trigger" @click="tiktokColumnConfigOpen = true">列配置</button><button
                :disabled="tiktokLoading" @click="loadTikTok">刷新</button><button v-if="canFinance('export', 'tiktok')" class="export-btn"
                :disabled="tiktokLoading || !!downloadBusy" @click="exportTikTokReport">{{ downloadBusy === 'export' ?
                '生成中…' : '下载报表' }}</button></div>
          </div>
          <TaxColumnConfigPanel v-if="['settlements', 'orders'].includes(tiktokTab)" :open="tiktokColumnConfigOpen"
            :headers="tiktokConfigHeaders" :visible="tiktokConfigVisible" :pinned="tiktokConfigPinned"
            :fixed-count="tiktokTab === 'settlements' ? 5 : 2" @cancel="tiktokColumnConfigOpen = false"
            @apply="applyTikTokColumnConfig" />
          <p v-if="downloadError" class="error" role="alert">{{ downloadError }}</p>
          <div v-if="tiktokTab === 'quarters'" class="summary-toolbar">
            <div><strong>季度汇总</strong><span>按日期范围查看季度数据</span></div>
            <div class="summary-controls"><div v-if="countryId" class="currency-segment" role="group" aria-label="TikTok 汇总币种"><button
                :aria-pressed="tiktokCurrency === 'CNY'" :class="{ active: tiktokCurrency === 'CNY' }"
                @click="switchTikTokCurrency('CNY')">人民币</button><button :aria-pressed="tiktokCurrency === 'USD'"
                :class="{ active: tiktokCurrency === 'USD' }" @click="switchTikTokCurrency('USD')">美元</button></div></div>
          </div>
          <div id="tiktok-table" class="tiktok-table-region" role="tabpanel"
            :aria-labelledby="`tiktok-${tiktokTab}-tab`" :aria-busy="tiktokLoading">
            <div v-if="tiktokLoading" class="tiktok-loading" role="status"><span>正在加载数据…</span></div>
            <p v-if="tiktokError" class="error" role="alert">{{ tiktokError }}</p>
            <div v-else :key="tiktokTab" class="table-scroll tiktok-scroll" tabindex="0"
              :aria-label="`TikTok ${reportLabel(tiktokTab)}`">
              <table v-if="tiktokTab === 'details'" class="tiktok-table tiktok-details">
                <colgroup>
                  <col class="tk-month" />
                  <col class="tk-quarter" />
                  <col class="tk-date" />
                  <col class="tk-business" />
                  <col class="tk-amount" />
                  <col class="tk-amount" />
                  <col class="tk-commission" />
                </colgroup>
                <thead>
                  <tr>
                    <th scope="col" class="tk-month">月份<button class="filter-trigger"
                        :class="{ active: !!currentTikTokFilters.month }" :aria-label="'筛选月份'"
                        :aria-expanded="tiktokFilterField === 'month'"
                        @click.stop="toggleTikTokFilter('month', $event)">⌕</button></th>
                    <th scope="col" class="tk-quarter">季度<button class="filter-trigger"
                        :class="{ active: !!currentTikTokFilters.quarter }" :aria-label="'筛选季度'"
                        :aria-expanded="tiktokFilterField === 'quarter'"
                        @click.stop="toggleTikTokFilter('quarter', $event)">⌕</button></th>
                    <th scope="col">交易入账时间</th>
                    <th scope="col">业务单号</th>
                    <th scope="col" class="numeric">收入金额</th>
                    <th scope="col" class="numeric">退款金额</th>
                    <th scope="col" class="numeric">平台佣金/服务费金额</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in tiktokDetails" :key="row.id">
                    <td class="tk-month">{{ row.month }}</td>
                    <td class="tk-quarter">{{ row.quarter }}</td>
                    <td>{{ formatDateOnly(row.posted_at) }}</td>
                    <td :title="row.business_id">{{ row.business_id }}</td>
                    <td class="numeric">${{ money(row.income) }}</td>
                    <td class="numeric">${{ money(row.refund) }}</td>
                    <td class="numeric">${{ money(row.commission) }}</td>
                  </tr>
                  <tr v-if="!tiktokLoading && !tiktokDetails.length">
                    <td colspan="7" class="empty">暂无数据</td>
                  </tr>
                </tbody>
                <tfoot v-if="tiktokDetails.length">
                  <tr>
                    <th colspan="4">总计</th>
                    <td class="numeric">{{ totalMoney(tiktokTotals.details, 'income', 'USD') }}</td>
                    <td class="numeric">{{ totalMoney(tiktokTotals.details, 'refund', 'USD') }}</td>
                    <td class="numeric">{{ totalMoney(tiktokTotals.details, 'commission', 'USD') }}</td>
                  </tr>
                </tfoot>
              </table>
              <table v-else-if="tiktokTab === 'ads'" class="tiktok-table tiktok-ads">
                <colgroup>
                  <col style="width: 80px" />
                  <col style="width: 86px" />
                  <col style="width: 130px" />
                  <col style="width: 110px" />
                  <col style="width: 220px" />
                  <col style="width: 190px" />
                  <col style="width: 140px" />
                  <col style="width: 150px" />
                  <col style="width: 170px" />
                  <col style="width: 140px" />
                  <col style="width: 130px" />
                </colgroup>
                <thead>
                  <tr>
                    <th scope="col">月份<button class="filter-trigger" :class="{ active: !!currentTikTokFilters.month }"
                        :aria-label="'筛选月份'" :aria-expanded="tiktokFilterField === 'month'"
                        @click.stop="toggleTikTokFilter('month', $event)">⌕</button></th>
                    <th scope="col">季度<button class="filter-trigger" :class="{ active: !!currentTikTokFilters.quarter }"
                        :aria-label="'筛选季度'" :aria-expanded="tiktokFilterField === 'quarter'"
                        @click.stop="toggleTikTokFilter('quarter', $event)">⌕</button></th>
                    <th scope="col" class="numeric">费用</th>
                    <th scope="col">日期</th>
                    <th scope="col">系列名称</th>
                    <th scope="col">推广系列ID</th>
                    <th scope="col" class="numeric">现金消耗</th>
                    <th scope="col" class="numeric">信用额度消耗</th>
                    <th scope="col" class="numeric">广告费用赠款消耗</th>
                    <th scope="col" class="numeric">金额</th>
                    <th scope="col">类型</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in tiktokAds" :key="row.id">
                    <td>{{ row.month }}</td>
                    <td>{{ row.quarter }}</td>
                    <td class="numeric"><span v-if="row.fee == null" class="rate-pending">待换算</span><template v-else>¥{{
                        money(row.fee) }}</template>
                    </td>
                    <td>{{ row.date }}</td>
                    <td :title="row.campaign_name">{{ row.campaign_name }}</td>
                    <td :title="row.campaign_id">{{ row.campaign_id }}</td>
                    <td class="numeric">${{ money(row.cash_spend) }}</td>
                    <td class="numeric">${{ money(row.credit_spend) }}</td>
                    <td class="numeric">${{ money(row.grant_spend) }}</td>
                    <td class="numeric">${{ money(row.amount) }}</td>
                    <td :title="row.type">{{ row.type }}</td>
                  </tr>
                  <tr v-if="!tiktokLoading && !tiktokAds.length">
                    <td colspan="11" class="empty">暂无数据</td>
                  </tr>
                </tbody>
                <tfoot v-if="tiktokAds.length">
                  <tr>
                    <th colspan="2">总计</th>
                    <td class="numeric">{{ totalMoney(tiktokTotals.ads, 'fee') }}</td>
                    <th colspan="3"></th>
                    <td class="numeric">{{ totalMoney(tiktokTotals.ads, 'cash_spend', 'USD') }}</td>
                    <td class="numeric">{{ totalMoney(tiktokTotals.ads, 'credit_spend', 'USD') }}</td>
                    <td class="numeric">{{ totalMoney(tiktokTotals.ads, 'grant_spend', 'USD') }}</td>
                    <td class="numeric">{{ totalMoney(tiktokTotals.ads, 'amount', 'USD') }}</td>
                    <th></th>
                  </tr>
                </tfoot>
              </table>
              <table v-else-if="tiktokTab === 'fbt'" class="tiktok-table tiktok-fbt">
                <thead>
                  <tr>
                    <th>月份<button class="filter-trigger" :class="{ active: !!currentTikTokFilters.month }"
                        @click.stop="toggleTikTokFilter('month', $event)">⌕</button></th>
                    <th>季度<button class="filter-trigger" :class="{ active: !!currentTikTokFilters.quarter }"
                        @click.stop="toggleTikTokFilter('quarter', $event)">⌕</button></th>
                    <th>Goods ID（商品ID）</th>
                    <th>SKU ID（SKU）</th>
                    <th>Product ID（产品ID）</th>
                    <th>Product Name（产品名称）</th>
                    <th>Shop ID（店铺ID）</th>
                    <th>Shop Name（店铺名称）</th>
                    <th>Cost Rate（成本率）</th>
                    <th>GMV（成交总额）</th>
                    <th>Total Logistics Cost（物流总成本）</th>
                    <th>FBT Fulfillment Fee（履约费）</th>
                    <th>FBT Fulfillment Fee Reimbursement（履约费返还）</th>
                    <th>Shipping Fee Discount（运费折扣）</th>
                    <th>Hub Placement Fee（枢纽安置费）</th>
                    <th>Storage Fee（仓储费）</th>
                    <th>Inbound Shipping Fee（入库运输费）</th>
                    <th>Inbound Incidents Fee（入库事件费）</th>
                    <th>Inbound Booking Non-Compliance Fee（入库预约不合规费）</th>
                    <th>Routing Non-Compliance Fee（路线不合规费）</th>
                    <th>Outbound Appointment No-Show Fee（出库预约未到费）</th>
                    <th>Delayed Response Fee（延迟响应费）</th>
                    <th>Disposal Fee（处置费）</th>
                    <th>Return to Seller Shipping Fee（退回卖家运输费）</th>
                    <th>Return to Seller Handling Fee（退回卖家处理费）</th>
                    <th>Inbound Return Operation Fee（入库退货操作费）</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in tiktokFBT" :key="row.id">
                    <td>{{ row.month }}</td>
                    <td>{{ row.quarter }}</td>
                    <td>{{ row.goodsId }}</td>
                    <td>{{ row.skuId }}</td>
                    <td>{{ row.productId }}</td>
                    <td>{{ row.productName }}</td>
                    <td>{{ row.shopId }}</td>
                    <td>{{ row.shopName }}</td>
                    <td class="numeric">{{ fbtRate(row.costRate) }}</td>
                    <td v-for="key in fbtAmountKeys.slice(1)" :key="key" class="numeric">${{ money(row[key]) }}</td>
                  </tr>
                  <tr v-if="!tiktokLoading && !tiktokFBT.length">
                    <td colspan="26" class="empty">暂无数据</td>
                  </tr>
                </tbody>
                <tfoot v-if="tiktokFBT.length">
                  <tr>
                    <th colspan="9">总计</th>
                    <td v-for="key in fbtAmountKeys.slice(1)" :key="key" class="numeric">{{ fbtTotal(key) }}</td>
                  </tr>
                </tfoot>
              </table>
              <table v-else-if="tiktokTab === 'settlements'" class="tiktok-table tiktok-settlements"
                :style="{ width: `${526 + displayedSettlementColumns.length * 180}px` }">
                <colgroup>
                  <col style="width:80px" />
                  <col style="width:86px" />
                  <col style="width:110px" />
                  <col style="width:110px" />
                  <col style="width:140px" />
                  <col v-for="column in displayedSettlementColumns" :key="column.key" style="width:180px" />
                </colgroup>
                <thead>
                  <tr>
                    <th v-for="field in ['month', 'quarter']" :key="field" scope="col">{{ field === 'month' ? '月份' :
                      '季度' }}<button class="filter-trigger" :class="{ active: !!currentTikTokFilters[field] }"
                        :aria-label="field === 'month' ? '筛选月份' : '筛选季度'" :aria-expanded="tiktokFilterField === field"
                        @click.stop="toggleTikTokFilter(field, $event)">⌕</button></th>
                    <th v-for="field in ['logistics_method', 'order_type']" :key="field" scope="col">{{
                      tikTokFilterLabels[field]
                      }}<button class="filter-trigger" :class="{ active: !!currentTikTokFilters[field] }"
                        :aria-label="`筛选${tikTokFilterLabels[field]}`" :aria-expanded="tiktokFilterField === field"
                        @click.stop="toggleTikTokFilter(field, $event)">⌕</button></th>
                    <th scope="col" class="numeric">费用</th>
                    <th v-for="column in displayedSettlementColumns" :key="column.key" scope="col"
                      :title="`${column.name || column.key}(${column.label})`">{{ column.name || column.key }}({{
                      column.label }})
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in tiktokSettlements" :key="row.id">
                    <td>{{ row.month }}</td>
                    <td>{{ row.quarter }}</td>
                    <td>{{ row.logistics_method || '—' }}</td>
                    <td>{{ row.order_type || '—' }}</td>
                    <td class="numeric"><span v-if="row.fee == null" class="rate-pending">待换算</span><template v-else>¥{{
                        money(row.fee) }}</template></td>
                    <td v-for="column in displayedSettlementColumns" :key="column.key"
                      :class="{ numeric: settlementAmountKeys.has(column.key) }"
                      :title="settlementCellValue(row, column.key)">{{
                        settlementCellValue(row, column.key) }}</td>
                  </tr>
                  <tr v-if="!tiktokLoading && !tiktokSettlements.length">
                    <td :colspan="displayedSettlementColumns.length + 5" class="empty">暂无数据</td>
                  </tr>
                </tbody>
                <tfoot v-if="tiktokSettlements.length">
                  <tr>
                    <th colspan="4">总计</th>
                    <td class="numeric">{{ totalMoney(tiktokTotals.settlements, 'fee', 'CNY') }}</td>
                    <td v-for="column in displayedSettlementColumns" :key="column.key"
                      :class="{ numeric: settlementAmountKeys.has(column.key) }">{{ settlementAmountKeys.has(column.key)
                        ?
                        (settlementQuantityKeys.has(column.key) ? totalQuantity(tiktokTotals.settlements, column.key) :
                      totalMoney(tiktokTotals.settlements, column.key, 'USD')) : '' }}</td>
                  </tr>
                </tfoot>
              </table>
              <table v-else-if="tiktokTab === 'orders'" class="tiktok-table tiktok-orders"
                :style="{ width: `${166 + displayedOrderColumns.length * 180}px` }">
                <colgroup>
                  <col style="width:80px" />
                  <col style="width:86px" />
                  <col v-for="column in displayedOrderColumns" :key="column.key" style="width:180px" />
                </colgroup>
                <thead>
                  <tr>
                    <th v-for="field in ['month', 'quarter']" :key="field" scope="col">{{ field === 'month' ? '月份' :
                      '季度' }}<button class="filter-trigger" :class="{ active: !!currentTikTokFilters[field] }"
                        :aria-label="field === 'month' ? '筛选月份' : '筛选季度'" :aria-expanded="tiktokFilterField === field"
                        @click.stop="toggleTikTokFilter(field, $event)">⌕</button></th>
                    <th v-for="column in displayedOrderColumns" :key="column.key" scope="col"
                      :title="`${column.key}(${column.label})`">{{ column.key }}({{ column.label }})</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in tiktokOrders" :key="row.id">
                    <td>{{ row.month }}</td>
                    <td>{{ row.quarter }}</td>
                    <td v-for="column in displayedOrderColumns" :key="column.key" :title="row.data[column.key]">{{
                      row.data[column.key] ?? '' }}</td>
                  </tr>
                  <tr v-if="!tiktokLoading && !tiktokOrders.length">
                    <td :colspan="displayedOrderColumns.length + 2" class="empty">暂无数据</td>
                  </tr>
                </tbody>
                <tfoot v-if="tiktokOrders.length">
                  <tr>
                    <th colspan="2">总计</th>
                    <td v-for="column in displayedOrderColumns" :key="column.key"
                      :class="{ numeric: !isExcludedTaxTotalColumn(column.key) }">{{ orderTotalValue(column.key) }}</td>
                  </tr>
                </tfoot>
              </table>
              <table v-else class="tiktok-table tiktok-summary">
                <colgroup>
                  <col style="width: 120px" />
                  <col v-for="key in tiktokSummaryKeys" :key="key"
                    :style="key === 'delivery' ? { width: '190px' } : undefined" />
                </colgroup>
                <thead>
                  <tr>
                    <th scope="col">季度<button class="filter-trigger" :class="{ active: !!currentTikTokFilters.quarter }"
                        @click.stop="toggleTikTokFilter('quarter', $event)">⌕</button></th>
                    <th v-for="column in tiktokSummaryColumns" :key="column.key" scope="col" class="numeric"><span
                        class="summary-header-label">{{ column.label }}
                        <TaxFormulaHelp v-if="column.formula" :label="column.label" :formula="column.formula" />
                      </span></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in filteredTikTokQuarters" :key="row.quarter">
                    <td>{{ row.quarter }}</td>
                    <td v-for="column in tiktokSummaryColumns" :key="column.key" class="numeric">
                      <TaxTransactionCountCell v-if="column.key === 'transaction_count'" :row="row" />
                      <TaxFBTFeeCell v-else-if="column.key === 'fbt_fee'" :row="row" :currency="tiktokCurrency" />
                      <TaxCommissionCell v-else-if="column.key === 'commission'" :row="row"
                        :currency="tiktokCurrency" />
                      <TaxAffiliateCommissionCell v-else-if="column.key === 'affiliate_commission'" :row="row"
                        :currency="tiktokCurrency" />
                      <TaxSummaryAmountCell v-else-if="column.key === 'sales_amount'" :value="row.income"
                        :detail-value="row.sales_amount" detail-label="结算单销售额"
                        :details="[{ label: '运费合计（客户运费抵扣 + 平台运费折扣 + 客户运费 + 退款客户运费）', value: row.shipping_fee }, { label: '客户运费抵扣', value: row.shipping_customer_offset }, { label: '平台运费折扣', value: row.shipping_platform_discount }, { label: '客户运费', value: row.shipping_customer }, { label: '退款客户运费', value: row.shipping_refunded_customer }]"
                        :currency="tiktokCurrency" />
                      <TaxSummaryAmountCell v-else-if="column.key === 'refund'" :value="row.refund"
                        :detail-value="row.settlement_refund_subtotal" detail-label="退款明细"
                        :details="[{ label: '退款合计', value: row.settlement_refund_subtotal }]"
                        :currency="tiktokCurrency" />
                      <TaxSummaryAmountCell v-else-if="column.key === 'other_fees'" :value="row[column.key]"
                        :detail-value="row.settlement_other_fees" detail-label="其他费用"
                        :details="[{ label: '签名确认服务费', value: row.settlement_other_fees_signature_confirmation }, { label: '共担促销费', value: row.settlement_other_fees_cofunded_promotion }, { label: '配送保障服务费', value: row.settlement_other_fees_shipping_protection }, { label: '负余额扣款', value: row.settlement_other_fees_negative_balance_deduction }, { label: '负余额销售收入', value: row.settlement_other_fees_negative_balance_sales }]"
                        :pending-text="column.pending ? '待计算' : '待换算'" :currency="tiktokCurrency" />
                      <TaxSummaryAmountCell v-else-if="column.key === 'campaign_service_fee'" :value="row[column.key]"
                        :detail-value="row.settlement_campaign_service_fee" detail-label="活动服务费"
                        :details="[{ label: '活动服务费', value: row.settlement_campaign_service_fee_item }, { label: '附加活动套餐', value: row.settlement_additional_campaign_package }, { label: '活动套餐', value: row.settlement_campaign_package }]"
                        :currency="tiktokCurrency" />
                      <TaxSummaryAmountCell v-else-if="column.key === 'smart_promotion_fee'" :value="row[column.key]"
                        :detail-value="row.settlement_smart_promotion_fee" detail-label="智能营销费"
                        :details="[{ label: '智能营销费', value: row.settlement_smart_promotion }, { label: '智能营销周期费', value: row.settlement_smart_promotion_period }]"
                        :pending-text="column.pending ? '待计算' : '待换算'" :currency="tiktokCurrency" />
                      <TaxSummaryAmountCell v-else-if="column.key === 'tax'" :value="row[column.key]"
                        :detail-value="row.settlement_tax" detail-label="税费"
                        :details="[{ label: '销售税', value: row.settlement_tax_sales }, { label: '销售税付款', value: row.settlement_tax_sales_payment }, { label: '销售税退款', value: row.settlement_tax_sales_refund }, { label: '零售配送费', value: row.settlement_tax_retail_delivery }, { label: '零售配送费付款', value: row.settlement_tax_retail_payment }, { label: '零售配送费退款', value: row.settlement_tax_retail_refund }]"
                        :currency="tiktokCurrency" />
                      <span v-else-if="column.key === 'expense_total'">{{ tiktokQuarterTotalExpenses(row) }}</span><TaxSummaryAmountCell v-else-if="column.settlementOnly" :value="row[column.key]"
                        :detail-value="row[`settlement_${column.key}`]"
                        :details="[{ label: '明细金额', value: row[`settlement_${column.key}`] }]" detail-label="明细"
                        :pending-text="column.pending ? '待计算' : '待换算'" :currency="tiktokCurrency" /><template
                        v-else-if="column.key === 'taxable_income'"><span>{{ summaryMoney(row[column.key],
                          tiktokCurrency) }}</span><small class="summary-subformula">{{ incomeMinusRefundLabel(row)
                          }}</small></template><template v-else><span v-if="row[column.key] == null"
                          :class="column.pending ? 'summary-pending' : 'rate-pending'">{{ column.pending ? '待计算' : '待换算'
                          }}</span><span v-else>{{ summaryMoney(row[column.key], tiktokCurrency) }}</span></template>
                    </td>
                  </tr>
                  <tr v-if="!tiktokLoading && !filteredTikTokQuarters.length">
                    <td :colspan="tiktokSummaryKeys.length + 2" class="empty">暂无数据</td>
                  </tr>
                </tbody>
                <tfoot v-if="filteredTikTokQuarters.length">
                  <tr>
                    <th>总计</th>
                    <td v-for="column in tiktokSummaryColumns" :key="column.key" class="numeric"><span>{{column.key ===
                      'transaction_count' ? `${totalValue(filteredTikTokQuarters, row =>
                        row.transaction_count).toLocaleString('zh-CN')}笔` : tiktokSummaryTotal(column.key)
                        }}</span><small v-if="column.key === 'taxable_income'" class="summary-subformula">收入金额 − 退款 {{
                          summaryMoney(totalValue(filteredTikTokQuarters, row => row.income) -
                            totalValue(filteredTikTokQuarters, row =>
                        row.refund), tiktokCurrency) }}</small><small
                        v-else-if="column.key !== 'fbt_fee' && tiktokSummaryDetailTotal(column.key) != null"
                        class="summary-subformula">明细合计 {{ tiktokSummaryDetailTotal(column.key) }}</small><small
                        v-else-if="column.key === 'transaction_count'" class="summary-subformula">销量 {{
                          totalValue(filteredTikTokQuarters, row => row.sales_volume).toLocaleString('zh-CN') }}件</small>
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>
          <div v-if="tiktokTab !== 'quarters' && !tiktokError" class="pager">
            <span class="row-count">共 {{ currentTikTokPagination.total }} 条</span>
            <button :disabled="currentTikTokPagination.page <= 1 || tiktokLoading"
              @click="changeTikTokPage(-1)">上一页</button>
            <label class="page-size-control">每页 <select v-model.number="currentTikTokPagination.pageSize"
                :disabled="tiktokLoading" @change="applyTikTokPageSize">
                <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
              </select> 条</label>
            <span>{{ currentTikTokPagination.page }} / {{ Math.max(1, Math.ceil(currentTikTokPagination.total /
              currentTikTokPagination.pageSize)) }}</span>
            <button
              :disabled="currentTikTokPagination.page * currentTikTokPagination.pageSize >= currentTikTokPagination.total || tiktokLoading"
              @click="changeTikTokPage(1)">下一页</button>
          </div>
        </template>
        <template v-else>
          <div class="toolbar">
            <div class="tabs" role="tablist" aria-label="报税数据表">
              <button v-for="item in tabs" :id="`tab-${item.key}`" :key="item.key" role="tab"
                :aria-selected="tab === item.key" aria-controls="tax-table" :class="{ active: tab === item.key }"
                @click="switchTab(item.key)">{{ item.label }}</button>
            </div>
            <div class="filters">
              <button v-if="tab === 'raw'" class="column-config-trigger" :class="{ active: columnConfigOpen }"
                @click.stop="toggleColumnConfig">列配置</button>
              <button :disabled="loading" @click="loadData">刷新</button>
              <button v-if="tab !== 'rules' && canFinance('export', platform)" class="export-btn" :disabled="loading || !!downloadBusy"
                @click="exportReport">{{ downloadBusy === 'export' ? '生成中…' : '下载报表' }}</button>
            </div>
          </div>
          <div v-if="columnConfigOpen" class="column-config-overlay" @click.self="cancelColumnConfig" @click.stop>
            <section class="column-config-dialog" role="dialog" aria-modal="true" aria-labelledby="column-config-title">
              <header class="column-config-heading">
                <div>
                  <h2 id="column-config-title">列配置</h2>
                  <p>选择需要展示的列，并拖动右侧列名调整顺序</p>
                </div><button class="column-config-close" aria-label="关闭列配置" @click="cancelColumnConfig">×</button>
              </header>
              <div class="column-config-body">
                <section class="column-config-available" aria-label="可选列">
                  <label class="column-config-search"><svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
                      <circle cx="8.7" cy="8.7" r="5.7" stroke="currentColor" stroke-width="1.7" />
                      <path d="m13 13 4 4" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" />
                    </svg><input v-model="columnSearch" type="search" placeholder="搜索列名"
                      aria-label="搜索列名" /><kbd>⌕</kbd></label>
                  <div class="column-config-groups">
                    <section v-for="group in columnGroups" :key="group.name" class="column-config-group">
                      <div class="column-group-heading"><span>{{ group.name }}</span><button type="button"
                          @click="toggleColumnGroup(group)">{{group.headers.every(header =>
                            draftVisibleColumns.includes(header)) ? '取消' : '全选' }}</button></div>
                      <div class="column-option-grid"><label v-for="header in group.headers" :key="header"
                          class="column-option"><input type="checkbox" :checked="draftVisibleColumns.includes(header)"
                            @change="toggleColumn(header)" /><span :title="header">{{ headerLabel(header)
                            }}</span></label></div>
                    </section>
                    <p v-if="!columnGroups.length" class="column-config-empty">没有匹配的列</p>
                  </div>
                </section>
                <aside class="column-config-selected" aria-label="已选列排序">
                  <div class="selected-heading"><strong>已选 <span>({{ selectedDraftColumns.length
                        }})</span></strong><button type="button" @click="resetColumns">恢复默认</button></div>
                  <p class="selected-hint">最多可固定 {{ maxPinnedColumns }} 列</p>
                  <div class="selected-column-list">
                    <div v-for="(header, index) in selectedDraftColumns" :key="header" class="selected-column"
                      :class="{ dragging: draggedColumn === header, 'drag-target': dragTarget === header, 'drag-after': dragAfter && dragTarget === header }"
                      @dragenter.prevent="dragTarget = header" @dragover.prevent="updateDragPosition(header, $event)"
                      @dragleave="clearDragTarget(header)" @drop="dropColumn(header, $event)"><span
                        class="selected-drag-handle" title="拖动排序" draggable="true"
                        @dragstart="startColumnDrag(header, $event)" @dragend="endColumnDrag">⠿</span><span
                        class="selected-index">{{ index + 1 }}</span><span class="selected-name" :title="header">{{
                        headerLabel(header) }}</span><button type="button" class="selected-pin"
                        :class="{ active: draftPinnedColumns.includes(header) }"
                        :disabled="!draftPinnedColumns.includes(header) && pinnedHeaders.length + draftPinnedColumns.length >= maxPinnedColumns"
                        :aria-label="`${draftPinnedColumns.includes(header) ? '取消固定' : '固定'}${header}`"
                        :title="draftPinnedColumns.includes(header) ? '取消固定列' : '固定列'"
                        @click="togglePinnedColumn(header)"><svg viewBox="0 0 20 20" aria-hidden="true">
                          <path
                            d="m13.5 2.8 3.7 3.7-2 1.1-.8 3.2-2.1 2.1-2.1-2.1-5.6 5.6-.9-.9 5.6-5.6-2.1-2.1 2.1-2.1 3.2-.8 1-2.1Z"
                            fill="currentColor" />
                          <path d="m10.2 11.5-4.4 4.4" stroke="currentColor" stroke-width="1.4"
                            stroke-linecap="round" />
                        </svg></button><button type="button" class="selected-remove" :aria-label="`移除${header}`"
                        @click="toggleColumn(header)">×</button></div>
                    <p v-if="!selectedDraftColumns.length" class="selected-empty">请从左侧选择要显示的列</p>
                  </div>
                  <p class="selected-hint">拖动手柄可调整列顺序</p>
                </aside>
              </div>
              <footer class="column-config-footer"><button class="column-reset"
                  @click="resetColumns">恢复默认</button><span>固定列最多 {{ maxPinnedColumns }} 项</span>
                <div><button @click="cancelColumnConfig">取消</button><button class="column-apply"
                    @click="applyColumnConfig">保存并应用</button></div>
              </footer>
            </section>
          </div>
          <p v-if="downloadError" class="error" role="alert">{{ downloadError }}</p>
          <p v-if="error" class="error" role="alert">{{ error }}</p>
          <div id="tax-table" role="tabpanel" :aria-labelledby="`tab-${tab}`" :aria-busy="loading">
            <div v-if="tab === 'quarters'" class="summary-toolbar">
              <div><strong>季度汇总</strong><span>按日期范围查看季度和月份数据</span></div>
              <div class="summary-controls"><div v-if="countryId" class="currency-segment" role="group" aria-label="汇总币种"><button
                  :class="{ active: summaryCurrency === 'CNY' }" @click="switchSummaryCurrency('CNY')">人民币</button><button
                  :class="{ active: summaryCurrency === 'USD' }" @click="switchSummaryCurrency('USD')">美元</button></div></div>
            </div>
            <div v-if="!error" class="table-scroll" tabindex="0" :aria-label="tab === 'raw' ? '原始账单表' : '季度汇总表'">
              <div v-if="loading" class="table-loading" role="status">正在更新数据…</div>
              <table v-if="tab === 'raw'" class="raw-table" :style="{ '--column-count': displayHeaders.length }">
                <colgroup>
                  <col v-for="header in displayHeaders" :key="header" :class="columnClass(header)" />
                </colgroup>
                <thead>
                  <tr>
                    <th v-for="header in displayHeaders" :key="header" scope="col" :class="columnClass(header)"
                      :style="columnStyle(header)" :title="headerLabel(header)">{{ headerLabel(header) }} <button
                        v-if="['月份', '季度', '费用类型'].includes(header)" class="filter-trigger"
                        :class="{ active: !!activeFilters[header] }" :aria-label="`筛选${header}`"
                        @click.stop="toggleFilter(header, $event)">⌕</button>
                      <Teleport to="body">
                        <div v-if="filterPopoverField === header" class="header-popover"
                          :style="{ top: `${filterPosition.top}px`, left: `${filterPosition.left}px` }" @click.stop>
                          <div class="popover-title">筛选 {{ header }}</div>
                          <div v-if="filterOptionsLoading" class="filter-loading">加载中…</div><template v-else><button
                              class="filter-option" :class="{ active: !activeFilters[header] }"
                              @click="selectFilter(header, '')">全部</button><button
                              v-for="option in filterOptions(header)" :key="option" class="filter-option"
                              :class="{ active: periodSelected(activeFilters[header], option, header !== '费用类型') }"
                              @click="selectFilter(header, option)"><span v-if="header !== '费用类型'"
                                class="filter-check">{{ periodSelected(activeFilters[header], option) ? '✓' : ''
                                }}</span>{{ option || '空值' }}</button>
                            <div v-if="!filterOptions(header).length" class="filter-loading">暂无可选值</div>
                          </template>
                        </div>
                      </Teleport>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in rawRows" :key="row.id">
                    <td v-for="header in displayHeaders" :key="header" :class="columnClass(header)"
                      :style="columnStyle(header)" :title="String(cellValue(row, header))">{{ cellValue(row, header)
                      }}<small v-if="header === '费用' && rawRates[row.id]" class="rate-note">汇率 {{ rawRates[row.id].rate
                        }}</small></td>
                  </tr>
                  <tr v-if="!loading && !rawRows.length">
                    <td :colspan="displayHeaders.length" class="empty">暂无数据</td>
                  </tr>
                </tbody>
                <tfoot v-if="rawRows.length">
                  <tr>
                    <th v-for="header in displayHeaders" :key="header" :class="columnClass(header)"
                      :style="columnStyle(header)">{{
                        header === '月份' ? '总计' : header === '季度' || header === '费用类型' ? '' : walmartRawTotal(header) }}
                    </th>
                  </tr>
                </tfoot>
              </table>
              <table v-else-if="tab === 'quarters'">
                <thead>
                  <tr>
                    <th scope="col">季度<button class="filter-trigger" :class="{ active: !!summaryQuarterFilter }"
                        @click.stop="toggleSummaryFilter($event)">⌕</button>
                      <Teleport to="body">
                        <div v-if="filterPopoverField === 'summary-quarter'" class="header-popover"
                          :style="{ top: `${filterPosition.top}px`, left: `${filterPosition.left}px` }" @click.stop>
                          <div class="popover-title">筛选季度</div><button class="filter-option"
                            :class="{ active: !summaryQuarterFilter }"
                            @click="summaryQuarterFilter = ''">全部</button><button v-for="value in availableQuarters"
                            :key="value" class="filter-option"
                            :class="{ active: periodSelected(summaryQuarterFilter, value) }"
                            @click="summaryQuarterFilter = togglePeriodValue(summaryQuarterFilter, value)"><span
                              class="filter-check">{{ periodSelected(summaryQuarterFilter, value) ? '✓' : '' }}</span>{{
                            value
                            }}</button>
                        </div>
                      </Teleport>
                    </th>
                    <th v-for="category in categories" :key="category" scope="col" class="numeric">{{ category }}</th>
                    <th scope="col" class="numeric">销量</th>
                    <th scope="col" class="numeric">总费用</th>
                    <th scope="col" class="numeric">净收入（{{ summaryCurrency === 'USD' ? '美元' : '元' }}）<small
                        class="header-formula">销售额 − 退款</small></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in filteredQuarters" :key="row.quarter">
                    <td>{{ row.quarter }}<small v-if="row.unmatched_rows" class="unmatched-note"
                        title="未匹配费用单独展示，不计入总计及报税收入">{{
                        row.unmatched_rows }} 条待分类</small></td>
                    <td v-for="category in categories" :key="category" class="numeric">{{
                      summaryMoney(summaryTotals(row)[category],
                      summaryCurrency) }}</td>
                    <td class="numeric">{{ salesVolume(row.sales_volume) }}件</td>
                    <td class="numeric">{{ summaryMoney(walmartQuarterExpenseValue(row), summaryCurrency) }}</td>
                    <td class="numeric income">{{ summaryMoney(summaryIncome(row), summaryCurrency) }}</td>
                  </tr>
                  <tr v-if="!loading && !filteredQuarters.length">
                    <td :colspan="categories.length + 4" class="empty">暂无数据</td>
                  </tr>
                </tbody>
                <tfoot>
                  <tr>
                    <th>总计</th>
                    <th v-for="category in categories" :key="category" class="numeric">{{ walmartQuarterTotal(category)
                      }}</th>
                    <th class="numeric">{{salesVolume(totalValue(filteredQuarters, row => row.sales_volume))}}件</th>
                    <th class="numeric">{{ walmartQuarterExpenses() }}</th>
                    <th class="numeric income">{{summaryMoney(totalValue(filteredQuarters, row => summaryIncome(row)),
                      summaryCurrency) }}</th>
                  </tr>
                </tfoot>
              </table>
              <section v-if="tab === 'rules'" class="rules-panel">
                <div class="rules-heading">
                  <div><strong>费用类型匹配规则</strong><span>导入数据按照以下规则匹配费用类型</span></div>
                  <div class="rules-actions"><span class="rules-count">{{ rules.length }} 条规则</span><button
                      class="primary" @click="openRuleEditor">新增规则</button></div>
                </div>
                <div class="rules-table-wrap">
                  <table class="rules-table">
                    <thead>
                      <tr>
                        <th>Transaction Type</th>
                        <th>Transaction Description</th>
                        <th>Amount Type</th>
                        <th>费用类型</th>
                        <th>分摊</th>
                        <th>备注</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="rule in rules" :key="rule.id || rule.match_key">
                        <td>{{ rule.transaction_type || '—' }}</td>
                        <td>{{ rule.transaction_description || '—' }}</td>
                        <td>{{ rule.amount_type || '—' }}</td>
                        <td><span class="rule-category">{{ rule.custom_fee_type }}</span></td>
                        <td>{{ rule.allocation || '—' }}</td>
                        <td>{{ rule.note || '—' }}</td>
                      </tr>
                      <tr v-if="!rules.length">
                        <td colspan="10" class="empty">暂无匹配规则</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </section>
            </div>
            <div v-if="tab === 'raw' && !error" class="pager">
              <span class="row-count">共 {{ total }} 条</span>
              <button :disabled="page <= 1 || loading" @click="changePage(-1)">上一页</button>
              <label class="page-size-control">每页 <select v-model.number="pageSize" @change="applyPageSize">
                  <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
                </select> 条</label><span>{{ page }} / {{ Math.max(1, Math.ceil(total / pageSize)) }}</span>
              <button :disabled="page * pageSize >= total || loading" @click="changePage(1)">下一页</button>
            </div>
          </div>
        </template>
      </section>
      <Teleport to="body">
        <div v-if="tiktokFilterField" class="header-popover"
          :style="{ top: `${tiktokFilterPosition.top}px`, left: `${tiktokFilterPosition.left}px` }" @click.stop>
          <div class="popover-title">筛选{{ tikTokFilterLabels[tiktokFilterField] }}</div>
          <div v-if="tiktokFilterLoading" class="filter-loading">加载中…</div>
          <div v-else-if="tiktokFilterError" class="filter-loading" role="alert">{{ tiktokFilterError }}</div>
          <template v-else><button class="filter-option" :class="{ active: !currentTikTokFilters[tiktokFilterField] }"
              @click="selectTikTokFilter('')">全部</button><button v-for="value in tiktokFilterValues" :key="value"
              class="filter-option" :class="{ active: periodSelected(currentTikTokFilters[tiktokFilterField], value) }"
              @click="selectTikTokFilter(value)"><span class="filter-check">{{
                periodSelected(currentTikTokFilters[tiktokFilterField], value) ? '✓' : '' }}</span>{{ value ===
                  '__empty__' ? '空值' : value }}</button>
            <div v-if="!tiktokFilterValues.length" class="filter-loading">暂无可选值</div>
          </template>
        </div>
      </Teleport>
      <Teleport to="body">
        <div v-if="ruleEditorOpen" class="rule-dialog" @click.self="ruleEditorOpen = false">
          <form class="rule-form" @submit.prevent="createRule">
            <div class="rule-form-head"><strong>新增匹配规则</strong><button type="button"
                @click="ruleEditorOpen = false">×</button></div><label>Transaction Type<input
                v-model="ruleForm.transaction_type" required /></label><label>Transaction Description<input
                v-model="ruleForm.transaction_description" required /></label><label>Amount Type<input
                v-model="ruleForm.amount_type" required /></label><label>费用类型<input v-model="ruleForm.custom_fee_type"
                required /></label><label>分摊<input v-model="ruleForm.allocation" /></label><label>备注<textarea
                v-model="ruleForm.note" rows="2" /></label>
            <p v-if="ruleError" class="error">{{ ruleError }}</p>
            <div class="rule-form-actions"><button type="button" @click="ruleEditorOpen = false">取消</button><button
                type="submit" class="primary" :disabled="ruleSaving">{{ ruleSaving ? '保存中…' : '保存规则' }}</button></div>
          </form>
        </div>
      </Teleport>
    </div>
    <Teleport to="body">
      <div v-if="catalogDialogOpen" class="history-dialog" role="dialog" aria-modal="true" aria-label="添加店铺或站点">
        <div class="history-backdrop" aria-hidden="true" @click="closeCatalogDialog" />
        <div class="history-position import-position">
          <div class="history-panel import-panel catalog-panel">
            <form @submit.prevent="saveCatalogItem">
              <div class="history-heading">
                <div>
                  <h2 class="history-title">添加{{ catalogKind === 'store' ? '店铺' : '站点' }}</h2>
                  <p>{{platformOptions.find(item => item.value === platform)?.label}}</p>
                </div><button type="button" :disabled="catalogSaving" @click="closeCatalogDialog">关闭</button>
              </div>
              <div class="catalog-field"><span>类型</span>
                <div class="catalog-kind-switch"><button type="button" :class="{ active: catalogKind === 'store' }"
                    :disabled="catalogSaving" @click="catalogKind = 'store'">店铺</button><button type="button"
                    :class="{ active: catalogKind === 'site' }" :disabled="catalogSaving"
                    @click="catalogKind = 'site'">站点</button></div>
              </div>
              <label class="catalog-field">名称<input v-model.trim="catalogName" required maxlength="100"
                  :disabled="catalogSaving" /></label>
              <label v-if="catalogKind === 'site'" class="catalog-field">币种代码<input v-model.trim="catalogCurrency"
                  required maxlength="3" pattern="[A-Za-z]{3}" placeholder="例如 USD、CAD、EUR"
                  :disabled="catalogSaving" /></label>
              <p v-if="catalogError" class="catalog-error" role="alert">{{ catalogError }}</p>
              <div class="rule-form-actions"><button type="button" :disabled="catalogSaving"
                  @click="closeCatalogDialog">取消</button><button type="submit" class="primary"
                  :disabled="catalogSaving">{{
                    catalogSaving ? '保存中…' : '保存' }}</button></div>
            </form>
          </div>
        </div>
      </div>
    </Teleport>
    <Teleport to="body">
      <div v-if="importOpen" class="history-dialog" role="dialog" aria-modal="true" aria-label="导入报税数据">
        <div class="history-backdrop" aria-hidden="true" @click="closeImportDialog" />
        <div class="history-position import-position">
          <form class="history-panel import-panel" @submit.prevent="submitImports">
            <div class="history-heading">
              <div>
                <h2 class="history-title">导入报税数据</h2>
                <p>选择平台、店铺和对应报表文件，可一次选择多个文件。</p>
              </div><button type="button" aria-label="关闭导入弹窗" @click="closeImportDialog">关闭</button>
            </div>
            <div class="import-fields"><label>平台<select v-model="importPlatform" @change="syncImportStore">
                  <option v-for="item in platformOptions" :key="item.value" :value="item.value">{{ item.label }}
                  </option>
                </select></label><label>店铺<select v-model="importStore">
                  <option v-for="store in storesForPlatform(importPlatform)" :key="store.value" :value="store.value">{{
                    store.label }}</option>
                </select></label><label>站点<select v-model="importSite">
                  <option v-for="site in sitesForPlatform(importPlatform)" :key="site.value" :value="site.value">{{
                    site.label }}</option>
                </select></label></div>
            <div class="import-file-grid"><label v-for="item in importFileKinds(importPlatform)" :key="item.key"
                class="import-file-item"><span>{{ item.label }}<small>{{ item.hint }}</small></span><input type="file"
                  multiple :accept="item.accept" :disabled="importing"
                  @change="setImportFiles(item.key, $event)" /><em>{{
                    importFiles[item.key]?.length || 0 }} 个文件</em></label></div>
            <p v-if="importError" class="error" role="alert">{{ importError }}</p>
            <p v-if="importMessage" class="import-success">{{ importMessage }}</p>
            <div class="rule-form-actions"><button type="button" @click="closeImportDialog">取消</button><button
                class="primary" type="submit" :disabled="importing || !hasImportFiles">{{ importing ? `正在导入
                ${importProgress}` : '开始导入' }}</button></div>
          </form>
        </div>
      </div>
    </Teleport>
    <Teleport to="body">
      <div v-if="historyOpen" class="history-dialog" role="dialog" aria-modal="true" aria-label="操作记录">
        <div class="history-backdrop" aria-hidden="true" />
        <div class="history-position">
          <div class="history-panel">
            <div class="history-heading">
              <div>
                <h2 class="history-title">操作记录</h2>
                <p>导入来源与报表下载记录</p>
              </div><button aria-label="关闭操作记录" @click="historyOpen = false">关闭</button>
            </div>
            <div class="history-toolbar">
              <div class="history-tabs"><button :class="{ active: historyTab === 'imports' }"
                  @click="openHistory('imports')">导入日志</button><button :class="{ active: historyTab === 'downloads' }"
                  @click="openHistory('downloads')">下载记录</button></div><button :disabled="historyLoading"
                @click="loadHistory">刷新</button>
            </div>
            <p v-if="historyTab === 'downloads'" class="history-hint">保留下载时的文件版本；记录表示服务端已生成并返回文件。</p>
            <p v-if="historyError" class="error" role="alert">{{ historyError }}</p>
            <p v-if="downloadError" class="error" role="alert">{{ downloadError }}</p>
            <div class="history-content" :aria-busy="historyLoading">
              <p v-if="historyLoading" class="empty" role="status">加载中…</p>
              <table v-else-if="!historyError" class="history-table">
                <thead>
                  <tr>
                    <th>文件 / 报表</th>
                    <th>{{ historyTab === 'imports' ? '导入者 / 时间' : '下载者 / 时间' }}</th>
                    <th>{{ historyTab === 'imports' ? '导入结果' : '数据范围' }}</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="record in historyRows" :key="record.id">
                    <td class="file-cell"><span :title="record.file_name">{{ record.file_name }}</span><small>{{
                      historyTab
                        === 'imports' ? `店铺 ${record.store_id || 'Forious'} · ${siteOptions.find(item => item.value ===
                          (record.site_id || 'US'))?.label || record.site_id || '美国'}` : reportLabel(record.report_type)
                        }}</small></td>
                    <td><span>{{ historyTab === 'imports' ? record.importer : record.username }}</span><small>{{
                      formatDate(record.created_at) }}</small></td>
                    <td v-if="historyTab === 'imports'"><span :class="['log-status', record.status]">{{
                        statusLabel(record.status) }}</span><small v-if="record.report_type !== 'finance_summary'">入库 {{
                        record.imported_rows }} · 跳过 {{ record.skipped_rows }} / {{ record.source_rows }}
                        条</small><small v-if="record.report_type === 'finance_summary' && record.message"
                        class="summary-extract">{{
                        summaryExtract(record.message) }}</small><small v-else-if="record.message"
                        class="failure-message">{{ record.message }}</small></td>
                    <td v-else><span>{{ [record.month, record.quarter, record.currency].filter(Boolean).join(' · ') ||'全部季度/ 原文件' }}</span><small>{{ record.row_count }} 条</small></td>
                    <td><button class="file-download" :disabled="!!downloadBusy" @click="downloadHistoryFile(record)">{{
                      downloadBusy === `${historyTab}-${record.id}` ? '下载中…' : historyTab === 'imports' ? '下载原文件' :
                        '下载文件'
                        }}</button></td>
                  </tr>
                  <tr v-if="!historyRows.length">
                    <td colspan="4" class="empty">暂无{{ historyTab === 'imports' ? '导入日志' : '下载记录' }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="pager"><span class="row-count">共 {{ historyTotal }} 条</span><button
                :disabled="historyLoading || historyPage <= 1" @click="changeHistoryPage(-1)">上一页</button><span>{{
                historyPage }} / {{ Math.max(1, Math.ceil(historyTotal / 20)) }}</span><button
                :disabled="historyLoading || historyPage * 20 >= historyTotal"
                @click="changeHistoryPage(1)">下一页</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
    <Dialog :open="!!confirmPrompt" class="import-confirm-dialog" :initial-focus="confirmCancelButton"
      @close="finishImportConfirmation(false)">
      <div class="import-confirm-backdrop" aria-hidden="true" />
      <div class="import-confirm-position">
        <DialogPanel class="import-confirm-panel">
          <DialogTitle class="import-confirm-title">文件已导入过</DialogTitle>
          <DialogDescription class="import-confirm-description">{{ confirmPrompt }}</DialogDescription>
          <p class="import-confirm-notice">继续导入会再次新增全部财务明细，并计入汇总金额。</p>
          <div class="import-confirm-actions">
            <button ref="confirmCancelButton" type="button" @click="finishImportConfirmation(false)">取消导入</button>
            <button type="button" class="primary" @click="finishImportConfirmation(true)">继续导入</button>
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  </MainLayout>
</template>

<script setup>
import { computed, reactive, ref, onMounted, onBeforeUnmount } from 'vue'
import { Dialog, DialogPanel, DialogTitle, DialogDescription, Listbox, ListboxButton, ListboxOptions, ListboxOption } from '@headlessui/vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import TaxColumnConfigPanel from '@/components/TaxColumnConfigPanel.vue'
import TaxDateRangePicker from '@/components/TaxDateRangePicker.vue'
import AmazonTaxTables from '@/components/AmazonTaxTables.vue'
import TaxCommissionCell from '@/components/TaxCommissionCell.vue'
import TaxFBTFeeCell from '@/components/TaxFBTFeeCell.vue'
import TaxAffiliateCommissionCell from '@/components/TaxAffiliateCommissionCell.vue'
import TaxTransactionCountCell from '@/components/TaxTransactionCountCell.vue'
import TaxSummaryAmountCell from '@/components/TaxSummaryAmountCell.vue'
import TaxFormulaHelp from '@/components/TaxFormulaHelp.vue'
import request from '@/utils/request'
import { isExcludedTaxTotalColumn } from '@/utils/taxTotals'

const tabs = [{ key: 'raw', label: '原始账单' }, { key: 'quarters', label: '季度汇总' }, { key: 'rules', label: '匹配规则' }]
const platformOptions = [{ value: 'walmart', label: 'Walmart' }, { value: 'tiktok', label: 'TikTok' }, { value: 'amazon', label: 'Amazon' }]
const financePermissions = (() => { try { return JSON.parse(localStorage.getItem('user') || '{}').finance_permissions || {} } catch { return {} } })()
const canFinance = (action, platformKey) => {
  const permissions = financePermissions[platformKey]
  return !Object.keys(financePermissions).length || !!permissions?.includes(action)
}
const availablePlatformOptions = computed(() => platformOptions.filter(item => canFinance('view', item.value)))
const platform = ref('walmart')
const storeCatalog = reactive({ walmart: [{ value: 'TVOFEK', label: 'TVOFEK' }], tiktok: [{ value: 'Forious', label: 'Forious' }], amazon: [{ value: 'sunoutly', label: 'sunoutly' }, { value: 'DearOcean', label: 'DearOcean' }, { value: 'Qfn', label: 'Qfn' }] })
const siteCatalog = reactive({ walmart: [{ value: 'US', label: '美国', currency: 'USD' }, { value: 'CA', label: '加拿大', currency: 'CAD' }], tiktok: [{ value: 'US', label: '美国', currency: 'USD' }, { value: 'CA', label: '加拿大', currency: 'CAD' }], amazon: [{ value: 'US', label: '美国', currency: 'USD' }, { value: 'CA', label: '加拿大', currency: 'CAD' }] })
const catalogDialogOpen = ref(false), catalogKind = ref('store'), catalogName = ref(''), catalogCurrency = ref(''), catalogError = ref('')
const catalogSaving = ref(false)
async function loadCatalog() {
  try {
    const result = await request.get('/tax-data/catalog')
    for (const item of result.data.entries || []) {
      const entries = (item.kind === 'site' ? siteCatalog : storeCatalog)[item.platform]
      if (entries && !entries.some(existing => existing.value === item.value)) entries.push({ value: item.value, label: item.label, currency: item.currency || '' })
    }
  } catch { error.value = '店铺和站点加载失败，请刷新重试' }
}
function openCatalogDialog() { catalogKind.value = 'store'; catalogName.value = ''; catalogCurrency.value = ''; catalogError.value = ''; catalogDialogOpen.value = true }
function closeCatalogDialog() { if (!catalogSaving.value) catalogDialogOpen.value = false }
async function saveCatalogItem() {
  if (catalogSaving.value) return
  const label = catalogName.value.trim()
  const currency = catalogKind.value === 'site' ? catalogCurrency.value.trim().toUpperCase() : ''
  const value = catalogKind.value === 'site' ? '' : label
  if (!label || (catalogKind.value === 'site' && !currency)) { catalogError.value = catalogKind.value === 'site' ? '请填写站点名称和币种代码' : '请填写店铺名称'; return }
  const entries = (catalogKind.value === 'site' ? siteCatalog : storeCatalog)[platform.value]
  if (entries.some(item => item.label.toLowerCase() === label.toLowerCase())) { catalogError.value = '当前平台已存在相同名称'; return }
  catalogSaving.value = true
  catalogError.value = ''
  try {
    const result = await request.post('/tax-data/catalog', { platform: platform.value, kind: catalogKind.value, value, label, currency })
    entries.push({ value: result.data.value, label: result.data.label, currency: result.data.currency || '' })
    catalogDialogOpen.value = false
    if (catalogKind.value === 'store') { storeId.value = result.data.value; switchStore() }
    else { countryId.value = result.data.value; switchCountry() }
  } catch (error) { catalogError.value = error.response?.data?.error || '保存失败，请重试' }
  finally { catalogSaving.value = false }
}
const storeId = ref('')
const siteId = ref('')
const countryId = ref('')
const storeOptions = computed(() => storeCatalog[platform.value] || [])
const siteOptions = computed(() => siteCatalog[platform.value] || [])
const storeLabel = computed(() => storeOptions.value.find(item => item.value === storeId.value)?.label || '全部')
const countryLabel = computed(() => siteOptions.value.find(item => item.value === countryId.value)?.label || '全部')
const selectedSiteCurrency = computed(() => siteOptions.value.find(item => item.value === countryId.value)?.currency || ({ US: 'USD', CA: 'CAD' }[countryId.value] || 'USD'))
const importOpen = ref(false), importPlatform = ref('walmart'), importStore = ref(storeCatalog.walmart[0]?.value || ''), importSite = ref('US'), importFiles = ref({}), importing = ref(false), importProgress = ref(''), importError = ref(''), importMessage = ref('')
const storesForPlatform = value => storeCatalog[value] || []
const sitesForPlatform = value => siteCatalog[value] || []
const importFileKinds = value => value === 'walmart' ? [{ key: 'raw', label: 'Walmart 原始账单', hint: 'CSV / XLSX / XLS', accept: '.csv,.xlsx,.xls' }] : value === 'tiktok' ? [{ key: 'details', label: '涉税申报明细', hint: 'CSV / XLSX / XLS', accept: '.csv,.xlsx,.xls' }, { key: 'ads', label: '广告费', hint: 'CSV / XLSX / XLS', accept: '.csv,.xlsx,.xls' }, { key: 'fbt', label: 'FBT', hint: 'CSV / XLSX / XLS', accept: '.csv,.xlsx,.xls' }, { key: 'settlements', label: '结算单', hint: 'CSV / XLSX / XLS', accept: '.csv,.xlsx,.xls' }, { key: 'orders', label: '订单', hint: 'CSV / XLSX / XLS', accept: '.csv,.xlsx,.xls' }] : [{ key: 'shipments', label: '亚马逊配送货件', hint: 'CSV / XLSX / XLS', accept: '.csv,.xlsx,.xls' }, { key: 'finance', label: '付款报告_联合报告_交易', hint: 'CSV / XLSX / XLS', accept: '.csv,.xlsx,.xls' }, { key: 'finance_summary', label: '付款报告_联合报告_一览 PDF', hint: 'PDF', accept: '.pdf' }]
const hasImportFiles = computed(() => importFileKinds(importPlatform.value).some(item => item.key !== 'finance_summary' && importFiles.value[item.key]?.length))
function openImportDialog() { importPlatform.value = platform.value; importStore.value = storeId.value; importSite.value = siteId.value; importFiles.value = {}; importError.value = ''; importMessage.value = ''; importOpen.value = true }
function syncImportStore() { importStore.value = storesForPlatform(importPlatform.value)[0]?.value || ''; importSite.value = sitesForPlatform(importPlatform.value)[0]?.value || '' }
function closeImportDialog() { if (!importing.value) importOpen.value = false }
function setImportFiles(key, event) { importFiles.value = { ...importFiles.value, [key]: [...(event.target.files || [])] } }
function switchStore() { if (platform.value === 'tiktok') loadTikTok(); else if (platform.value === 'walmart') { page.value = 1; rawFilterOptions.value = {}; loadData() } }
function switchCountry() {
  siteId.value = countryId.value
  if (!countryId.value) {
    // All countries: quarterly summaries must use CNY.
    summaryCurrency.value = 'CNY'
    tiktokCurrency.value = 'CNY'
  }
  if (platform.value === 'tiktok') loadTikTok(); else if (platform.value === 'walmart') loadData()
}
function switchPeriod() { if (platform.value === 'tiktok') loadTikTok(); else if (platform.value === 'walmart') loadData() }
function switchSummaryCurrency(value) { if (summaryCurrency.value === value) return; summaryCurrency.value = value; loadData() }
async function submitImports() {
  if (importing.value) return
  importing.value = true; importError.value = ''; importMessage.value = ''
  const jobs = []
  for (const item of importFileKinds(importPlatform.value)) if (item.key !== 'finance_summary') for (const [fileIndex, file] of (importFiles.value[item.key] || []).entries()) jobs.push({ item, file, fileIndex })
  try {
    for (let index = 0; index < jobs.length; index++) {
      const { item, file, fileIndex } = jobs[index]; importProgress.value = `${index + 1}/${jobs.length} · ${file.name}`
      const endpoint = importPlatform.value === 'walmart' ? '/tax-data/raw/upload' : ` /tax-data/${importPlatform.value}/${item.key}/upload`.trim()
      const submitJob = (confirmed = false) => {
        const form = new FormData(); form.append('file', file); form.append('store_id', importStore.value); form.append('site_id', importSite.value)
        if (confirmed) form.append('confirm_duplicate', '1')
        if (item.key === 'finance' && importFiles.value.finance_summary?.[fileIndex]) form.append('summary_file', importFiles.value.finance_summary[fileIndex])
        return request.post(endpoint, form)
      }
      try {
        await submitJob()
      } catch (error) {
        if (error.response?.status !== 409 || !error.response?.data?.requires_confirmation) throw error
        const prompt = error.response.data.msg || `文件 ${file.name} 已导入过，是否继续？`
        if (!await requestImportConfirmation(prompt)) throw new Error(`已取消导入：${file.name}`)
        await submitJob(true)
      }
    }
    importMessage.value = `成功提交 ${jobs.length} 个文件`; storeId.value = importStore.value; siteId.value = importSite.value; countryId.value = importSite.value; platform.value = importPlatform.value; importOpen.value = false
    if (platform.value === 'tiktok') loadTikTok(); else if (platform.value === 'walmart') loadData()
  } catch (e) { importError.value = e.response?.data?.msg || e.response?.data?.error || e.message || '导入失败，请检查文件格式后重试' } finally { importing.value = false; importProgress.value = '' }
}
const confirmPrompt = ref(''), confirmCancelButton = ref(null)
let resolveImportConfirmation = null
function requestImportConfirmation(prompt) {
  confirmPrompt.value = prompt
  return new Promise(resolve => { resolveImportConfirmation = resolve })
}
function finishImportConfirmation(confirmed) {
  const resolve = resolveImportConfirmation
  resolveImportConfirmation = null
  confirmPrompt.value = ''
  resolve?.(confirmed)
}
onBeforeUnmount(() => finishImportConfirmation(false))

const tiktokTab = ref('details')
const tiktokCurrency = ref('CNY')
const tiktokSummaryColumns = [
  { key: 'transaction_count', label: '交易笔数', count: true, formula: '交易笔数来自 TikTok 交易汇总表；展开后显示销量与样品数' },
  { key: 'sales_amount', label: '销售额', formula: '结算单销售额；运费 = -（客户运费抵扣 + 平台运费折扣 + 客户运费 + 退款客户运费）' },
  { key: 'refund', label: '退款费', formula: '退款费 = 结算单退款金额按当前币种汇总' },
  { key: 'taxable_income', label: '净收入', formula: '净收入 = 收入金额 − 退款' },
  { key: 'commission', label: '平台佣金/服务费', formula: '平台佣金 + 物流相关佣金 + 活动服务费 + 退货运费 + FBT客户运费抵扣调整 + FBT配送费赔偿调整 + 平台佣金退款' },
  { key: 'affiliate_commission', label: '联盟佣金', settlementOnly: true, formula: '联盟佣金 = 直接联盟佣金 + 伙伴佣金 + 店铺广告佣金 + 预留金额 + 退款 + 伙伴广告佣金' },
  { key: 'fbt_fee', label: 'FBT费', settlementOnly: true, formula: 'FBT费 = 仓储费 + 中转仓安置费 + 其他' },
  { key: 'advertising', label: '广告费', formula: '信用额度消耗合计' },
  { key: 'other_fees', label: '其他费用', settlementOnly: true, formula: '其他费用 = 签名确认服务费 + 共担促销 + 配送保障服务费 + 负余额扣款 + 负余额销售收入' },
  { key: 'campaign_service_fee', label: '活动服务费', settlementOnly: true, formula: '活动服务费 = Campaign service fee 金额列合计 + Type=Additional Campaign Package 的 Total settlement amount 合计 + Type=Campaign Package 的 Total settlement amount 合计' },
  { key: 'smart_promotion_fee', label: '智能营销费', pending: true, settlementOnly: true, formula: '智能营销费 = Smart Promotion fee + Smart Promotion campaign period fee' },
  { key: 'tax', label: '税', settlementOnly: true, formula: '税 = Sales tax + Sales tax payment + Sales tax refund + Retail delivery fee + Retail delivery fee payment + Retail delivery fee refund' },
  { key: 'expense_total', label: '总费用', formula: '所有费用合计，除销售额、广告费、退款' },
]
const tiktokSummaryKeys = tiktokSummaryColumns.map(column => column.key)
const tiktokCurrencyLabel = computed(() => tiktokCurrency.value === 'USD' ? '美元' : '元')
const tiktokCurrencySymbol = computed(() => tiktokCurrency.value === 'USD' ? '$' : '¥')
const tiktokFilters = ref({ details: {}, ads: {}, fbt: {}, settlements: {}, orders: {}, quarters: {} })
const tiktokPagination = ref({ details: { page: 1, pageSize: 50, total: 0 }, ads: { page: 1, pageSize: 50, total: 0 }, fbt: { page: 1, pageSize: 50, total: 0 }, settlements: { page: 1, pageSize: 50, total: 0 }, orders: { page: 1, pageSize: 50, total: 0 } })
const tiktokTotals = ref({ details: {}, ads: {}, fbt: {}, settlements: {}, orders: {} })
const tiktokFBT = ref([])
const tiktokSettlements = ref([])
const settlementColumns = ref([])
const tiktokOrders = ref([])
const orderColumns = ref([])
const tiktokColumnConfigOpen = ref(false)
const tiktokConfigVisible = ref([])
const tiktokConfigPinned = ref([])
const tiktokConfigHeaders = computed(() => tiktokTab.value === 'settlements' ? settlementColumns.value.map(column => column.key) : orderColumns.value.map(column => column.key))
const displayedSettlementColumns = computed(() => settlementColumns.value.filter(column => tiktokConfigVisible.value.includes(column.key)))
const displayedOrderColumns = computed(() => orderColumns.value.filter(column => tiktokConfigVisible.value.includes(column.key)))
function applyTikTokColumnConfig(config) { tiktokConfigVisible.value = config.visible; tiktokConfigPinned.value = config.pinned; localStorage.setItem(`tiktok-columns:${storeId.value}:${siteId.value}:${tiktokTab.value}`, JSON.stringify(config)); tiktokColumnConfigOpen.value = false }
function loadTikTokColumnConfig(kind, columns) { let saved = null; try { saved = JSON.parse(localStorage.getItem(`tiktok-columns:${storeId.value}:${siteId.value}:${kind}`) || 'null') } catch { } const headers = columns.map(column => column.key); tiktokConfigVisible.value = saved?.visible?.filter(column => headers.includes(column)) || headers; tiktokConfigPinned.value = saved?.pinned?.filter(column => headers.includes(column)) || [] }
// The first nine original columns are dates, IDs, currency, type, quantity and names.
const settlementAmountKeys = computed(() => new Set(settlementColumns.value.filter(column => column.key === 'Quantity' || (settlementColumns.value.indexOf(column) >= 9 && column.key !== 'Related order ID')).map(column => column.key)))
const settlementQuantityKeys = new Set(['Quantity'])
function settlementCellValue(row, key) {
  const value = row.data[key] ?? ''
  if (!settlementAmountKeys.value.has(key) || String(value).trim() === '') return value
  const amount = String(value).trim()
  if (settlementQuantityKeys.has(key)) return amount
  return amount.startsWith('$') ? amount : `$${amount}`
}
const fbtAmountKeys = ['costRate', 'gmv', 'totalLogisticsCost', 'fbtFulfillmentFee', 'fbtFulfillmentFeeReimbursement', 'shippingFeeDiscount', 'hubPlacementFee', 'storageFee', 'inboundShippingFee', 'inboundIncidentsFee', 'inboundBookingNonComplianceFee', 'routingNonComplianceFee', 'outboundAppointmentNoShowFee', 'delayedResponseFee', 'disposalFee', 'returnToSellerShippingFee', 'returnToSellerHandlingFee', 'inboundReturnOperationFee']
const fbtRate = value => `${(Number(value || 0) * 100).toFixed(2)}%`
const currentTikTokPagination = computed(() => tiktokPagination.value[tiktokTab.value])
const currentTikTokFilters = computed(() => tiktokFilters.value[tiktokTab.value] || {})
const tikTokFilterLabels = { month: '月份', quarter: '季度', logistics_method: '物流方式', order_type: '订单类型' }
const tiktokFilterField = ref('')
const tiktokFilterPosition = ref({ top: 0, left: 0 })
const tiktokFilterValues = ref([])
const tiktokFilterLoading = ref(false)
const tiktokFilterError = ref('')
let tiktokFilterRequest = 0
function closeTikTokFilter() { tiktokFilterField.value = ''; tiktokFilterRequest++ }
async function toggleTikTokFilter(field, event) {
  if (tiktokFilterField.value === field) { closeTikTokFilter(); return }
  const rect = event.currentTarget.getBoundingClientRect()
  tiktokFilterPosition.value = { top: rect.bottom + 6, left: Math.max(8, Math.min(rect.left, window.innerWidth - 220)) }
  tiktokFilterField.value = field
  tiktokFilterLoading.value = true
  tiktokFilterError.value = ''
  const id = ++tiktokFilterRequest
  try {
    if (tiktokTab.value === 'quarters') { tiktokFilterValues.value = tiktokQuarters.value.map(row => row.quarter); return }
    const result = await request.get('/tax-data/tiktok/filter-options', { params: { type: tiktokTab.value, field, store_id: storeId.value, site_id: countryId.value, ...currentTikTokFilters.value } })
    if (id === tiktokFilterRequest) tiktokFilterValues.value = result.data.values || []
  } catch (e) { if (id === tiktokFilterRequest) tiktokFilterError.value = e.response?.data?.error || '筛选值加载失败' }
  finally { if (id === tiktokFilterRequest) tiktokFilterLoading.value = false }
}
function selectTikTokFilter(value) {
  const filters = { ...currentTikTokFilters.value }
  const selected = togglePeriodValue(filters[tiktokFilterField.value], value)
  if (selected) filters[tiktokFilterField.value] = selected
  else delete filters[tiktokFilterField.value]
  tiktokFilters.value[tiktokTab.value] = filters
  if (tiktokTab.value === 'quarters') return
  currentTikTokPagination.value.page = 1
  loadTikTok()
}
const tiktokDetails = ref([])
const tiktokAds = ref([])
const tiktokQuarters = ref([])
const filteredTikTokQuarters = computed(() => tiktokQuarters.value.filter(row => !tiktokFilters.value.quarters.quarter || periodSelected(tiktokFilters.value.quarters.quarter, row.quarter)))
const tiktokLoading = ref(false)
const tiktokError = ref('')
let tiktokRequestId = 0
const tab = ref('raw')
const dateFrom = ref('')
const dateTo = ref('')
const quarter = ref('')
const availableQuarters = ref([])
const availableMonths = ref([])
const quarters = ref([])
const summaryQuarterFilter = ref('')
const filteredQuarters = computed(() => quarters.value.filter(row => !summaryQuarterFilter.value || periodSelected(summaryQuarterFilter.value, row.quarter)))
const categories = ref([])
const rules = ref([])
const ruleEditorOpen = ref(false)
const ruleSaving = ref(false)
const ruleError = ref('')
const ruleForm = ref({ transaction_type: '', transaction_description: '', amount_type: '', custom_fee_type: '', allocation: '', note: '' })
const summaryCurrency = ref('USD')
const rawRows = ref([])
const rawRates = ref({})
const rawTotals = ref({})
const rawFilterOptions = ref({})
const filterOptionsLoading = ref(false)
const rawHeaders = ref([])
const pinnedHeaders = ['月份', '季度', '费用类型', '费用']
const headerLabels = { 'Period Start Date': 'Period Start Date(期间开始日期)', 'Period End Date': 'Period End Date(期间结束日期)', 'Total Payable': 'Total Payable(应付总额)', Currency: 'Currency(币种)', 'Transaction Key': 'Transaction Key(交易键)', 'Transaction Posted Timestamp': 'Transaction Posted Timestamp(交易入账时间)', 'Transaction Type': 'Transaction Type(交易类型)', 'Transaction Description': 'Transaction Description(交易描述)', Amount: 'Amount(金额)', 'Amount Type': 'Amount Type(金额类型)', 'Ship Qty': 'Ship Qty(发货数量)', 'Commission Rate': 'Commission Rate(佣金比例)', 'Base Commission Rate': 'Base Commission Rate(基础佣金比例)', 'Transaction Reason Description': 'Transaction Reason Description(交易原因)', 'Partner Item Id': 'Partner Item Id(合作方商品编号)', 'Partner GTIN': 'Partner GTIN(商品条码)', 'Partner Item Name': 'Partner Item Name(商品名称)', 'Product Tax Code': 'Product Tax Code(商品税码)', 'Ship to State': 'Ship to State(收货州)', 'Ship to City': 'Ship to City(收货城市)', 'Ship to Zipcode': 'Ship to Zipcode(收货邮编)', 'Contract Category': 'Contract Category(合同类别)', 'Product Type': 'Product Type(商品类型)', 'Commission Rule': 'Commission Rule(佣金规则)', 'Shipping Method': 'Shipping Method(配送方式)', 'Fulfillment Type': 'Fulfillment Type(履约类型)', 'Fulfillment Details': 'Fulfillment Details(履约详情)', 'Original Commission': 'Original Commission(原始佣金)', 'Commission Incentive Program': 'Commission Incentive Program(佣金激励计划)', 'Commission Saving': 'Commission Saving(佣金节省)', 'Customer Promo Type': 'Customer Promo Type(客户促销类型)', 'Total Walmart Funded Savings Program': 'Total Walmart Funded Savings Program(Walmart承担优惠总额)', 'Campaign Id': 'Campaign Id(活动编号)', 'Item Condition': 'Item Condition(商品状况)', 'Original charge': 'Original charge(原始收费)', 'Charge Savings': 'Charge Savings(收费节省)', 'Incentive Program Name': 'Incentive Program Name(激励计划名称)', 'Ship to country': 'Ship to country(收货国家)', 'Customer Order #': 'Customer Order(客户订单号)', 'Customer Order line #': 'Customer Order line(客户订单行号)', 'Purchase Order #': 'Purchase Order(采购订单号)', 'Purchase Order line #': 'Purchase Order line(采购订单行号)' }
const headerLabel = header => headerLabels[header] || header
const visibleColumns = ref([])
const columnOrder = ref([])
const pinnedExtra = ref([])
const draftVisibleColumns = ref([])
const draftColumnOrder = ref([])
const draftPinnedColumns = ref([])
const columnSearch = ref('')
const columnConfigOpen = ref(false)
const columnsInitialized = ref(false)
const draggedColumn = ref('')
const dragTarget = ref('')
const dragAfter = ref(false)
const maxPinnedColumns = 7
const columnPreferenceKey = () => { try { const user = JSON.parse(localStorage.getItem('user') || '{}'); return `tax-data-columns:${user.id || user.username || 'current'}` } catch { return 'tax-data-columns:current' } }
const configurableHeaders = computed(() => columnOrder.value)
const selectedDraftColumns = computed(() => draftColumnOrder.value.filter(header => draftVisibleColumns.value.includes(header)))
function rawColumnGroup(header) {
  if (/^(Period Start Date|Period End Date|Currency|Transaction Key|Transaction Posted Timestamp|Transaction Type|Transaction Description|Transaction Reason Description)$/i.test(header)) return '基础信息'
  if (/^(Customer Order #|Customer Order line #|Purchase Order #|Purchase Order line #|Partner Item|Product Tax Code|Ship to |Contract Category|Product Type|Commission Rule|Shipping Method|Fulfillment |Item Condition)/i.test(header)) return '订单与商品信息'
  if (/Amount|Total Payable|Ship Qty/i.test(header)) return '金额与数量'
  if (/Commission|Saving|Promo|Campaign|Incentive/i.test(header)) return '佣金与促销'
  return '其他信息'
}
const columnGroups = computed(() => {
  const groups = new Map()
  const search = columnSearch.value.trim().toLowerCase()
  for (const header of draftColumnOrder.value) {
    if (search && !header.toLowerCase().includes(search)) continue
    const name = rawColumnGroup(header)
    if (!groups.has(name)) groups.set(name, [])
    groups.get(name).push(header)
  }
  return [...groups].map(([name, headers]) => ({ name, headers }))
})
const pinnedDisplayHeaders = computed(() => [...pinnedHeaders, ...columnOrder.value.filter(header => visibleColumns.value.includes(header) && pinnedExtra.value.includes(header))])
const scrollHeaders = computed(() => columnOrder.value.filter(header => visibleColumns.value.includes(header) && !pinnedExtra.value.includes(header)))
const displayHeaders = computed(() => [...pinnedDisplayHeaders.value, ...scrollHeaders.value])
const columnClass = header => ({
  月份: 'pin-month',
  季度: 'pin-quarter',
  费用类型: 'pin-type',
  费用: 'pin-fee numeric',
}[header] || (pinnedExtra.value.includes(header) ? 'regular-column pin-user' : 'regular-column'))
function columnStyle(header) {
  const index = pinnedDisplayHeaders.value.indexOf(header)
  if (index < pinnedHeaders.length) return undefined
  return { left: `${(index - pinnedHeaders.length) * 180}px` }
}
function cellValue(row, header) {
  const field = { 月份: 'month', 季度: 'quarter', 费用类型: 'fee_type', 费用: 'fee' }[header]
  if (!field) {
    const raw = row.data[header] ?? ''
    if (header !== 'Amount' || String(raw).trim() === '') return raw
    const normalized = String(raw).replace(/[$,]/g, '').trim().replace(/^\((.*)\)$/, '-$1')
    return Number.isFinite(Number(normalized)) ? `$${money(normalized)}` : raw
  }
  const value = rawRates.value[row.id]?.[field]
  if (value == null) return '—'
  return field === 'fee' ? `¥${money(value)}` : value
}
const filterField = ref('')
const filterValue = ref('')
const activeFilters = ref({})
const filterPopoverField = ref('')
const filterPosition = ref({ top: 0, left: 0 })
async function toggleFilter(header, event) { filterPopoverField.value = filterPopoverField.value === header ? '' : header; filterField.value = header; if (event?.currentTarget) { const rect = event.currentTarget.getBoundingClientRect(); filterPosition.value = { top: rect.bottom + 6, left: Math.min(rect.left, window.innerWidth - 230) } }; if (filterPopoverField.value === header) { filterOptionsLoading.value = true; try { const result = await request.get('/tax-data/raw/filter-options', { params: { field: header, quarter: quarter.value, store_id: storeId.value, site_id: countryId.value, filters: JSON.stringify(activeFilters.value) } }); rawFilterOptions.value = { ...rawFilterOptions.value, [header]: result.data.values || [] } } catch (e) { error.value = e.response?.data?.error || '筛选值加载失败' } finally { filterOptionsLoading.value = false } } }
function filterOptions(header) { return rawFilterOptions.value[header] || [] }
function periodSelected(selected, value, multi = true) { return multi ? String(selected || '').split(',').includes(value) : selected === value }
function togglePeriodValue(selected, value) {
  if (!value) return ''
  const values = String(selected || '').split(',').filter(Boolean)
  return (values.includes(value) ? values.filter(item => item !== value) : [...values, value]).join(',')
}
function selectFilter(header, value) {
  const next = { ...activeFilters.value }
  const selected = header === '费用类型' ? value : togglePeriodValue(next[header], value)
  if (selected) next[header] = selected
  else delete next[header]
  activeFilters.value = next
  if (header === '费用类型') filterPopoverField.value = ''
  page.value = 1
  loadData({ filters: next })
}
function toggleSummaryFilter(event) {
  filterPopoverField.value = filterPopoverField.value === 'summary-quarter' ? '' : 'summary-quarter'
  const rect = event.currentTarget.getBoundingClientRect()
  filterPosition.value = { top: rect.bottom + 6, left: Math.min(rect.left, window.innerWidth - 230) }
}
function toggleColumn(header) { draftVisibleColumns.value = draftVisibleColumns.value.includes(header) ? draftVisibleColumns.value.filter(value => value !== header) : [...draftVisibleColumns.value, header]; if (!draftVisibleColumns.value.includes(header)) draftPinnedColumns.value = draftPinnedColumns.value.filter(value => value !== header) }
function toggleColumnGroup(group) { const allVisible = group.headers.every(header => draftVisibleColumns.value.includes(header)); draftVisibleColumns.value = allVisible ? draftVisibleColumns.value.filter(header => !group.headers.includes(header)) : [...new Set([...draftVisibleColumns.value, ...group.headers])] }
function toggleColumnConfig() {
  draftColumnOrder.value = [...columnOrder.value]
  draftVisibleColumns.value = [...visibleColumns.value]
  draftPinnedColumns.value = [...pinnedExtra.value]
  columnSearch.value = ''
  columnConfigOpen.value = true
}
function cancelColumnConfig() { columnConfigOpen.value = false; draggedColumn.value = ''; dragTarget.value = ''; dragAfter.value = false }
function applyColumnConfig() {
  columnOrder.value = [...draftColumnOrder.value]
  visibleColumns.value = [...draftVisibleColumns.value]
  pinnedExtra.value = [...draftPinnedColumns.value]
  persistColumnPreference()
  cancelColumnConfig()
}
function togglePinnedColumn(header) {
  if (draftPinnedColumns.value.includes(header)) draftPinnedColumns.value = draftPinnedColumns.value.filter(value => value !== header)
  else if (pinnedHeaders.length + draftPinnedColumns.value.length < maxPinnedColumns) draftPinnedColumns.value = [...draftPinnedColumns.value, header]
}
function persistColumnPreference() { localStorage.setItem(columnPreferenceKey(), JSON.stringify({ order: columnOrder.value, visible: visibleColumns.value, pinned: pinnedExtra.value })) }
function startColumnDrag(header, event) { draggedColumn.value = header; dragTarget.value = ''; dragAfter.value = false; if (event?.dataTransfer) { event.dataTransfer.effectAllowed = 'move'; event.dataTransfer.setData('text/plain', header); const source = event.currentTarget.closest('.selected-column'); const ghost = source?.cloneNode(true) || document.createElement('div'); ghost.className = 'column-drag-ghost'; ghost.style.width = `${source?.getBoundingClientRect().width || 260}px`; document.body.appendChild(ghost); event.dataTransfer.setDragImage(ghost, 24, 20); requestAnimationFrame(() => ghost.remove()) } }
function updateDragPosition(header, event) { dragTarget.value = header; const rect = event.currentTarget.getBoundingClientRect(); dragAfter.value = event.clientY > rect.top + rect.height / 2 }
function clearDragTarget(header) { if (dragTarget.value === header) { dragTarget.value = ''; dragAfter.value = false } }
function endColumnDrag() { draggedColumn.value = ''; dragTarget.value = ''; dragAfter.value = false }
function dropColumn(target, event) { event?.preventDefault(); const source = draggedColumn.value; const after = dragAfter.value; dragTarget.value = ''; dragAfter.value = false; if (!source || source === target) return; const updated = [...draftColumnOrder.value]; const from = updated.indexOf(source); if (from < 0 || updated.indexOf(target) < 0) return; updated.splice(from, 1); const to = updated.indexOf(target); updated.splice(Math.max(0, to + (after ? 1 : 0)), 0, source); draftColumnOrder.value = updated }
function resetColumns() { draftColumnOrder.value = [...rawHeaders.value.filter(header => !pinnedHeaders.includes(header))]; draftVisibleColumns.value = [...draftColumnOrder.value]; draftPinnedColumns.value = [] }
function summaryTotals(row) { return summaryCurrency.value === 'USD' ? (row.usd_totals || {}) : (row.totals || {}) }
function summaryTotal(row) { return summaryCurrency.value === 'USD' ? row.usd_total : row.total }
function summaryIncome(row) { return summaryCurrency.value === 'USD' ? row.usd_taxable_income : row.taxable_income }
function openRuleEditor() { ruleForm.value = { transaction_type: '', transaction_description: '', amount_type: '', custom_fee_type: '', allocation: '', note: '' }; ruleError.value = ''; ruleEditorOpen.value = true }
async function createRule() { ruleSaving.value = true; ruleError.value = ''; try { const result = await request.post('/tax-data/rules', ruleForm.value); rules.value = [...rules.value, result.data]; ruleEditorOpen.value = false } catch (e) { ruleError.value = e.response?.data?.error || '规则保存失败' } finally { ruleSaving.value = false } }
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const pageSizeOptions = [25, 50, 100, 200]
const loading = ref(false)
const error = ref('')
let requestId = 0
const money = value => Number(value || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const salesVolume = value => Math.round(Number(value || 0)).toLocaleString('zh-CN')
const summaryMoney = (value, currency) => Number(value || 0) === 0 ? '-' : `${currency === 'USD' ? '$' : '¥'}${money(value)}`
const incomeMinusRefundLabel = row => row.income == null || row.refund == null ? '待换算' : summaryMoney(Number(row.income) - Number(row.refund), tiktokCurrency.value)
const numericValue = value => { if (value == null || value === '') return 0; const normalized = String(value).replace(/[$,\s]/g, '').replace(/^\((.*)\)$/, '-$1'); const result = Number(normalized); return Number.isFinite(result) ? result : 0 }
const totalValue = (rows, getter) => rows.reduce((total, row) => total + numericValue(getter(row)), 0)
const totalMoney = (source, key) => Number(source?.[key] || 0) === 0 ? '-' : money(source?.[key])
const totalQuantity = (source, key) => Number(source?.[key] || 0).toLocaleString('zh-CN')
const settlementTotal = key => totalMoney(tiktokTotals.value.settlements, key)
const orderTotal = key => totalMoney(tiktokTotals.value.orders, key, 'USD')
const orderQuantityKeys = new Set(['Quantity', 'Sku Quantity of return'])
const orderNonTotalKeys = new Set(['Tracking ID', 'Buyer Nickname', 'Address Line 2', 'Seller Note'])
const orderTotalValue = key => orderNonTotalKeys.has(key) || isExcludedTaxTotalColumn(key) || !(key in (tiktokTotals.value.orders || {})) ? '' : orderQuantityKeys.has(key) ? totalQuantity(tiktokTotals.value.orders, key) : orderTotal(key)
const fbtTotal = key => totalMoney(tiktokTotals.value.fbt, key, 'USD')
const tiktokSummaryTotal = key => summaryMoney(totalValue(filteredTikTokQuarters.value, row => row[key]), tiktokCurrency.value)
const tiktokSummaryTotalFor = key => key === 'expense_total' ? tiktokQuarterTotalExpenses() : tiktokSummaryTotal(key)
const tiktokExpenseKeys = tiktokSummaryColumns.filter(column => !['transaction_count', 'sales_amount', 'refund', 'advertising', 'taxable_income'].includes(column.key)).map(column => column.key)
const tiktokQuarterExpenseValue = row => row.missing_rates ? null : tiktokExpenseKeys.reduce((sum, key) => sum + Number(row[key] || 0), 0)
const tiktokQuarterTotalExpenses = row => summaryMoney(row ? tiktokQuarterExpenseValue(row) : totalValue(filteredTikTokQuarters.value, tiktokQuarterExpenseValue), tiktokCurrency.value)
const tiktokSummaryDetailFields = { sales_amount: 'sales_amount', refund: 'settlement_refund_subtotal', commission: 'settlement_commission_total', affiliate_commission: 'settlement_affiliate_commission', fbt_fee: 'settlement_fbt_fee', other_fees: 'settlement_other_fees', campaign_service_fee: 'settlement_campaign_service_fee', smart_promotion_fee: 'settlement_smart_promotion_fee', tax: 'settlement_tax' }
const tiktokSummaryDetailTotal = key => {
  const field = tiktokSummaryDetailFields[key] || (tiktokSummaryColumns.find(column => column.key === key)?.settlementOnly ? `settlement_${key}` : '')
  if (!field || !filteredTikTokQuarters.value.some(row => row[field] != null)) return null
  return tiktokSummaryTotalFromField(field)
}
const tiktokSummaryTotalFromField = field => summaryMoney(totalValue(filteredTikTokQuarters.value, row => row[field]), tiktokCurrency.value)
const walmartRawTotal = header => { if (isExcludedTaxTotalColumn(header)) return ''; const key = header === '费用' ? 'fee' : header; if (!(key in rawTotals.value)) return ''; return header === 'Ship Qty' ? Number(rawTotals.value[key] || 0).toLocaleString('zh-CN') : money(rawTotals.value[key]) }
const walmartQuarterTotal = category => money(totalValue(filteredQuarters.value, row => summaryTotals(row)[category]))
const excludedExpenseCategories = new Set(['销售额', '广告费', '退款', '未匹配'])
const walmartQuarterExpenseValue = row => Object.entries(summaryTotals(row)).reduce((sum, [category, amount]) => sum + (excludedExpenseCategories.has(category) ? 0 : Number(amount || 0)), 0)
const walmartQuarterExpenses = () => money(totalValue(filteredQuarters.value, walmartQuarterExpenseValue))

async function loadData(overrides = {}) {
  const id = ++requestId
  loading.value = true
  error.value = ''
  try {
    const params = { site_id: countryId.value, store_id: storeId.value }
    if (tab.value === 'raw') params.quarter = quarter.value
    if (tab.value === 'quarters') { params.date_from = dateFrom.value; params.date_to = dateTo.value }
    const filters = overrides.filters ?? activeFilters.value
    if (Object.keys(filters).length) params.filters = JSON.stringify(filters)
    const [summary, raw, ruleResult] = await Promise.all([
      tab.value === 'quarters' ? request.get('/tax-data/quarters', { params }) : null,
      tab.value === 'raw' ? request.get('/tax-data/raw', { params: { ...params, page: page.value, page_size: pageSize.value, include_totals: 1 } }) : null,
      tab.value === 'rules' ? request.get('/tax-data/rules') : null,
    ])
    if (id !== requestId) return
    if (summary) {
      if (!Array.isArray(summary.data?.items)) throw new Error('季度接口返回格式不正确，无法显示汇总。')
      quarters.value = summary.data.items
      categories.value = summary.data.categories || []
      availableQuarters.value = summary.data.available_quarters || []
      availableMonths.value = summary.data.available_months || []
    }
    if (ruleResult) rules.value = Array.isArray(ruleResult.data) ? ruleResult.data : []
    if (raw) {
      rawRows.value = raw.data.items || []
      rawRates.value = raw.data.rates || {}
      rawTotals.value = raw.data.totals || {}
      rawHeaders.value = [...new Set([...(raw.data.headers || []), ...rawRows.value.flatMap(row => Object.keys(row.data))])].filter(header => !/^store[\s_]?id$/i.test(String(header).trim()))
      columnOrder.value = columnOrder.value.filter(header => !/^store[\s_]?id$/i.test(String(header).trim()))
      visibleColumns.value = visibleColumns.value.filter(header => !/^store[\s_]?id$/i.test(String(header).trim()))
      pinnedExtra.value = pinnedExtra.value.filter(header => !/^store[\s_]?id$/i.test(String(header).trim()))
      if (!columnsInitialized.value) {
        const defaults = rawHeaders.value.filter(header => !pinnedHeaders.includes(header) && !/^store[\s_]?id$/i.test(String(header).trim()))
        let saved = null
        try { saved = JSON.parse(localStorage.getItem(columnPreferenceKey()) || 'null') } catch { saved = null }
        columnOrder.value = saved?.order?.filter(header => defaults.includes(header)).concat(defaults.filter(header => !saved?.order?.includes(header))) || [...defaults]
        visibleColumns.value = saved?.visible?.filter(header => defaults.includes(header)) || [...defaults]
        pinnedExtra.value = saved?.pinned?.filter(header => defaults.includes(header)) || []
        columnsInitialized.value = true
      }
      persistColumnPreference()
      total.value = raw.data.total
    }
  } catch (e) {
    if (id === requestId) error.value = [e.response?.data?.error, e.response?.data?.details || e.response?.data?.msg, e.message].filter(Boolean).join('：') || '数据加载失败，请重试。'
  } finally {
    if (id === requestId) loading.value = false
  }
}
async function loadTikTok() {
  const requestedTab = tiktokTab.value
  const pagination = tiktokPagination.value[requestedTab]
  const requestId = ++tiktokRequestId
  tiktokLoading.value = true
  tiktokError.value = ''
  try {
    const endpoint = `/tax-data/tiktok/${requestedTab}`
    const result = await request.get(endpoint, { params: requestedTab === 'quarters' ? { currency: tiktokCurrency.value, store_id: storeId.value, site_id: countryId.value, date_from: dateFrom.value, date_to: dateTo.value } : { ...tiktokFilters.value[requestedTab], page: pagination.page, page_size: pagination.pageSize, include_totals: 1, store_id: storeId.value, site_id: countryId.value } })
    if (requestId !== tiktokRequestId) return
    if (pagination) pagination.total = result.data.total || 0
    if (requestedTab === 'details') tiktokDetails.value = result.data.items || []
    else if (requestedTab === 'ads') tiktokAds.value = result.data.items || []
    else if (requestedTab === 'fbt') tiktokFBT.value = result.data.items || []
    else if (requestedTab === 'settlements') { tiktokSettlements.value = result.data.items || []; settlementColumns.value = result.data.columns || []; loadTikTokColumnConfig('settlements', settlementColumns.value) }
    else if (requestedTab === 'orders') { tiktokOrders.value = result.data.items || []; orderColumns.value = result.data.columns || []; loadTikTokColumnConfig('orders', orderColumns.value) }
    else tiktokQuarters.value = result.data.items || []
    if (requestedTab !== 'quarters') tiktokTotals.value[requestedTab] = result.data.totals || {}
  } catch (e) {
    if (requestId === tiktokRequestId) tiktokError.value = e.response?.data?.error || 'TikTok 数据加载失败，请重试'
  } finally {
    if (requestId === tiktokRequestId) tiktokLoading.value = false
  }
}
function changeTikTokPage(delta) {
  currentTikTokPagination.value.page += delta
  closeTikTokFilter()
  loadTikTok()
}
function applyTikTokPageSize() {
  if (tiktokTab.value === 'quarters') return
  currentTikTokPagination.value.page = 1
  loadTikTok()
}
function switchTikTokCurrency(value) {
  if (tiktokCurrency.value === value) return
  tiktokCurrency.value = value
  tiktokQuarters.value = []
  loadTikTok()
}
function switchTikTokTab(value) {
  if (tiktokTab.value === value) return
  closeTikTokFilter()
  tiktokTab.value = value
  loadTikTok()
}
function switchPlatform(value) {
  if (platform.value === value) return
  closeTikTokFilter()
  historyOpen.value = false
  historyRequest++
  downloadError.value = ''
  filterPopoverField.value = ''
  columnConfigOpen.value = false
  platform.value = value
  storeId.value = ''
  countryId.value = ''
  siteId.value = ''
  if (value === 'tiktok') loadTikTok()
  else {
    tiktokRequestId++
    tiktokLoading.value = false
    if (value === 'walmart') loadData()
  }
}
function switchTab(value) {
  if (tab.value === value) return
  tab.value = value
  loadData()
}
function applyQuarter() { page.value = 1; rawFilterOptions.value = {}; activeFilters.value = {}; filterField.value = ''; filterPopoverField.value = ''; filterValue.value = ''; loadData() }
function applyPageSize() { page.value = 1; loadData() }
function changePage(delta) { page.value += delta; loadData() }
const historyOpen = ref(false)
const historyTab = ref('imports')
const historyRows = ref([])
const historyPage = ref(1)
const historyTotal = ref(0)
const historyLoading = ref(false)
const historyError = ref('')
const downloadBusy = ref('')
const downloadError = ref('')
let historyRequest = 0
const formatDate = value => new Date(value).toLocaleString('zh-CN', { hour12: false })
const formatDateOnly = value => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).replace(/\//g, '-')
}
const statusLabel = value => ({ processing: '处理中', locked: '已完成', needs_review: '待核对', duplicate: '重复已跳过', failed: '失败' }[value] || value)
const summaryExtract = value => {
  try {
    const data = JSON.parse(value)
    const money = amount => (amount === undefined || amount === null || amount === '' ? '-' : amount)
    return ['提取数据：', '期间', data.start, '~', data.end].join(' ') + ' | ' + ["币种 " + (data.currency || '-')].join('') + ' | ' + ["税金 " + money(data.tax)] + ' | ' + ["仓储费 " + money(data.storage)] + ' | ' + ["配送费 " + money(data.delivery)]
  } catch { return value }
}

const reportLabel = value => ({ shipments: '亚马逊配送货件', finance: '付款报告_联合报告_交易', finance_summary: '付款报告_联合报告_一览', original: '原始文件', raw: '原始账单', quarters: '季度汇总', details: '涉税申报明细', ads: '广告费', orders: '订单', original_orders: '订单原文件', settlements: '结算单', original_settlements: '结算单原文件', original_details: '申报明细原文件', original_ads: '广告费原文件' }[value] || value)
function openHistory(value) {
  historyTab.value = value
  historyPage.value = 1
  historyOpen.value = true
  downloadError.value = ''
  loadHistory()
}
function changeHistoryPage(delta) { historyPage.value += delta; loadHistory() }
async function loadHistory() {
  const id = ++historyRequest
  historyLoading.value = true
  historyError.value = ''
  try {
    const result = await request.get(`/tax-data/${historyTab.value}`, { params: { platform: platform.value, page: historyPage.value, page_size: 20, site_id: countryId.value, store_id: storeId.value } })
    if (id !== historyRequest) return
    historyRows.value = result.data.items || []
    historyTotal.value = result.data.total
  } catch (e) {
    if (id === historyRequest) historyError.value = e.response?.data?.error || '记录加载失败，请重试。'
  } finally { if (id === historyRequest) historyLoading.value = false }
}
async function downloadFile(url, key, fallbackName, params) {
  if (downloadBusy.value) return
  downloadBusy.value = key
  downloadError.value = ''
  try {
    const result = await request.get(url, { params, responseType: 'blob' })
    const disposition = result.headers['content-disposition'] || ''
    const encodedName = disposition.match(/filename\*=UTF-8''([^;]+)/i)?.[1]
    const plainName = disposition.match(/filename="?([^";]+)"?/i)?.[1]
    const filename = encodedName ? decodeURIComponent(encodedName) : plainName || fallbackName
    const href = URL.createObjectURL(result.data)
    const link = document.createElement('a')
    link.href = href
    link.download = filename
    document.body.appendChild(link)
    link.click()
    link.remove()
    setTimeout(() => URL.revokeObjectURL(href), 1000)
    if (historyOpen.value && historyTab.value === 'downloads') await loadHistory()
  } catch (e) {
    let message = '下载失败，请重试。'
    if (e.response?.data instanceof Blob) {
      try { message = JSON.parse(await e.response.data.text()).error || message } catch { /* Keep fallback for non-JSON responses. */ }
    }
    downloadError.value = message
  } finally { downloadBusy.value = '' }
}
function exportReport() {
  downloadFile('/tax-data/export', 'export', `${reportLabel(tab.value)}_${quarter.value || '全部季度'}.csv`, { type: tab.value, ...(tab.value === 'quarters' ? { currency: summaryCurrency.value, date_from: dateFrom.value, date_to: dateTo.value } : {}), quarter: tab.value === 'quarters' ? summaryQuarterFilter.value : quarter.value, store_id: storeId.value, site_id: countryId.value, filters: JSON.stringify(activeFilters.value) })
}
function exportTikTokReport() {
  const params = { type: tiktokTab.value, store_id: storeId.value, site_id: countryId.value, ...(tiktokTab.value === 'quarters' ? { currency: tiktokCurrency.value, ...currentTikTokFilters.value } : currentTikTokFilters.value) }
  downloadFile('/tax-data/tiktok/export', 'export', `TikTok_${reportLabel(tiktokTab.value)}.csv`, params)
}
function downloadHistoryFile(record) {
  downloadFile(`/tax-data/${historyTab.value}/${record.id}/file`, `${historyTab.value}-${record.id}`, record.file_name, { platform: platform.value, site_id: countryId.value })
}
function closeFilterOnOutsideClick(event) {
  if (tiktokFilterField.value && !event.target.closest('.filter-trigger, .header-popover')) closeTikTokFilter()
  if (filterPopoverField.value && !event.target.closest('.filter-trigger, .header-popover')) filterPopoverField.value = ''
  if (columnConfigOpen.value && !event.target.closest('.column-config-trigger, .column-config-overlay')) cancelColumnConfig()
}
onMounted(() => { loadCatalog(); loadData(); document.addEventListener('click', closeFilterOnOutsideClick); window.addEventListener('resize', closeTikTokFilter); window.addEventListener('scroll', closeTikTokFilterOnScroll, true); document.addEventListener('keydown', closeTikTokFilterOnEscape) })
function closeTikTokFilterOnScroll(event) { if (!event.target.closest?.('.header-popover')) closeTikTokFilter() }
function closeTikTokFilterOnEscape(event) { if (event.key === 'Escape') closeTikTokFilter() }
onBeforeUnmount(() => { document.removeEventListener('click', closeFilterOnOutsideClick); window.removeEventListener('resize', closeTikTokFilter); window.removeEventListener('scroll', closeTikTokFilterOnScroll, true); document.removeEventListener('keydown', closeTikTokFilterOnEscape) })
</script>

<style scoped>
.tax-page {
  width: 100%;
  height: calc(100dvh - 64px);
  min-width: 0;
  min-height: 0;
  margin: auto;
  padding: 24px;
  box-sizing: border-box;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  color: #253447;
}

.panel {
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  background: #fff;
  border: 1px solid #dfe5ec;
  border-radius: 8px;
  overflow: visible;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 64px;
  padding: 0 18px;
  border-bottom: 1px solid #e3e8ef;
}

.tabs,
.filters {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filters {
  font-size: 12px;
  color: #68788a;
  gap: 10px;
}

.column-config-trigger.active {
  color: #285d83;
  border-color: #b9ccdc;
  background: #f1f6fa;
}

button,
select {
  border: 1px solid #dce4e7;
  border-radius: 6px;
  background: #fff;
  height: 32px;
  padding: 0 12px;
  font-size: 12px;
  color: #465669;
}

button {
  cursor: pointer;
}

button:hover {
  background: #f3f6f9;
  border-color: #bdcad7;
}

button:disabled {
  opacity: .5;
  cursor: default;
}

.tabs {
  align-self: stretch;
  gap: 24px;
}

.tabs button {
  position: relative;
  height: 100%;
  min-height: 52px;
  padding: 0 2px;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: #738093;
  font-size: 13px;
}

.tabs .active {
  color: #285d83;
  font-weight: 600;
}

.tabs .active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 2px;
  background: #285d83;
}

select {
  min-width: 132px;
}

.table-scroll {
  overflow: auto;
  flex: 1 1 auto;
  height: auto;
  min-height: 0;
  max-width: 100%;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: #aebbc8 #f5f7fa;
}

.tiktok-table-region,
#tax-table {
  min-height: 0;
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
}

.tiktok-scroll {
  min-height: 0;
}

.table-loading {
  position: sticky;
  z-index: 20;
  top: 50%;
  display: block;
  width: fit-content;
  margin: 120px auto -156px;
  padding: 8px 14px;
  border: 1px solid #dce5ed;
  border-radius: 16px;
  color: #62788b;
  background: rgb(255 255 255 / 94%);
  box-shadow: 0 2px 8px rgb(30 50 70 / 8%);
  font-size: 11px;
  pointer-events: none;
}

table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  font-size: 12px;
}

th,
td {
  height: 36px;
  padding: 8px 12px;
  text-align: left;
  white-space: nowrap;
  border-bottom: 1px solid #e8edf2;
  border-right: 1px solid #f1f3f6;
}

th {
  position: sticky;
  top: 0;
  z-index: 2;
  height: 48px;
  background: #f4f6f9;
  color: #596b7e;
  font-weight: 600;
  line-height: 1.4;
  white-space: normal;
}

td {
  color: #455961;
  background: #fff;
}

tbody tr:nth-child(even) td {
  background: #fcfdfe;
}

tbody tr:hover td {
  background: #f0f6fb;
}

/* Keep each total cell vertically fixed while leaving the whole row in the table's horizontal scroll track. */
tfoot {
  position: static;
}

tbody+tfoot tr {
  box-shadow: 0 -5px 12px rgb(44 67 87 / 10%);
}

tfoot th,
tfoot td {
  position: sticky;
  top: auto;
  bottom: 0;
  z-index: 2;
  height: 42px;
  padding: 9px 12px;
  color: #30495f;
  background: #edf3f7;
  border-top: 2px solid #b9cbd8;
  border-bottom: 0;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

tfoot th:first-child {
  color: #285d83;
  letter-spacing: .02em;
}

tfoot tr:hover th,
tfoot tr:hover td {
  background: #edf3f7;
}

tfoot .pin-month,
tfoot .pin-quarter,
tfoot .pin-type,
tfoot .pin-fee,
tfoot .pin-user {
  z-index: 5;
  background: #edf3f7;
}

.numeric {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.unmatched-note {
  display: block;
  margin-top: 4px;
  font-size: 10px;
  color: #926a28;
}

.rate-note {
  display: block;
  margin-top: 3px;
  color: #738292;
  font-size: 10px;
  font-weight: 400;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
}

.income {
  color: #285d83;
  font-weight: 600;
}

.empty {
  text-align: center;
  color: #89949b;
  padding: 40px;
  font-size: 13px;
}

.pager {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
  min-height: 56px;
  padding: 10px 18px;
  border-top: 1px solid #e3e8ef;
  font-size: 12px;
  color: #738093;
}

.row-count {
  margin-right: auto;
}

.page-size-control {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
}

.page-size-control select {
  min-width: 64px;
  height: 30px;
  padding: 0 7px;
}

.error {
  margin: 16px 20px;
  color: #a34635;
  font-size: 13px;
}

button:focus-visible,
select:focus-visible,
.table-scroll:focus-visible {
  outline: 2px solid #285d83;
  outline-offset: 2px;
}

.raw-table {
  table-layout: fixed;
  width: calc((var(--column-count) - 4) * 180px + var(--month-width) + var(--quarter-width) + var(--type-width) + var(--fee-width));
  min-width: 100%;
  --month-width: 86px;
  --quarter-width: 90px;
  --type-width: 100px;
  --fee-width: 124px;
}

.raw-table th,
.raw-table td {
  overflow: hidden;
  text-overflow: ellipsis;
}

.raw-table .regular-column {
  width: 180px;
}

.raw-table .pin-month {
  width: var(--month-width);
}

.raw-table .pin-quarter {
  width: var(--quarter-width);
}

.raw-table .pin-type {
  width: var(--type-width);
}

.raw-table .pin-fee {
  width: var(--fee-width);
}

.raw-table :is(th, td).pin-month,
.raw-table :is(th, td).pin-quarter,
.raw-table :is(th, td).pin-type,
.raw-table :is(th, td).pin-fee {
  position: sticky;
  z-index: 3;
  box-sizing: border-box;
  background: #fff;
}

.raw-table :is(th, td).pin-month {
  left: 0;
}

.raw-table :is(th, td).pin-quarter {
  left: var(--month-width);
}

.raw-table :is(th, td).pin-type {
  left: calc(var(--month-width) + var(--quarter-width));
}

.raw-table :is(th, td).pin-fee {
  left: calc(var(--month-width) + var(--quarter-width) + var(--type-width));
}

.raw-table thead .pin-month,
.raw-table thead .pin-quarter,
.raw-table thead .pin-type,
.raw-table thead .pin-fee {
  z-index: 6;
  background: #f0f3f7;
}

.raw-table :is(th, td).pin-user {
  position: sticky;
  z-index: 3;
  box-sizing: border-box;
}

.raw-table thead th.pin-user {
  top: 0;
  z-index: 5;
  background: #f0f3f7;
  box-shadow: 2px 0 4px -4px #8092a6;
}

@media (max-width: 900px) {
  .raw-table {
    --month-width: 64px;
    --quarter-width: 66px;
    --type-width: 72px;
    --fee-width: 94px;
  }

  .raw-table th,
  .raw-table td {
    padding: 8px 5px;
  }
}

@media (max-width: 600px) {
  .tax-page {
    padding: 16px 8px;
  }

  .raw-table {
    --month-width: 58px;
    --quarter-width: 60px;
    --type-width: 64px;
    --fee-width: 82px;
  }

  th,
  td {
    padding: 8px;
  }

  .table-scroll {
    height: calc(100dvh - 320px);
  }

  .toolbar {
    flex-wrap: wrap;
    padding: 0 12px 12px;
    gap: 4px;
  }

  .tabs {
    width: 100%;
  }

  .filters {
    width: 100%;
  }

  .filters select {
    flex: 1;
  }

  .pager {
    gap: 10px;
    padding: 12px;
  }
}

:global(.import-confirm-dialog) {
  position: fixed;
  inset: 0;
  z-index: 10020;
}

.import-confirm-backdrop {
  position: fixed;
  inset: 0;
  background: rgb(18 30 44 / 36%);
}

.import-confirm-position {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  overflow-y: auto;
}

.import-confirm-panel {
  box-sizing: border-box;
  width: min(440px, 100%);
  max-height: calc(100dvh - 40px);
  overflow-y: auto;
  padding: 24px;
  border: 1px solid #dce5eb;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 20px 50px rgb(20 35 50 / 20%);
}

.import-confirm-title {
  margin: 0;
  color: #253447;
  font-size: 16px;
  font-weight: 600;
}

.import-confirm-description {
  margin: 14px 0 0;
  color: #526274;
  font-size: 13px;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.import-confirm-notice {
  margin: 12px 0 0;
  color: #926a28;
  font-size: 12px;
  line-height: 1.6;
}

.import-confirm-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 24px;
}

.import-confirm-actions button {
  height: 34px;
  padding: 0 14px;
  border: 1px solid #dce4e7;
  border-radius: 6px;
  background: #fff;
  color: #465669;
  font-size: 12px;
  cursor: pointer;
}

.import-confirm-actions button.primary {
  color: #fff;
  border-color: #285d83;
  background: #285d83;
}

.import-confirm-actions button:hover {
  filter: brightness(.96);
}

.import-confirm-actions button:focus-visible {
  outline: 2px solid #285d83;
  outline-offset: 3px;
}

.history-actions {
  display: flex;
  gap: 8px;
}

.export-btn {
  color: #fff;
  background: #285d83;
  border-color: #285d83;
}

.export-btn:hover {
  background: #204e70;
}

.history-dialog {
  position: fixed !important;
  inset: 0 !important;
  z-index: 9999 !important;
  width: 100%;
  height: 100%;
}

.history-backdrop {
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: auto;
  background: rgb(18 30 44 / 28%);
}

.history-position {
  position: absolute;
  inset: 0;
  z-index: 1;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
}

.import-position {
  justify-content: center;
  align-items: center;
  padding: 24px;
  box-sizing: border-box;
}

.history-panel {
  pointer-events: auto;
  display: flex;
  flex-direction: column;
  width: min(100%, 960px);
  max-width: 100vw;
  height: 100%;
  max-height: 100dvh;
  overflow: hidden;
  background: white;
  box-shadow: -8px 0 32px rgb(20 35 50 / 10%);
}

.history-heading {
  flex: 0 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px;
  border-bottom: 1px solid #e3e8ef;
}

.history-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #253447;
}

.history-heading p {
  margin: 6px 0 0;
  font-size: 12px;
  color: #738093;
}

.history-tabs {
  display: flex;
  gap: 20px;
  height: 56px;
  align-items: center;
}

.history-tabs button {
  height: 100%;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: #738093;
}

.history-tabs button.active {
  color: #285d83;
  border-bottom: 2px solid #285d83;
  font-weight: 600;
}

.history-toolbar {
  flex: 0 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  min-height: 56px;
  border-bottom: 1px solid #e3e8ef;
}

.history-hint {
  flex: 0 0 auto;
  margin: 0;
  padding: 10px 24px;
  color: #738093;
  background: #f8fafc;
  font-size: 11px;
}

.history-content {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  overflow: auto;
  overscroll-behavior: contain;
}

.history-table {
  min-width: 720px;
  width: 100%;
}

.history-table th {
  position: static;
  top: 0;
  z-index: 1;
  background: #f4f6f9;
}

.history-table td {
  padding: 14px 16px;
  vertical-align: top;
}

.history-table small {
  display: block;
  margin-top: 6px;
  font-size: 11px;
  color: #7b8795;
}

.file-cell {
  max-width: 280px;
}

.file-cell>span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-status {
  font-size: 11px;
  padding: 3px 6px;
  background: #edf2f7;
  color: #556b80;
  border-radius: 4px;
}

.log-status.locked {
  background: #edf6f1;
  color: #39755a;
}

.log-status.needs_review {
  background: #fff5e5;
  color: #926a28;
}

.log-status.failed {
  background: #fff0ed;
  color: #a34635;
}

.history-table .failure-message {
  color: #a34635;
  white-space: normal;
  max-width: 220px;
}

.history-table .summary-extract {
  color: #556b80;
  white-space: normal;
  max-width: 260px;
}

.file-download {
  color: #285d83;
  border-color: #dce5ed;
}

@media (max-width: 600px) {
  .history-panel {
    width: 100vw;
  }

  .history-heading {
    padding: 18px 16px;
  }

  .history-toolbar {
    padding: 0 16px;
    gap: 16px;
  }

  .filters {
    flex-wrap: wrap;
  }
}

.summary-rate {
  display: block;
  margin-top: 4px;
  color: #8493a0;
  font-size: 10px;
  font-weight: 400;
}

.summary-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 18px;
  border-bottom: 1px solid #e5ebf0;
  background: #fbfcfd;
}

.summary-toolbar strong {
  display: block;
  color: #33485b;
  font-size: 12px;
}

.summary-toolbar span {
  display: block;
  margin-top: 3px;
  color: #8794a0;
  font-size: 11px;
}

.summary-controls { display:flex; flex-direction:column; align-items:flex-start; gap:8px; }

.currency-segment {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border: 1px solid #d8e1e8;
  border-radius: 7px;
  background: #f1f4f7;
}

.currency-segment button {
  min-width: 68px;
  height: 32px;
  padding: 0 10px;
  border: 0;
  border-radius: 5px;
  color: #6d7c89;
  background: transparent;
  font-size: 11px;
}

.currency-segment button:hover {
  border-color: transparent;
  background: #e5ebf0;
}

.currency-segment button.active {
  color: #285d83;
  background: #fff;
  box-shadow: 0 1px 4px rgb(38 56 74 / 12%);
  font-weight: 700;
}

th {
  position: sticky;
  top: 0;
  z-index: 2;
}

.header-popover {
  position: fixed;
  z-index: 10001;
  display: flex;
  flex-direction: column;
  gap: 3px;
  width: 190px;
  max-height: min(320px, calc(100dvh - 24px));
  overflow-y: auto;
  padding: 7px;
  background: #fff;
  border: 1px solid #d7e0e8;
  border-radius: 8px;
  box-shadow: 0 12px 30px rgb(20 35 50 / 17%);
}

.column-config-overlay {
  position: fixed;
  inset: 0;
  z-index: 10002;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 22px;
  background: rgb(20 35 50 / 32%);
  backdrop-filter: blur(2px);
}

.column-config-dialog {
  display: flex;
  flex-direction: column;
  width: min(1180px, 100%);
  height: min(900px, calc(100dvh - 44px));
  min-height: 360px;
  overflow: hidden;
  border: 1px solid #dce4ec;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 24px 64px rgb(20 35 50 / 22%);
}

.column-config-heading,
.column-config-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.column-config-heading {
  flex: 0 0 auto;
  padding: 20px 24px 16px;
  border-bottom: 1px solid #e8edf2;
}

.column-config-heading h2 {
  margin: 0;
  color: #26384a;
  font-size: 20px;
  font-weight: 650;
}

.column-config-heading p {
  margin: 6px 0 0;
  color: #81909e;
  font-size: 12px;
}

.column-config-close {
  display: grid;
  place-items: center;
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 0;
  color: #7b8997;
  background: transparent;
  font-size: 23px;
}

.column-config-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  flex: 1 1 auto;
  min-height: 0;
}

.column-config-available {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  padding: 20px 24px;
}

.column-config-search {
  display: flex;
  align-items: center;
  flex: 0 0 44px;
  gap: 10px;
  padding: 0 12px;
  border: 1px solid #dce4ec;
  border-radius: 7px;
  color: #91a0ae;
}

.column-config-search:focus-within {
  border-color: #75a5c7;
  box-shadow: 0 0 0 3px rgb(40 111 170 / 9%);
}

.column-config-search svg {
  width: 17px;
  height: 17px;
  flex: 0 0 17px;
}

.column-config-search input {
  width: 100%;
  min-width: 0;
  height: 38px;
  padding: 0;
  border: 0;
  outline: 0;
  color: #33485b;
  background: transparent;
  font: inherit;
  font-size: 13px;
}

.column-config-search kbd {
  padding: 3px 6px;
  border: 1px solid #e1e7ed;
  border-radius: 4px;
  color: #a0abb5;
  background: #f8fafb;
  font: inherit;
  font-size: 10px;
  white-space: nowrap;
}

.column-config-groups {
  flex: 1 1 auto;
  min-height: 0;
  margin-top: 14px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #c7d1da transparent;
}

.column-config-group {
  padding: 12px 0 14px;
  border-bottom: 1px solid #edf1f5;
}

.column-group-heading {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 11px;
  color: #8392a0;
  font-size: 12px;
}

.column-group-heading::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #aab9c7;
}

.column-group-heading button,
.selected-heading button {
  padding: 3px 5px;
  border: 0;
  color: #2677e8;
  background: transparent;
  font-size: 11px;
}

.column-option-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 11px 16px;
}

.column-option {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: #48596a;
  font-size: 12px;
  cursor: pointer;
}

.column-option span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.column-option input {
  flex: 0 0 15px;
  width: 15px;
  height: 15px;
  margin: 0;
  accent-color: #347df4;
}

.column-config-selected {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  padding: 22px 18px 12px;
  border-left: 1px solid #e7edf2;
  background: #fafbfd;
}

.selected-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #27394b;
  font-size: 15px;
}

.selected-heading span {
  color: #8b98a5;
  font-weight: 500;
}

.selected-hint {
  margin: 10px 0 12px;
  color: #99a5b0;
  font-size: 12px;
}

.selected-column-list {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  scrollbar-width: thin;
  scrollbar-color: #c7d1da transparent;
}

.selected-column {
  display: flex;
  align-items: center;
  min-height: 39px;
  gap: 9px;
  padding: 0 2px;
  border-bottom: 1px solid #f0f3f6;
  color: #344557;
  font-size: 12px;
}

.selected-column.dragging {
  opacity: .4;
  background: #eef5fa;
}

.selected-column.drag-target {
  box-shadow: inset 0 2px #347df4;
  background: #f1f7fd;
}

.selected-column.drag-after {
  box-shadow: inset 0 -2px #347df4;
}

.selected-drag-handle {
  flex: 0 0 17px;
  color: #aebbc7;
  font-size: 18px;
  cursor: grab;
  touch-action: none;
  user-select: none;
}

.selected-index {
  flex: 0 0 18px;
  color: #7d8b99;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.selected-name {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-pin {
  display: grid;
  place-items: center;
  flex: 0 0 27px;
  width: 27px;
  height: 27px;
  padding: 0;
  border: 1px solid #e1e7ed;
  border-radius: 6px;
  color: #94a2ae;
  background: #fff;
}

.selected-pin svg {
  width: 15px;
  height: 15px;
  transform: rotate(-38deg);
}

.selected-pin.active {
  border-color: #bad1e3;
  color: #2e6e9f;
  background: #eef6fb;
}

.selected-pin:disabled {
  opacity: .4;
  cursor: not-allowed;
}

.selected-remove {
  flex: 0 0 24px;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  color: #8c9aa7;
  background: transparent;
  font-size: 17px;
}

.selected-empty,
.column-config-empty {
  padding: 28px 10px;
  color: #9aa6b1;
  font-size: 12px;
  text-align: center;
}

.column-config-footer {
  flex: 0 0 auto;
  min-height: 64px;
  padding: 10px 20px 10px 24px;
  border-top: 1px solid #e7edf2;
}

.column-config-footer>span {
  margin: 0;
  color: #8a98a5;
  font-size: 11px;
}

.column-config-footer>div {
  display: flex;
  gap: 8px;
  margin-left: auto;
}

.column-config-footer button {
  min-width: 76px;
  height: 34px;
  padding: 0 13px;
  border: 1px solid #d5dee7;
  border-radius: 6px;
  color: #536577;
  background: #fff;
  font-size: 12px;
}

.column-config-footer .column-reset {
  margin-right: auto;
  color: #536577;
}

.column-config-footer .column-apply {
  border-color: #1264f5;
  color: #fff;
  background: #1264f5;
}

.column-config-footer .column-apply:hover {
  border-color: #0754dc;
  background: #0754dc;
}

@media (max-width: 800px) {
  .column-config-overlay {
    padding: 10px;
  }

  .column-config-dialog {
    height: calc(100dvh - 20px);
    min-height: 0;
  }

  .column-config-body {
    grid-template-columns: minmax(0, 1fr) 250px;
  }

  .column-config-available {
    padding: 16px;
  }

  .column-config-selected {
    padding: 18px 12px 10px;
  }

  .column-option-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .column-config-body {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: minmax(0, 1fr) minmax(180px, 38%);
  }

  .column-config-available {
    padding: 12px 14px;
  }

  .column-config-selected {
    padding: 12px 14px 8px;
    border-top: 1px solid #e7edf2;
    border-left: 0;
  }

  .column-config-heading {
    padding: 14px 16px 12px;
  }

  .column-config-heading h2 {
    font-size: 17px;
  }

  .column-config-footer {
    min-height: 56px;
    padding: 8px 12px;
    gap: 6px;
  }

  .column-config-footer>span {
    display: none;
  }

  .column-config-footer button {
    min-width: 64px;
    padding: 0 9px;
  }

  .selected-hint:last-child {
    display: none;
  }
}

.rules-panel {
  margin-top: 18px;
  border: 1px solid #e0e7ed;
  border-radius: 9px;
  background: #fff;
}

.rules-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 18px;
  border-bottom: 1px solid #edf1f4;
}

.rules-heading div {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.rules-heading strong {
  color: #33485b;
  font-size: 13px;
}

.rules-heading span {
  color: #8997a4;
  font-size: 11px;
}

.rules-count {
  color: #6c8090 !important;
}

.rules-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.rules-actions .primary {
  height: 30px;
}

.rules-table-wrap {
  overflow-x: auto;
  overflow-y: visible;
}

.rules-table {
  min-width: 900px;
}

.rules-table th {
  position: sticky;
  top: 0;
  z-index: 3;
}

.rules-table td {
  white-space: normal;
  min-width: 140px;
}

.rule-category {
  display: inline-block;
  padding: 3px 8px;
  border-radius: 12px;
  color: #285d83;
  background: #eaf3f8;
  font-size: 11px;
}

.rule-dialog {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgb(20 36 50 / 35%);
}

.rule-form {
  width: min(520px, 100%);
  padding: 22px;
  border: 1px solid #dce5eb;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 20px 50px rgb(20 45 65 / 22%);
}

.rule-form-head,
.rule-form-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.rule-form-head {
  margin-bottom: 16px;
  color: #33485b;
  font-size: 16px;
}

.rule-form-head button {
  border: 0;
  background: transparent;
  font-size: 22px;
}

.rule-form label {
  display: block;
  margin-top: 11px;
  color: #5b6d7d;
  font-size: 12px;
}

.rule-form input,
.rule-form textarea {
  display: block;
  width: 100%;
  box-sizing: border-box;
  margin-top: 5px;
  padding: 8px 10px;
  border: 1px solid #d9e3e8;
  border-radius: 6px;
  color: #33485b;
  font: inherit;
}

.rule-form-actions {
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
}

.column-option {
  display: flex;
  flex: 1 1 auto;
  align-items: center;
  gap: 5px;
  min-width: 0;
  color: #526274;
  font-size: 11px;
  cursor: pointer;
}

.column-option span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.column-option input,
.column-pin input {
  flex: 0 0 auto;
  width: 13px;
  height: 13px;
  margin: 0;
  accent-color: #285d83;
}

.column-pin {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 3px;
  color: #788797;
  font-size: 10px;
  cursor: pointer;
}

.column-config-footer {
  padding-top: 7px;
  border-top: 1px solid #edf1f5;
}

.column-reset {
  height: 28px;
  color: #285d83;
}

.popover-title {
  padding: 6px 8px 7px;
  border-bottom: 1px solid #edf1f5;
  color: #506174;
  font-size: 11px;
  font-weight: 700;
}

.filter-check {
  display: inline-flex;
  width: 16px;
  height: 16px;
  flex: 0 0 16px;
  align-items: center;
  justify-content: center;
  margin-right: 8px;
  border: 1px solid #bac8d6;
  border-radius: 3px;
  color: #235c84;
}

.filter-option {
  width: 100%;
  height: 30px;
  padding: 0 9px;
  border: 0;
  border-radius: 5px;
  color: #526274;
  background: transparent;
  text-align: left;
  font-size: 12px;
}

.filter-option:hover {
  color: #285d83;
  background: #f1f6fa;
  border-color: transparent;
}

.filter-option.active {
  color: #285d83;
  background: #eaf2f8;
  font-weight: 600;
}

.filter-loading {
  padding: 9px 8px;
  color: #8997a4;
  font-size: 11px;
}

.filter-trigger {
  border: 0;
  background: transparent;
  color: #7c8b9a;
  padding: 0 3px;
  font-size: 14px;
  cursor: pointer;
}

.filter-trigger.active {
  color: #285d83;
}

.header-filter {
  width: 140px;
  height: 30px;
  border: 1px solid #dce4e7;
  padding: 0 8px;
}

.rate-edit-btn {
  color: #285d83;
}

.rate-dialog {
  position: fixed;
  inset: 0;
  z-index: 10000;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgb(18 30 44 / 32%);
  backdrop-filter: blur(2px);
}

.rate-card {
  width: min(430px, 100%);
  overflow: hidden;
  border: 1px solid #d9e1e8;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 24px 64px rgb(20 35 50 / 22%);
}

.rate-card-header {
  display: flex;
  justify-content: space-between;
  gap: 18px;
  padding: 22px 24px 18px;
  border-bottom: 1px solid #edf1f5;
}

.eyebrow {
  margin: 0 0 5px;
  color: #7890a3;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: .08em;
  text-transform: uppercase;
}

.rate-card h3 {
  margin: 0;
  color: #26384a;
  font-size: 18px;
  font-weight: 700;
}

.rate-card-header p:last-child {
  margin: 7px 0 0;
  color: #7b8997;
  font-size: 12px;
  line-height: 1.5;
}

.dialog-close {
  width: 30px;
  height: 30px;
  padding: 0;
  border: 0;
  color: #7b8997;
  background: transparent;
  font-size: 23px;
  line-height: 1;
}

.dialog-close:hover {
  color: #26384a;
  background: #f2f5f7;
  border-color: transparent;
}

.rate-card-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 22px 24px 24px;
}

.rate-card-body label {
  margin-top: 3px;
  color: #536577;
  font-size: 11px;
  font-weight: 700;
}

.rate-card-body select {
  width: 100%;
  height: 38px;
  box-sizing: border-box;
}

.rate-input-wrap {
  display: flex;
  align-items: center;
  height: 38px;
  overflow: hidden;
  border: 1px solid #d7e0e8;
  border-radius: 6px;
}

.rate-input-wrap:focus-within {
  border-color: #7ba1bd;
  box-shadow: 0 0 0 3px rgb(40 93 131 / 9%);
}

.rate-input-wrap span {
  padding-left: 11px;
  color: #738493;
  font-size: 12px;
  white-space: nowrap;
}

.rate-input-wrap input {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  padding: 0 10px;
  border: 0;
  outline: 0;
  color: #26384a;
  font-size: 13px;
}

.rate-hint {
  margin: 3px 0 0;
  color: #8a98a5;
  font-size: 11px;
  line-height: 1.5;
}

.rate-card-body .error {
  margin: 4px 0 0;
  font-size: 12px;
}

.rate-card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 24px;
  border-top: 1px solid #edf1f5;
  background: #fbfcfd;
}

.rate-card-actions button {
  min-width: 78px;
  height: 34px;
}

.rate-card-actions .primary {
  color: #fff;
  background: #285d83;
  border-color: #285d83;
}

.rate-card-actions .primary:hover {
  color: #fff;
  background: #204e70;
  border-color: #204e70;
}

.rate-card-actions .secondary {
  color: #596b7e;
}

/* TikTok column widths are proportional so wide screens do not stretch a single column. */
.platform-toolbar {
  position: relative;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  padding: 12px 18px;
  border-bottom: 1px solid #e3e8ef;
  background: #fbfcfd;
  border-radius: 8px 8px 0 0;
}

.platform-picker {
  position: relative;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  flex: 1 1 auto;
  min-width: 0;
}

.store-picker {
  position: relative;
  display: inline-flex;
  flex: 0 1 auto;
  align-items: center;
  width: max-content;
  min-width: 178px;
  max-width: min(480px, 100%);
  height: 36px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #7b8997;
  font-size: 12px;
}

.store-label {
  position: absolute;
  left: 12px;
  z-index: 1;
  color: #7b8997;
  white-space: nowrap;
  pointer-events: none;
}

.store-trigger {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  width: max-content;
  min-width: 178px;
  max-width: 100%;
  height: 36px;
  padding: 0 12px 0 58px;
  border: 1px solid #dce4ec;
  border-radius: 6px;
  background: #fff;
  color: #253f54;
  font-size: 13px;
  font-weight: 600;
}

.store-trigger strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.store-trigger.open {
  border-color: #8daec4;
  background: #f5f9fc;
}

.store-trigger .platform-label {
  color: #7b8997;
}

.store-trigger .platform-chevron {
  width: 16px;
  height: 16px;
  margin-left: auto;
}

.store-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  right: auto;
  z-index: 50;
  width: max-content;
  min-width: 178px;
  max-width: min(480px, calc(100vw - 24px));
  padding: 5px;
  border: 1px solid #dce4ec;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 24px rgb(25 45 65 / 12%);
  list-style: none;
}

.store-options {
  display: block;
  width: 100%;
  margin: 0;
  padding: 0;
  border: 0;
  outline: 0;
  list-style: none;
}

.store-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 36px;
  padding: 7px 10px;
  border-radius: 5px;
  color: #465669;
  cursor: pointer;
  list-style: none;
  white-space: nowrap;
}

.store-option.highlighted {
  background: #e6f0f7;
}

.store-option.selected {
  color: #235c84;
  background: #f1f6fa;
  font-weight: 600;
}

.store-selected-icon {
  display: block;
  width: 12px;
  height: 12px;
  flex: 0 0 12px;
  visibility: hidden;
}

.store-selected-icon.visible {
  visibility: visible;
}

.import-trigger {
  height: 36px;
  color: #fff;
  border-color: #285d83;
  background: #285d83;
}

.catalog-add-button {
  height: 36px;
  padding: 0 10px;
  color: #285d83;
  border: 1px solid #bfd0dc;
  border-radius: 6px;
  background: #fff;
  font-size: 12px;
}

.catalog-add-button:hover {
  background: #f1f6fa;
}

.catalog-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 24px 0;
  color: #536577;
  font-size: 12px;
  font-weight: 600;
}

.catalog-field input {
  appearance: none;
  box-sizing: border-box;
  width: 100%;
  height: 42px;
  padding: 0 12px;
  border: 1px solid #d7e0e8;
  border-radius: 7px;
  color: #26384a;
  background: #fff;
  font: inherit;
  font-size: 14px;
  font-weight: 400;
  outline: none;
}

.catalog-field input:focus {
  border-color: #285d83;
  box-shadow: 0 0 0 3px rgb(40 93 131 / 10%);
}

.catalog-kind-switch {
  display: flex;
  gap: 4px;
  padding: 4px;
  background: #f0f3f6;
  border-radius: 8px;
}

.catalog-kind-switch button {
  flex: 1;
  height: 34px;
  border: 0;
  background: transparent;
  color: #64748b;
  font-size: 13px;
}

.catalog-kind-switch button.active {
  background: #fff;
  color: #285d83;
  box-shadow: 0 1px 3px rgb(20 35 50 / 10%);
  font-weight: 600;
}

.catalog-error {
  padding: 0 24px;
  color: #b42318;
  font-size: 12px;
}

.catalog-panel .rule-form-actions {
  margin-top: 20px;
}

.catalog-panel {
  height: auto;
  max-height: none;
}

.import-trigger:hover {
  color: #fff;
  border-color: #204e70;
  background: #204e70;
}

.import-panel {
  width: min(720px, 100%);
  height: auto;
  max-height: min(760px, 94dvh);
  align-self: center;
  border-radius: 12px;
  box-shadow: 0 24px 70px rgb(20 35 50 / 22%);
}

.catalog-panel {
  width: min(440px, 100%);
  max-height: calc(100dvh - 48px);
  overflow-y: auto;
}

.catalog-panel button {
  appearance: none;
  padding: 0 14px;
  min-height: 34px;
  border: 1px solid #d7e0e8;
  border-radius: 6px;
  background: #fff;
  color: #465669;
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.catalog-panel button.primary {
  background: #285d83;
  border-color: #285d83;
  color: #fff;
}

.catalog-panel button:disabled {
  opacity: .55;
  cursor: default;
}

.catalog-panel .catalog-kind-switch button {
  border: 0;
  background: transparent;
}

.catalog-panel .catalog-kind-switch button.active {
  background: #fff;
  color: #285d83;
}

.import-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  padding: 18px 24px 4px;
}

.import-fields label {
  color: #536577;
  font-size: 12px;
  font-weight: 600;
}

.import-fields select {
  display: block;
  width: 100%;
  height: 38px;
  margin-top: 6px;
  padding: 0 10px;
  border: 1px solid #d7e0e8;
  border-radius: 6px;
  color: #26384a;
  background: #fff;
}

.import-file-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  max-height: 360px;
  padding: 14px 24px;
  overflow: auto;
}

.import-file-item {
  display: flex;
  flex-direction: column;
  gap: 7px;
  min-height: 96px;
  padding: 12px;
  border: 1px solid #dce5eb;
  border-radius: 8px;
  color: #33485b;
  background: #f9fbfc;
  font-size: 12px;
}

.import-file-item small {
  display: block;
  margin-top: 4px;
  color: #8a98a5;
  font-size: 10px;
  font-weight: 400;
}

.import-file-item input {
  width: 100%;
  font-size: 11px;
}

.import-file-item em {
  color: #6e8090;
  font-size: 11px;
  font-style: normal;
}

.import-success {
  margin: 0 24px;
  padding: 8px 10px;
  color: #39755a;
  background: #edf6f1;
  font-size: 12px;
}

.import-panel .history-heading {
  padding: 20px 24px 16px;
  background: #fbfcfd;
}

.import-panel .history-heading p {
  max-width: 480px;
  line-height: 1.5;
}

.import-panel .rule-form-actions {
  padding: 14px 24px 20px;
  border-top: 1px solid #edf1f5;
}

.platform-trigger {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 36px;
  min-width: 178px;
  padding: 0 12px;
  background: #fff;
}

.platform-trigger.open {
  border-color: #8daec4;
  background: #f5f9fc;
}

.platform-label {
  font-size: 12px;
  color: #7b8997;
}

.platform-trigger strong {
  color: #253f54;
  font-size: 13px;
  font-weight: 600;
}

.platform-chevron {
  display: block;
  flex: 0 0 16px;
  width: 16px;
  height: 16px;
  margin-left: auto;
  color: #728395;
}

.platform-chevron.open {
  transform: rotate(180deg);
}

.platform-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 40;
  width: 178px;
  min-width: 178px;
  padding: 5px;
  box-sizing: border-box;
  border: 1px solid #dce4ec;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 24px rgb(25 45 65 / 12%);
}

.platform-menu :deep(.platform-options) {
  display: block;
  width: 100%;
  list-style: none;
  margin: 0;
  padding: 0;
  border: 0;
  background: #fff;
  outline: none;
}

.platform-menu :deep(.platform-option) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-sizing: border-box;
  width: 100%;
  gap: 16px;
  min-height: 38px;
  margin: 0;
  padding: 8px 10px;
  border: 0;
  border-radius: 5px;
  font-size: 13px;
  line-height: 22px;
  color: #465669;
  white-space: nowrap;
  list-style: none;
  cursor: pointer;
  user-select: none;
}

.platform-menu :deep(.platform-option.selected) {
  color: #235c84;
  background: #f1f6fa;
  font-weight: 600;
}

.platform-menu :deep(.platform-option.highlighted) {
  background: #e6f0f7;
}

.platform-menu :deep(.platform-selected-icon) {
  display: block;
  width: 16px;
  height: 16px;
  flex: 0 0 16px;
  visibility: hidden;
}

.platform-menu :deep(.platform-selected-icon.visible) {
  visibility: visible;
}

@media (max-width: 600px) {
  .platform-toolbar {
    padding: 10px 12px;
  }

  .platform-picker {
    width: 100%;
  }

  .store-picker {
    max-width: 100%;
  }

  .store-trigger {
    max-width: 100%;
  }

  .store-menu {
    max-width: calc(100vw - 24px);
  }

  .history-actions {
    width: 100%;
  }

  .history-actions button {
    flex: 1;
  }

  .import-position {
    padding: 12px;
  }

  .import-fields,
  .import-file-grid {
    grid-template-columns: 1fr;
    padding-left: 16px;
    padding-right: 16px;
  }

  .import-panel .history-heading {
    padding-left: 16px;
    padding-right: 16px;
  }

  .import-panel .rule-form-actions {
    padding-left: 16px;
    padding-right: 16px;
  }
}

.tiktok-table-region {
  position: relative;
}

.tiktok-loading {
  position: absolute;
  inset: 49px 0 0;
  z-index: 8;
  display: grid;
  place-items: center;
  background: rgb(255 255 255 / 65%);
}

.tiktok-loading span {
  padding: 10px 18px;
  border: 1px solid #dfe5ec;
  border-radius: 8px;
  background: #fff;
  color: #62788b;
  font-size: 12px;
  box-shadow: 0 3px 12px rgb(30 50 70 / 6%);
}

.tiktok-table {
  table-layout: fixed;
}

.tiktok-table th,
.tiktok-table td {
  box-sizing: border-box;
}

.tiktok-table td:not(.empty) {
  height: 44px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tiktok-summary td:not(.empty) {
  overflow: visible;
  text-overflow: clip;
}

.tiktok-table th {
  height: 48px;
}

.tiktok-details {
  width: 100%;
  min-width: 1250px;
}

.tiktok-details .tk-month {
  width: 7%;
}

.tiktok-details .tk-quarter {
  width: 7%;
}

.tiktok-details .tk-date {
  width: 15%;
}

.tiktok-details .tk-business {
  width: 17%;
}

.tiktok-details .tk-amount {
  width: 12%;
}

.tiktok-details .tk-commission {
  width: 18%;
}

.tiktok-summary {
  min-width: 2540px;
  table-layout: auto;
}

.tiktok-summary :is(th, td) {
  min-width: 120px;
  white-space: nowrap;
}

.tiktok-summary thead th:first-child,
.tiktok-summary tbody td:first-child:not(.empty) {
  position: sticky;
  left: 0;
  z-index: 3;
  background: #fff;
  box-shadow: 3px 0 7px rgb(31 54 73 / 8%);
}

.tiktok-summary thead th:first-child {
  z-index: 6;
  background: #f4f6f9;
}

.tiktok-summary tfoot th:first-child {
  position: sticky;
  left: 0;
  z-index: 9;
  background: #edf3f7;
  box-shadow: 3px 0 7px rgb(31 54 73 / 8%);
}

.summary-pending {
  color: #7c8997;
  font-size: 11px;
  font-weight: 400;
}

.tiktok-summary thead tr:first-child {
  height: 64px;
}

.tiktok-summary thead tr:first-child th {
  height: 64px;
}

.tiktok-ads {
  min-width: 1546px;
}

.tiktok-settlements :is(th, td):first-child,
.tiktok-orders :is(th, td):first-child {
  position: sticky;
  left: 0;
  z-index: 3;
}

.tiktok-settlements :is(th, td):nth-child(2),
.tiktok-orders :is(th, td):nth-child(2) {
  position: sticky;
  left: 80px;
  z-index: 3;
  box-shadow: 3px 0 7px rgb(31 54 73 / 8%);
}

.tiktok-settlements :is(th, td):nth-child(3) {
  position: sticky;
  left: 166px;
  z-index: 3;
}

.tiktok-settlements :is(th, td):nth-child(4) {
  position: sticky;
  left: 276px;
  z-index: 3;
}

.tiktok-settlements :is(th, td):nth-child(5) {
  position: sticky;
  left: 386px;
  z-index: 3;
  box-shadow: 3px 0 7px rgb(31 54 73 / 8%);
}

.tiktok-settlements :is(th, td):nth-child(-n+5) {
  background: #fff;
}

.tiktok-settlements thead th:nth-child(-n+5),
.tiktok-orders thead th:nth-child(-n+2) {
  z-index: 6;
  background: #f4f6f9;
}

.tiktok-fbt {
  min-width: 2800px;
}

.tiktok-details :is(thead, tbody) th:nth-child(1),
.tiktok-details :is(thead, tbody) td:nth-child(1),
.tiktok-ads :is(thead, tbody) th:nth-child(1),
.tiktok-ads :is(thead, tbody) td:nth-child(1),
.tiktok-fbt :is(thead, tbody) th:nth-child(1),
.tiktok-fbt :is(thead, tbody) td:nth-child(1) {
  position: sticky;
  left: 0;
  z-index: 4;
  width: 80px;
  min-width: 80px;
  background: #fff;
}

.tiktok-details :is(thead, tbody) th:nth-child(2),
.tiktok-details :is(thead, tbody) td:nth-child(2),
.tiktok-ads :is(thead, tbody) th:nth-child(2),
.tiktok-ads :is(thead, tbody) td:nth-child(2),
.tiktok-fbt :is(thead, tbody) th:nth-child(2),
.tiktok-fbt :is(thead, tbody) td:nth-child(2) {
  position: sticky;
  left: 80px;
  z-index: 4;
  width: 86px;
  min-width: 86px;
  background: #fff;
  box-shadow: 3px 0 7px rgb(31 54 73 / 8%);
}

.tiktok-details thead th:nth-child(-n+2),
.tiktok-ads thead th:nth-child(-n+2),
.tiktok-fbt thead th:nth-child(-n+2) {
  z-index: 6;
  background: #f4f6f9;
}

.tiktok-table tfoot {
  position: sticky;
  bottom: 0;
  z-index: 8;
}

.tiktok-details tfoot th:first-child {
  position: sticky;
  left: 0;
  z-index: 9;
  background: #edf3f7;
}

.raw-table tfoot :is(th, td).pin-month,
.raw-table tfoot :is(th, td).pin-quarter,
.raw-table tfoot :is(th, td).pin-type,
.raw-table tfoot :is(th, td).pin-fee,
.raw-table tfoot :is(th, td).pin-user,
.tiktok-table tfoot :is(th, td) {
  background: #edf3f7;
  color: #30495f;
}

.raw-table tfoot :is(th, td).pin-month {
  color: #285d83;
}

.tiktok-settlements tfoot :is(th, td):nth-child(-n+5),
.tiktok-orders tfoot :is(th, td):nth-child(-n+2) {
  position: sticky;
  z-index: 7;
  background: #edf3f7;
}

.tiktok-settlements tfoot th:first-child {
  left: 0;
}

.tiktok-settlements tfoot td:nth-child(2) {
  left: 386px;
  box-shadow: 3px 0 7px rgb(31 54 73 / 8%);
}

.tiktok-settlements tfoot td:nth-child(n+3),
.tiktok-orders tfoot td:nth-child(n+2) {
  left: auto;
  z-index: 2;
  box-shadow: none;
}

.tiktok-orders tfoot th:first-child {
  left: 0;
}

.tiktok-details tfoot :is(th, td):first-child,
.tiktok-ads tfoot :is(th, td):first-child,
.tiktok-fbt tfoot :is(th, td):first-child,
.tiktok-summary tfoot :is(th, td):first-child {
  background: #edf3f7;
}

.raw-table tfoot>tr> :is(th, td),
.tiktok-table tfoot>tr> :is(th, td) {
  top: auto;
  background: #edf3f7;
}

.rate-pending {
  color: #9a762f;
  font-size: 12px;
  font-weight: 400;
}

.header-formula {
  display: block;
  margin-top: 4px;
  font-size: 11px;
  line-height: 1.4;
  font-weight: 400;
  color: #7c8997;
  white-space: nowrap;
}

.summary-subformula {
  display: block;
  margin-top: 4px;
  color: #8a96a3;
  font-size: 10px;
  font-weight: 400;
  white-space: nowrap;
}

.summary-header-label {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}

.formula-help {
  display: inline-grid;
  place-items: center;
  width: 18px;
  height: 18px;
  margin: 0;
  padding: 0;
  border: 1px solid #9aabba;
  border-radius: 50%;
  color: #536b80;
  background: #fff;
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  cursor: help;
  box-shadow: 0 1px 2px rgb(30 50 70 / 8%);
}

.formula-help:hover {
  border-color: #285d83;
  color: #fff;
  background: #285d83;
  box-shadow: 0 2px 5px rgb(40 93 131 / 24%);
}

.formula-help:focus-visible {
  outline: 2px solid #7ca8c7;
  outline-offset: 2px;
}

@media (max-width: 600px) {

  .tiktok-details th,
  .tiktok-details td {
    padding: 8px 6px;
  }

  .tiktok-toolbar {
    flex-wrap: wrap;
    padding: 0 12px 8px;
    gap: 4px 12px;
  }

  .tiktok-toolbar .tabs,
  .tiktok-toolbar .filters {
    width: auto;
  }

  .tiktok-toolbar .tabs {
    gap: 16px;
  }
}
</style>
