<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-4">
    <div class="mb-6 flex items-center justify-between gap-4">
      <ClientBackButton
        label="任务说明"
        @click="router.push({ name: 'ClientTask', params: { taskId } })"
      />
      <span class="flex items-center gap-1.5 text-xs text-muted-foreground">
        <svg-icon icon="lucide:cloud-check" />
        {{ saveCopy }}
      </span>
    </div>

    <ClientStatePanel
      v-if="loading"
      title="正在准备填写内容"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      title="填写内容暂不可用"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button class="!h-11 !rounded-xl" @click="loadQuestionnaire">重试</el-button>
    </ClientStatePanel>

    <template v-else-if="questionnaire">
      <div>
        <p class="text-sm font-medium text-primary">
          预计 {{ questionnaire.expectedMinutes || 1 }} 分钟
        </p>
        <h1 class="mt-3 text-[2rem] font-semibold leading-[1.15] tracking-[-0.04em]">
          {{ readableQuestionnaireTitle(questionnaire.title) }}
        </h1>
        <p
          v-if="questionnaire.purpose"
          class="mt-4 text-sm leading-6 text-muted-foreground"
        >
          {{ readableQuestionnairePurpose(questionnaire.purpose) }}
        </p>
      </div>

      <div class="mt-7 flex items-center justify-between gap-4 border-y border-border py-3 text-xs text-muted-foreground">
        <span>已填写 {{ completedCount }} / {{ questionnaire.questions.length }} 项</span>
        <span>带 * 的项目需要填写</span>
      </div>

      <div class="divide-y divide-border">
        <fieldset
          v-for="(question, index) in questionnaire.questions"
          :key="question.code"
          class="m-0 min-w-0 border-0 py-7"
        >
          <legend class="mb-4 block w-full text-base font-semibold leading-6">
            <span class="mb-1 block text-xs font-medium text-muted-foreground">第 {{ index + 1 }} 项</span>
            <span>{{ readableQuestionTitle(question.title) }}</span>
            <span v-if="question.required" class="ml-1 text-error">*</span>
          </legend>

          <el-radio-group
            v-if="question.type === 'SINGLE_CHOICE'"
            v-model="answers[question.code]"
            class="!flex !w-full !flex-col !items-stretch !gap-2"
          >
            <el-radio
              v-for="option in question.options"
              :key="option.code"
              :value="option.code"
              border
              class="!m-0 !h-auto !min-h-13 !w-full !rounded-xl !px-4"
            >
              {{ readableOptionLabel(option.label) }}
            </el-radio>
          </el-radio-group>

          <el-checkbox-group
            v-else-if="question.type === 'MULTIPLE_CHOICE'"
            v-model="answers[question.code]"
            class="!flex !w-full !flex-col !items-stretch !gap-2"
          >
            <el-checkbox
              v-for="option in question.options"
              :key="option.code"
              :value="option.code"
              border
              class="!m-0 !h-auto !min-h-13 !w-full !rounded-xl !px-4"
            >
              {{ readableOptionLabel(option.label) }}
            </el-checkbox>
          </el-checkbox-group>

          <el-input
            v-else-if="question.type === 'TEXT'"
            v-model="answers[question.code]"
            type="textarea"
            :maxlength="question.validation?.maxLength"
            :autosize="{ minRows: 3, maxRows: 6 }"
            show-word-limit
            placeholder="请输入"
          />

          <el-input-number
            v-else-if="question.type === 'NUMBER'"
            v-model="answers[question.code]"
            class="!w-full"
            :min="question.validation?.min"
            :max="question.validation?.max"
            controls-position="right"
          />

          <el-date-picker
            v-else-if="question.type === 'DATE'"
            v-model="answers[question.code]"
            class="!w-full"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="选择日期"
          />

          <el-radio-group
            v-else-if="question.type === 'BOOLEAN'"
            v-model="answers[question.code]"
            class="!grid !w-full !grid-cols-2 !gap-2"
          >
            <el-radio :value="true" border class="!m-0 !h-12 !w-full !rounded-xl !px-4">
              是
            </el-radio>
            <el-radio :value="false" border class="!m-0 !h-12 !w-full !rounded-xl !px-4">
              否
            </el-radio>
          </el-radio-group>
        </fieldset>
      </div>

      <ClientStatePanel
        v-if="validationMessage"
        class="mt-2"
        title="还有内容需要填写"
        :description="validationMessage"
        tone="danger"
        icon="lucide:circle-alert"
      />

      <div class="mt-8 grid grid-cols-[0.9fr_1.1fr] gap-3 border-t border-border pt-6">
        <el-button class="!m-0 !h-13 !rounded-xl" :loading="saving" @click="syncDraft">
          保存进度
        </el-button>
        <el-button type="primary" class="!m-0 !h-13 !rounded-xl !font-semibold" @click="goToConfirm">
          核对答案
        </el-button>
      </div>
    </template>
  </section>
</template>

<script setup>
  import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { getClientQuestionnaire, saveClientDraft } from '@/api/sleep-care/client-access'
  import { compactAnswers, newIdempotencyKey, readLocalDraft, unwrapClientResponse, writeLocalDraft } from './state'
  import {
    readableOptionLabel,
    readableQuestionnairePurpose,
    readableQuestionnaireTitle,
    readableQuestionTitle
  } from '@/utils/sleep-care-display'
  import ClientBackButton from '@/components/client-mobile/client-back-button.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'

  defineOptions({
    name: 'ClientTaskForm'
  })

  const route = useRoute()
  const router = useRouter()
  const taskId = Number(route.params.taskId)
  const loading = ref(true)
  const saving = ref(false)
  const questionnaire = ref(null)
  const answers = reactive({})
  const draftVersion = ref(0)
  const saveState = ref('idle')
  const errorMessage = ref('')
  const validationMessage = ref('')
  let hydrated = false
  let saveTimer = null
  let saveAgain = false

  const saveCopy = computed(() => ({
    idle: '更改会自动保存',
    saving: '正在保存…',
    saved: '进度已保存',
    offline: '已保存在本机，联网后重试',
    conflict: '其他页面已有更新'
  })[saveState.value])

  const initializeAnswerShapes = () => {
    for (const question of questionnaire.value.questions) {
      if (question.type === 'MULTIPLE_CHOICE' && !Array.isArray(answers[question.code])) {
        answers[question.code] = []
      }
    }
  }

  const loadQuestionnaire = async () => {
    loading.value = true
    errorMessage.value = ''
    hydrated = false
    try {
      questionnaire.value = unwrapClientResponse(await getClientQuestionnaire(taskId))
      const serverAnswers = questionnaire.value.draft?.answers || {}
      const local = readLocalDraft(taskId)
      const localIsNewer = local?.savedAt && (!questionnaire.value.draft?.savedAt || new Date(local.savedAt) > new Date(questionnaire.value.draft.savedAt))
      const initial = localIsNewer ? local.answers : serverAnswers
      Object.keys(answers).forEach((key) => delete answers[key])
      Object.assign(answers, initial)
      draftVersion.value = questionnaire.value.draft?.version || 0
      initializeAnswerShapes()
      await nextTick()
      hydrated = true
      saveState.value = questionnaire.value.draft ? 'saved' : 'idle'
    } catch (error) {
      errorMessage.value = error.message || '请稍后重试。'
    } finally {
      loading.value = false
    }
  }

  const syncDraft = async () => {
    if (!questionnaire.value) {
      return
    }
    if (saving.value) {
      saveAgain = true
      return
    }
    saving.value = true
    saveState.value = 'saving'
    const snapshot = compactAnswers(JSON.parse(JSON.stringify(answers)))
    try {
      const data = unwrapClientResponse(await saveClientDraft(taskId, newIdempotencyKey(), {
        expectedVersion: draftVersion.value,
        answers: snapshot
      }))
      draftVersion.value = data.version
      saveState.value = 'saved'
    } catch (error) {
      saveState.value = error.code === 41003 ? 'conflict' : 'offline'
    } finally {
      saving.value = false
      if (saveAgain) {
        saveAgain = false
        await syncDraft()
      }
    }
  }

  const scheduleSave = () => {
    writeLocalDraft(taskId, JSON.parse(JSON.stringify(answers)))
    saveState.value = navigator.onLine ? 'idle' : 'offline'
    if (saveTimer) {
      clearTimeout(saveTimer)
    }
    saveTimer = setTimeout(syncDraft, 900)
  }

  const answerIsEmpty = (value) => value === undefined || value === null || value === '' || (Array.isArray(value) && value.length === 0)

  const completedCount = computed(() => questionnaire.value?.questions.filter((question) => (
    !answerIsEmpty(answers[question.code])
  )).length || 0)

  const goToConfirm = async () => {
    const missing = questionnaire.value.questions.find((question) => question.required && answerIsEmpty(answers[question.code]))
    if (missing) {
      validationMessage.value = `请完成“${readableQuestionTitle(missing.title)}”`
      return
    }
    validationMessage.value = ''
    writeLocalDraft(taskId, JSON.parse(JSON.stringify(answers)))
    await syncDraft()
    if (saveState.value === 'conflict') {
      validationMessage.value = '进度已在其他页面更新，请刷新后再核对。'
      return
    }
    await router.push({ name: 'ClientTaskConfirm', params: { taskId } })
  }

  watch(answers, () => {
    if (hydrated) {
      validationMessage.value = ''
      scheduleSave()
    }
  }, { deep: true })

  const handleOnline = () => {
    if (saveState.value === 'offline') {
      syncDraft()
    }
  }

  onMounted(() => {
    window.addEventListener('online', handleOnline)
    loadQuestionnaire()
  })

  onBeforeUnmount(() => {
    window.removeEventListener('online', handleOnline)
    if (saveTimer) {
      clearTimeout(saveTimer)
    }
  })
</script>
