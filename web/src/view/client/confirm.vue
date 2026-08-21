<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-4">
    <ClientBackButton
      class="mb-7"
      label="返回修改"
      @click="router.push({ name: 'ClientTaskForm', params: { taskId } })"
    />

    <ClientStatePanel
      v-if="loading"
      title="正在准备核对内容"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage && !questionnaire"
      title="暂时无法核对"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button class="!h-11 !rounded-xl" @click="load">重试</el-button>
    </ClientStatePanel>

    <template v-else-if="questionnaire">
      <p class="text-sm font-medium text-primary">最后一步</p>
      <h1 class="mt-3 text-[2rem] font-semibold tracking-[-0.04em]">核对后提交</h1>
      <p class="mt-4 text-sm leading-6 text-muted-foreground">
        请确认以下内容。返回修改不会丢失已经保存的进度。
      </p>

      <dl class="mt-7 border-y border-border">
        <div
          v-for="question in questionnaire.questions"
          :key="question.code"
          class="border-b border-border py-4 last:border-b-0"
        >
          <dt class="text-sm leading-6 text-muted-foreground">{{ readableQuestionTitle(question.title) }}</dt>
          <dd class="mt-1.5 break-words text-base font-medium leading-6">{{ answerLabel(question) }}</dd>
        </div>
      </dl>

      <ClientStatePanel
        v-if="errorMessage"
        class="mt-5"
        title="提交没有完成"
        :description="errorMessage"
        tone="danger"
        icon="lucide:circle-alert"
      />

      <el-button
        type="primary"
        class="!mt-7 !h-14 !w-full !rounded-xl !text-base !font-semibold"
        :loading="submitting"
        @click="submit"
      >
        确认提交
      </el-button>
      <div class="mt-4 flex items-center justify-center gap-1.5 text-center text-xs leading-5 text-muted-foreground">
        <svg-icon icon="lucide:shield-check" />
        <span>网络中断时可以安全重试，不会产生重复提交</span>
      </div>
    </template>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { getClientQuestionnaire, submitClientTask } from '@/api/sleep-care/client-access'
  import { clearLocalTaskState, compactAnswers, getOrCreateSubmitKey, readLocalDraft, unwrapClientResponse } from './state'
  import { readableOptionLabel, readableQuestionTitle } from '@/utils/sleep-care-display'
  import ClientBackButton from '@/components/client-mobile/client-back-button.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'

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
