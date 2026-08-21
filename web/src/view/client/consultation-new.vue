<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-4">
    <ClientBackButton
      class="mb-7"
      label="返回咨询记录"
      @click="router.push({ name: 'ClientConsultations' })"
    />

    <p class="text-sm font-medium text-primary">联系服务团队</p>
    <h1 class="mt-3 text-[2rem] font-semibold tracking-[-0.04em]">发起在线咨询</h1>
    <p class="mt-4 text-sm leading-6 text-muted-foreground">
      请描述希望工作人员协助确认的事项。提交后可以继续补充信息并查看回复。
    </p>

    <ClientStatePanel
      class="mt-7"
      title="此处不提供急救服务"
      description="系统可随时接收留言，人工回复时间以服务安排为准。如遇紧急情况，请立即联系当地急救或前往正规医疗机构。"
      tone="warning"
      icon="lucide:triangle-alert"
    />

    <el-form
      class="client-service-form mt-7 border-t border-border"
      label-position="top"
      :model="form"
    >
      <el-form-item
        class="!mb-0 border-b border-border py-6"
        label="咨询主题"
        required
      >
        <el-input
          v-model="form.subject"
          maxlength="120"
          placeholder="例如：确认后续服务时间"
          show-word-limit
        />
      </el-form-item>

      <el-form-item
        class="!mb-0 border-b border-border py-6"
        label="希望如何联系"
        required
      >
        <el-radio-group
          v-model="form.urgency"
          class="grid w-full grid-cols-2 gap-3"
        >
          <el-radio-button
            class="!w-full"
            label="ROUTINE"
          >
            常规联系
          </el-radio-button>
          <el-radio-button
            class="!w-full"
            label="EXPEDITED"
          >
            优先联系
          </el-radio-button>
        </el-radio-group>
        <p class="mt-2 text-xs leading-5 text-muted-foreground">
          “优先联系”只用于安排处理顺序，不代表急救服务或固定响应时限。
        </p>
      </el-form-item>

      <el-form-item
        class="!mb-0 border-b border-border py-6"
        label="具体事项"
        required
      >
        <el-input
          v-model="form.message"
          maxlength="4000"
          placeholder="请说明需要工作人员协助确认的内容"
          :rows="7"
          show-word-limit
          type="textarea"
        />
      </el-form-item>
    </el-form>

    <ClientStatePanel
      v-if="errorMessage"
      class="mt-6"
      title="咨询没有提交"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    />

    <el-button
      type="primary"
      class="!mt-7 !h-14 !w-full !rounded-xl !text-base !font-semibold"
      :disabled="!canSubmit"
      :loading="submitting"
      @click="submit"
    >
      提交咨询
    </el-button>
  </section>
</template>

<script setup>
  import { computed, reactive, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { createClientConsultation } from '@/api/sleep-care/client-access'
  import { newIdempotencyKey, unwrapClientResponse } from './state'
  import ClientBackButton from '@/components/client-mobile/client-back-button.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'

  defineOptions({
    name: 'ClientConsultationNew'
  })

  const router = useRouter()
  const submitting = ref(false)
  const errorMessage = ref('')
  const form = reactive({
    subject: '',
    urgency: 'ROUTINE',
    message: ''
  })

  const canSubmit = computed(() => (
    form.subject.trim().length > 0 &&
    form.message.trim().length > 0 &&
    !submitting.value
  ))

  const submit = async () => {
    if (!canSubmit.value) {
      return
    }
    submitting.value = true
    errorMessage.value = ''
    try {
      const data = unwrapClientResponse(await createClientConsultation(newIdempotencyKey(), {
        subject: form.subject.trim(),
        urgency: form.urgency,
        message: form.message.trim()
      }))
      await router.replace({
        name: 'ClientConsultationDetail',
        params: { id: data.consultationId }
      })
    } catch (error) {
      errorMessage.value = error.message || '暂时无法提交，请稍后重试。'
    } finally {
      submitting.value = false
    }
  }
</script>

<style scoped>
  :deep(.client-service-form .el-form-item__label) {
    margin-bottom: 0.625rem;
    font-weight: 600;
    color: inherit;
  }

  :deep(.client-service-form .el-input__wrapper),
  :deep(.client-service-form .el-textarea__inner) {
    border-radius: 0.75rem;
  }
</style>
