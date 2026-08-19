<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-5">
    <button
      type="button"
      class="mb-6 inline-flex min-h-10 items-center gap-2 rounded-lg px-1 text-sm text-[#47766b] focus-visible:outline-2 focus-visible:outline-[#2c806c] dark:text-emerald-300"
      @click="router.push({ name: 'ClientTaskForm', params: { taskId } })"
    >
      <span aria-hidden="true">←</span>
      返回修改
    </button>

    <div v-if="loading" class="rounded-2xl border border-[#dce8e3] p-5 text-sm dark:border-slate-800">
      正在准备核对内容…
    </div>

    <div v-else-if="errorMessage && !questionnaire" class="rounded-2xl border border-red-200 bg-red-50 p-5 dark:border-red-950 dark:bg-red-950/40">
      <p class="font-medium text-red-700 dark:text-red-200">暂时无法核对</p>
      <p class="mt-2 text-sm leading-6 text-red-600 dark:text-red-300">{{ errorMessage }}</p>
      <el-button class="!mt-4 !h-11 !rounded-xl" @click="load">重试</el-button>
    </div>

    <template v-else-if="questionnaire">
      <p class="text-sm font-medium text-[#47766b] dark:text-emerald-300">最后一步</p>
      <h1 class="mt-2 text-[1.8rem] font-semibold tracking-[-0.035em]">核对后提交</h1>
      <p class="mt-3 text-sm leading-6 text-[#60766f] dark:text-slate-300">
        请确认以下内容。返回修改不会丢失已经保存的进度。
      </p>

      <div class="mt-7 overflow-hidden rounded-2xl border border-[#dce8e3] dark:border-slate-800">
        <div
          v-for="(question, index) in questionnaire.questions"
          :key="question.code"
          class="p-4"
          :class="index ? 'border-t border-[#e5ece9] dark:border-slate-800' : ''"
        >
          <p class="text-sm leading-6 text-[#60766f] dark:text-slate-400">{{ readableQuestionTitle(question.title) }}</p>
          <p class="mt-1.5 break-words text-base font-medium leading-6">{{ answerLabel(question) }}</p>
        </div>
      </div>

      <p v-if="errorMessage" class="mt-5 rounded-xl bg-red-50 p-3 text-sm leading-6 text-red-700 dark:bg-red-950/40 dark:text-red-200">
        {{ errorMessage }}
      </p>

      <el-button
        type="primary"
        class="!mt-7 !h-13 !w-full !rounded-xl !text-base"
        :loading="submitting"
        @click="submit"
      >
        确认提交
      </el-button>
      <p class="mt-3 text-center text-xs leading-5 text-[#7b8e89] dark:text-slate-500">
        网络中断时可以安全重试，不会产生重复提交
      </p>
    </template>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { getClientQuestionnaire, submitClientTask } from '@/api/sleep-care/client-access'
  import { clearLocalTaskState, compactAnswers, getOrCreateSubmitKey, readLocalDraft, unwrapClientResponse } from './state'
  import { readableOptionLabel, readableQuestionTitle } from '@/utils/sleep-care-display'

  defineOptions({
    name: 'ClientTaskConfirm'
  })

  const route = useRoute()
  const router = useRouter()
  const taskId = Number(route.params.taskId)
  const loading = ref(true)
  const submitting = ref(false)
  const questionnaire = ref(null)
  const answers = ref({})
  const errorMessage = ref('')

  const load = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      questionnaire.value = unwrapClientResponse(await getClientQuestionnaire(taskId))
      const local = readLocalDraft(taskId)
      answers.value = compactAnswers(local?.answers || questionnaire.value.draft?.answers || {})
      if (!local && !questionnaire.value.draft) {
        await router.replace({ name: 'ClientTaskForm', params: { taskId } })
      }
    } catch (error) {
      errorMessage.value = error.message || '请稍后重试。'
    } finally {
      loading.value = false
    }
  }

  const answerLabel = (question) => {
    const value = answers.value[question.code]
    if (question.type === 'SINGLE_CHOICE') {
      const option = question.options.find((item) => item.code === value)
      return option ? readableOptionLabel(option.label) : '未填写'
    }
    if (question.type === 'MULTIPLE_CHOICE') {
      const values = Array.isArray(value) ? value : []
      return values.map((code) => {
        const option = question.options.find((item) => item.code === code)
        return option ? readableOptionLabel(option.label) : '未识别选项'
      }).join('、') || '未填写'
    }
    if (question.type === 'BOOLEAN') {
      return typeof value === 'boolean' ? (value ? '是' : '否') : '未填写'
    }
    return value === undefined || value === null || value === '' ? '未填写' : String(value)
  }

  const submit = async () => {
    submitting.value = true
    errorMessage.value = ''
    try {
      unwrapClientResponse(await submitClientTask(taskId, getOrCreateSubmitKey(taskId), {
        expectedTaskVersion: questionnaire.value.taskVersion,
        source: 'CLIENT_SELF',
        answers: compactAnswers(answers.value),
        clientOccurredAt: new Date().toISOString()
      }))
      clearLocalTaskState(taskId)
      await router.replace({ name: 'ClientTaskSuccess', params: { taskId } })
    } catch (error) {
      errorMessage.value = error.message || '提交未完成，请检查网络后重试。'
    } finally {
      submitting.value = false
    }
  }

  onMounted(load)
</script>
