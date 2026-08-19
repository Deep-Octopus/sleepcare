<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-5">
    <div class="mb-6 flex items-center justify-between gap-4">
      <button
        type="button"
        class="inline-flex min-h-10 items-center gap-2 rounded-lg px-1 text-sm text-[#47766b] focus-visible:outline-2 focus-visible:outline-[#2c806c] dark:text-emerald-300"
        @click="router.push({ name: 'ClientTask', params: { taskId } })"
      >
        <span aria-hidden="true">←</span>
        任务说明
      </button>
      <span class="text-xs text-[#71847e] dark:text-slate-400">{{ saveCopy }}</span>
    </div>

    <div v-if="loading" class="rounded-2xl border border-[#dce8e3] p-5 text-sm dark:border-slate-800">
      正在准备表单…
    </div>

    <div v-else-if="errorMessage" class="rounded-2xl border border-red-200 bg-red-50 p-5 dark:border-red-950 dark:bg-red-950/40">
      <p class="font-medium text-red-700 dark:text-red-200">表单暂不可用</p>
      <p class="mt-2 text-sm leading-6 text-red-600 dark:text-red-300">{{ errorMessage }}</p>
      <el-button class="!mt-4 !h-11 !rounded-xl" @click="loadQuestionnaire">重试</el-button>
    </div>

    <template v-else-if="questionnaire">
      <div class="border-b border-[#e5ece9] pb-6 dark:border-slate-800">
        <p class="text-sm font-medium text-[#47766b] dark:text-emerald-300">
          预计 {{ questionnaire.expectedMinutes || 1 }} 分钟
        </p>
        <h1 class="mt-2 text-[1.75rem] font-semibold leading-tight tracking-[-0.035em]">{{ readableQuestionnaireTitle(questionnaire.title) }}</h1>
        <p v-if="questionnaire.purpose" class="mt-3 text-sm leading-6 text-[#60766f] dark:text-slate-300">{{ readableQuestionnairePurpose(questionnaire.purpose) }}</p>
      </div>

      <div class="mt-7 space-y-7">
        <fieldset
          v-for="(question, index) in questionnaire.questions"
          :key="question.code"
          class="m-0 min-w-0 border-0 p-0"
        >
          <legend class="mb-3 block w-full text-base font-semibold leading-6">
            <span class="mr-2 text-sm font-medium text-[#47766b] dark:text-emerald-300">{{ String(index + 1).padStart(2, '0') }}</span>
            {{ readableQuestionTitle(question.title) }}
            <span v-if="question.required" class="ml-1 text-red-500">*</span>
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
              class="!m-0 !h-auto !min-h-12 !w-full !rounded-xl !px-4"
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
              class="!m-0 !h-auto !min-h-12 !w-full !rounded-xl !px-4"
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

      <p v-if="validationMessage" class="mt-6 rounded-xl bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/40 dark:text-red-200">
        {{ validationMessage }}
      </p>

      <div class="mt-8 grid grid-cols-[0.9fr_1.1fr] gap-3">
        <el-button class="!m-0 !h-12 !rounded-xl" :loading="saving" @click="syncDraft">
          保存进度
        </el-button>
        <el-button type="primary" class="!m-0 !h-12 !rounded-xl" @click="goToConfirm">
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
