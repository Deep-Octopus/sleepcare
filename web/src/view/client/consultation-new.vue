<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-6">
    <button
      type="button"
      class="mb-6 inline-flex min-h-10 items-center gap-2 rounded-lg px-1 text-sm text-primary focus-visible:outline-2 focus-visible:outline-primary"
      @click="router.push({ name: 'ClientConsultations' })"
    >
      <span aria-hidden="true">←</span>
      返回咨询记录
    </button>

    <p class="text-sm text-muted-foreground">联系服务团队</p>
    <h1 class="mt-1 text-[1.8rem] font-semibold tracking-[-0.035em]">发起在线咨询</h1>
    <p class="mt-3 text-sm leading-6 text-muted-foreground">
      请描述希望工作人员协助确认的事项。提交后可以继续补充信息并查看回复。
    </p>

    <div class="mt-6 rounded-2xl border border-warning/30 bg-warning/8 p-4 text-sm leading-6 text-base-text">
      系统可随时接收；人工回复时间以服务安排为准。如遇紧急情况，请立即联系当地急救或前往正规医疗机构，本页面不提供急救服务。
    </div>

    <el-form
      class="mt-7"
      label-position="top"
      :model="form"
    >
      <el-form-item
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

    <p
      v-if="errorMessage"
      class="mt-2 text-sm leading-6 text-error"
    >
      {{ errorMessage }}
    </p>

    <el-button
      type="primary"
      class="!mt-4 !h-13 !w-full !rounded-xl !text-base"
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
