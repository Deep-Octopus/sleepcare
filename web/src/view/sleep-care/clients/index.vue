<template>
  <div>
    <div class="mb-4 border border-amber-300 rounded-lg bg-amber-50 px-4 py-3 text-amber-900">
      <div class="font-semibold">合成数据开发区</div>
      <div class="mt-1 text-sm">
        P1-02 只承载康养用户公开资料、合成测试授权和责任关系，不包含医疗内容、真实短信、康养用户账号或 AI 能力。
      </div>
    </div>

    <div class="gva-search-box">
      <el-form :inline="true" :model="searchForm">
        <el-form-item label="编码或名称">
          <el-input v-model="searchForm.keyword" clearable placeholder="仅搜索合成资料" @keyup.enter="search" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" clearable class="w-36">
            <el-option label="服务中" value="ACTIVE" />
            <el-option label="已停用" value="INACTIVE" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="search">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button v-if="btnAuth.createClient" type="primary" @click="openCreate">新建合成用户</el-button>
      </div>
      <el-table v-loading="loading" :data="tableData" row-key="id" empty-text="当前责任或组织范围内暂无康养用户">
        <el-table-column prop="displayCode" label="显示编码" min-width="150" />
        <el-table-column prop="displayName" label="显示名称" min-width="150" />
        <el-table-column prop="organizationName" label="机构" min-width="170" />
        <el-table-column prop="teamName" label="团队" min-width="150" />
        <el-table-column prop="contactMobile" label="联系电话" min-width="130" />
        <el-table-column label="责任关系" min-width="220">
          <template #default="scope">
            <div v-if="scope.row.currentAssignments?.length" class="flex flex-col gap-1">
              <span v-for="item in scope.row.currentAssignments" :key="item.id">
                {{ roleLabel(item.roleType) }}：{{ item.assigneeDisplayName }}
              </span>
            </div>
            <span v-else class="text-gray-400">未分配</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'ACTIVE' ? 'success' : 'info'">
              {{ scope.row.status === 'ACTIVE' ? '服务中' : '已停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="数据" width="90">
          <template #default="scope">
            <el-tag v-if="scope.row.synthetic" type="warning">合成</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" min-width="290">
          <template #default="scope">
            <el-button v-if="btnAuth.viewDetail" link type="primary" @click="openDetail(scope.row.id)">详情</el-button>
            <el-button v-if="btnAuth.maintainClient" link type="primary" @click="openEdit(scope.row)">维护</el-button>
            <el-button v-if="btnAuth.assignCare" link type="primary" @click="openAssignment(scope.row.id)">分配责任</el-button>
            <el-button v-if="btnAuth.recordConsent" link type="primary" @click="openConsent(scope.row.id)">授权留痕</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="gva-pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="loadTable"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <el-drawer v-model="detailVisible" title="康养用户详情" size="680px">
      <template v-if="detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="醒目标识"><el-tag type="warning">合成数据</el-tag></el-descriptions-item>
          <el-descriptions-item label="版本">{{ detail.version }}</el-descriptions-item>
          <el-descriptions-item label="显示编码">{{ detail.displayCode }}</el-descriptions-item>
          <el-descriptions-item label="显示名称">{{ detail.displayName }}</el-descriptions-item>
          <el-descriptions-item label="联系电话">{{ detail.contactMobile || '-' }}</el-descriptions-item>
          <el-descriptions-item label="服务包">{{ detail.servicePackageCode || '-' }}</el-descriptions-item>
          <el-descriptions-item label="机构">{{ detail.organizationName }}</el-descriptions-item>
          <el-descriptions-item label="团队">{{ detail.teamName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="非医疗服务原因" :span="2">{{ detail.serviceReason || '-' }}</el-descriptions-item>
        </el-descriptions>

        <h3 class="mt-6 mb-2 font-semibold">责任关系历史</h3>
        <el-table :data="detail.assignments" size="small" empty-text="暂无责任关系">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column label="角色" width="110"><template #default="scope">{{ roleLabel(scope.row.roleType) }}</template></el-table-column>
          <el-table-column prop="assigneeDisplayName" label="责任人" min-width="120" />
          <el-table-column prop="teamName" label="团队" min-width="130" />
          <el-table-column label="状态" width="90"><template #default="scope">{{ assignmentStatusLabel(scope.row.status) }}</template></el-table-column>
          <el-table-column label="生效时间" min-width="170"><template #default="scope">{{ formatDateTime(scope.row.validFrom) }}</template></el-table-column>
        </el-table>

        <h3 class="mt-6 mb-2 font-semibold">合成测试授权历史</h3>
        <el-alert class="mb-2" type="warning" :closable="false" title="这些记录仅用于技术流程验证，不代表真实用户授权。" />
        <el-table :data="detail.consentRecords" size="small" empty-text="暂无授权记录">
          <el-table-column label="动作" width="90"><template #default="scope">{{ scope.row.action === 'GRANT' ? '授权' : '撤回' }}</template></el-table-column>
          <el-table-column prop="textVersion" label="文本版本" min-width="120" />
          <el-table-column prop="source" label="来源" min-width="130" />
          <el-table-column label="发生时间" min-width="170"><template #default="scope">{{ formatDateTime(scope.row.occurredAt) }}</template></el-table-column>
          <el-table-column prop="reason" label="备注" min-width="180" />
        </el-table>
      </template>
    </el-drawer>

    <el-dialog v-model="clientDialogVisible" :title="editingId ? '维护合成康养用户' : '新建合成康养用户'" width="620px">
      <el-alert class="mb-4" type="warning" :closable="false" title="只允许录入合成公开资料，请勿输入真实个人信息或医疗内容。" />
      <el-form :model="clientForm" label-width="110px">
        <el-form-item v-if="!editingId" label="显示编码" required><el-input v-model="clientForm.displayCode" placeholder="例如 SYN-CLIENT-A003" /></el-form-item>
        <el-form-item label="显示名称" required><el-input v-model="clientForm.displayName" placeholder="必须包含合成标识" /></el-form-item>
        <el-form-item label="联系电话"><el-input v-model="clientForm.contactMobile" placeholder="仅合成号码" /></el-form-item>
        <el-form-item label="服务包编码"><el-input v-model="clientForm.servicePackageCode" placeholder="非医疗服务包编码" /></el-form-item>
        <el-form-item label="服务原因"><el-input v-model="clientForm.serviceReason" type="textarea" :rows="3" placeholder="仅非医疗、合成说明" /></el-form-item>
        <template v-if="!editingId">
          <el-form-item label="机构" required>
            <el-select v-model="clientForm.organizationId" class="w-full" @change="clientForm.teamId = undefined">
              <el-option v-for="item in organizationOptions" :key="item.departmentId" :label="item.name" :value="item.departmentId" />
            </el-select>
          </el-form-item>
        </template>
        <el-form-item label="团队">
          <el-select v-model="clientForm.teamId" clearable class="w-full">
            <el-option v-for="item in teamOptions" :key="item.departmentId" :label="item.name" :value="item.departmentId" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="editingId" label="状态">
          <el-radio-group v-model="clientForm.status"><el-radio value="ACTIVE">服务中</el-radio><el-radio value="INACTIVE">已停用</el-radio></el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="clientDialogVisible = false">取消</el-button><el-button type="primary" @click="saveClient">保存</el-button></template>
    </el-dialog>

    <el-dialog v-model="assignmentVisible" title="记录责任关系" width="580px">
      <el-alert class="mb-4" type="info" :closable="false" title="每个康养用户同一时间最多一名管家和一名医护；转交必须明确替代旧关系。" />
      <el-form :model="assignmentForm" label-width="110px">
        <el-form-item label="责任类型" required>
          <el-select v-model="assignmentForm.roleType" class="w-full" @change="assignmentForm.assigneeId = undefined">
            <el-option label="健康管家" value="CARE_STEWARD" /><el-option label="一线医护" value="CLINICIAN" />
          </el-select>
        </el-form-item>
        <el-form-item label="责任人" required>
          <el-select v-model="assignmentForm.assigneeId" class="w-full">
            <el-option v-for="item in assignmentAssignees" :key="item.id" :label="`${item.displayName}（${teamName(item.teamId)}）`" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="生效时间" required><el-date-picker v-model="assignmentForm.validFrom" type="datetime" class="w-full" /></el-form-item>
        <el-form-item label="转交原因" required><el-input v-model="assignmentForm.reason" type="textarea" :rows="3" placeholder="合成测试责任调整原因" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="assignmentVisible = false">取消</el-button><el-button type="primary" @click="saveAssignment">确认记录</el-button></template>
    </el-dialog>

    <el-dialog v-model="consentVisible" title="合成测试授权留痕" width="560px">
      <el-alert class="mb-4" type="warning" :closable="false" title="仅记录合成测试参与授权，不得作为真实法律或医疗授权。" />
      <el-form :model="consentForm" label-width="110px">
        <el-form-item label="动作" required><el-radio-group v-model="consentForm.action"><el-radio value="GRANT">授权</el-radio><el-radio value="WITHDRAW">撤回</el-radio></el-radio-group></el-form-item>
        <el-form-item label="文本版本" required><el-input v-model="consentForm.textVersion" /></el-form-item>
        <el-form-item label="发生时间" required><el-date-picker v-model="consentForm.occurredAt" type="datetime" class="w-full" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="consentForm.reason" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="consentVisible = false">取消</el-button><el-button type="primary" @click="saveConsent">确认记录</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useBtnAuth } from '@/utils/btnAuth'
  import {
    createCareAssignment,
    createCareClient,
    createCareConsent,
    getCareClient,
    getCareClientOptions,
    getCareClients,
    updateCareClient
  } from '@/api/sleep-care/care-clients'

  defineOptions({ name: 'CareClients' })

  const btnAuth = useBtnAuth()
  const searchForm = reactive({ keyword: '', status: '' })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detail = ref(null)
  const detailVisible = ref(false)
  const options = ref({ orgUnits: [], assignees: [] })
  const optionsLoaded = ref(false)
  const editingId = ref(0)
  const editingVersion = ref(0)
  const clientDialogVisible = ref(false)
  const assignmentVisible = ref(false)
  const consentVisible = ref(false)
  const actionClient = ref(null)

  const emptyClientForm = () => ({ displayCode: '', displayName: '[合成] ', contactMobile: '', serviceReason: '', servicePackageCode: '', organizationId: undefined, teamId: undefined, status: 'ACTIVE' })
  const clientForm = reactive(emptyClientForm())
  const assignmentForm = reactive({ roleType: 'CARE_STEWARD', assigneeId: undefined, validFrom: new Date(), reason: '' })
  const consentForm = reactive({ action: 'GRANT', textVersion: 'SYNTHETIC-V1', occurredAt: new Date(), reason: '合成测试授权记录' })

  const organizationOptions = computed(() => options.value.orgUnits.filter((item) => item.unitType === 'ORGANIZATION'))
  const teamOptions = computed(() => options.value.orgUnits.filter((item) => item.unitType === 'TEAM' && (!clientForm.organizationId || item.organizationId === clientForm.organizationId)))
  const assignmentAssignees = computed(() => options.value.assignees.filter((item) => item.roleType === assignmentForm.roleType && item.teamId === actionClient.value?.teamId))

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getCareClients({ page: page.value, pageSize: pageSize.value, ...searchForm })
      if (res.code === 0) {
        tableData.value = res.data.list || []
        total.value = res.data.total
      }
    } finally {
      loading.value = false
    }
  }
  const search = () => { page.value = 1; loadTable() }
  const resetSearch = () => { searchForm.keyword = ''; searchForm.status = ''; search() }
  const handleSizeChange = () => { page.value = 1; loadTable() }
  const ensureOptions = async () => {
    if (optionsLoaded.value) return true
    const res = await getCareClientOptions()
    if (res.code !== 0) return false
    options.value = res.data
    optionsLoaded.value = true
    return true
  }
  const loadDetail = async (id) => {
    const res = await getCareClient(id)
    if (res.code !== 0) return null
    detail.value = res.data
    return res.data
  }
  const openDetail = async (id) => { if (await loadDetail(id)) detailVisible.value = true }
  const openCreate = async () => {
    if (!(await ensureOptions())) return
    Object.assign(clientForm, emptyClientForm())
    editingId.value = 0
    editingVersion.value = 0
    clientDialogVisible.value = true
  }
  const openEdit = async (row) => {
    if (!(await ensureOptions())) return
    Object.assign(clientForm, { displayName: row.displayName, contactMobile: row.contactMobile, serviceReason: row.serviceReason, servicePackageCode: row.servicePackageCode, organizationId: row.organizationId, teamId: row.teamId, status: row.status })
    editingId.value = row.id
    editingVersion.value = row.version
    clientDialogVisible.value = true
  }
  const saveClient = async () => {
    if (!clientForm.displayName?.includes('合成')) return ElMessage.warning('显示名称必须醒目标注“合成”')
    let res
    if (editingId.value) {
      res = await updateCareClient(editingId.value, { expectedVersion: editingVersion.value, displayName: clientForm.displayName, contactMobile: clientForm.contactMobile, serviceReason: clientForm.serviceReason, servicePackageCode: clientForm.servicePackageCode, teamId: clientForm.teamId, status: clientForm.status })
    } else {
      if (!clientForm.displayCode || !clientForm.organizationId) return ElMessage.warning('显示编码和机构必填')
      res = await createCareClient({ ...clientForm, synthetic: true })
    }
    if (res.code === 0) { ElMessage.success(res.msg); clientDialogVisible.value = false; await loadTable() }
  }
  const openAssignment = async (id) => {
    if (!(await ensureOptions())) return
    actionClient.value = await loadDetail(id)
    if (!actionClient.value) return
    Object.assign(assignmentForm, { roleType: 'CARE_STEWARD', assigneeId: undefined, validFrom: new Date(), reason: '' })
    assignmentVisible.value = true
  }
  const saveAssignment = async () => {
    if (!assignmentForm.assigneeId || !assignmentForm.reason) return ElMessage.warning('责任人和原因必填')
    const assignee = options.value.assignees.find((item) => item.id === assignmentForm.assigneeId)
    const current = actionClient.value.assignments.find((item) => item.roleType === assignmentForm.roleType && item.status === 'ACTIVE')
    const res = await createCareAssignment(actionClient.value.id, { expectedVersion: actionClient.value.version, roleType: assignmentForm.roleType, assigneeId: assignmentForm.assigneeId, teamId: assignee.teamId, validFrom: assignmentForm.validFrom.toISOString(), replacesAssignmentId: current?.id, reason: assignmentForm.reason })
    if (res.code === 0) { ElMessage.success(res.msg); assignmentVisible.value = false; await loadTable() }
  }
  const openConsent = async (id) => {
    actionClient.value = await loadDetail(id)
    if (!actionClient.value) return
    Object.assign(consentForm, { action: actionClient.value.consentStatus === 'GRANTED' ? 'WITHDRAW' : 'GRANT', textVersion: 'SYNTHETIC-V1', occurredAt: new Date(), reason: '合成测试授权记录' })
    consentVisible.value = true
  }
  const saveConsent = async () => {
    const res = await createCareConsent(actionClient.value.id, { expectedVersion: actionClient.value.version, consentType: 'SYNTHETIC_TEST_PARTICIPATION', action: consentForm.action, textVersion: consentForm.textVersion, occurredAt: consentForm.occurredAt.toISOString(), source: 'STAFF_RECORDED', reason: consentForm.reason })
    if (res.code === 0) { ElMessage.success(res.msg); consentVisible.value = false; await loadTable() }
  }
  const roleLabel = (value) => value === 'CARE_STEWARD' ? '健康管家' : '一线医护'
  const assignmentStatusLabel = (value) => ({ ACTIVE: '生效中', SCHEDULED: '待生效', ENDED: '已结束', CANCELLED: '已取消' }[value] || value)
  const formatDateTime = (value) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
  const teamName = (id) => options.value.orgUnits.find((item) => item.departmentId === id)?.name || id

  onMounted(loadTable)
</script>
