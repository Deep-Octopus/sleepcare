package notification

import (
	"context"
	"strings"

	careconfig "github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	notificationres "github.com/flipped-aurora/gin-vue-admin/server/model/notification/response"
)

func (s *NotificationService) GetProviderReadiness(ctx context.Context) (notificationres.ProviderReadiness, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return notificationres.ProviderReadiness{}, normalizeAccessError(err)
	}
	switch decision.RoleType {
	case caremodel.AuthorityRoleCareSteward, caremodel.AuthorityRoleClinician, caremodel.AuthorityRoleSupervisor:
	default:
		return notificationres.ProviderReadiness{}, notificationmodel.NewForbiddenError("当前角色无权查看通知供应商门禁")
	}
	return providerReadiness(s.providerConfig(), s.fixturesEnabled()), nil
}

func providerReadiness(config careconfig.NotificationProvider, fixtureDataEnabled bool) notificationres.ProviderReadiness {
	mode := strings.ToUpper(strings.TrimSpace(config.Mode))
	if mode == "" {
		mode = notificationmodel.ProviderModeDisabled
	}
	contractMode := mode == notificationmodel.ProviderModeContractTest && fixtureDataEnabled
	requestSigningConfigured := len(config.RequestSigningSecret) >= 32
	callbackVerificationConfigured := len(config.CallbackVerificationSecret) >= 32
	readiness := notificationres.ProviderReadiness{
		Mode:                           mode,
		ProviderCode:                   strings.TrimSpace(config.ProviderCode),
		UsageScope:                     notificationmodel.ProviderUsageTestOnly,
		ContractTestEnabled:            contractMode,
		NetworkDeliveryEnabled:         false,
		FormalDeliveryEnabled:          false,
		QualificationEvidenceReviewed:  config.QualificationEvidenceReviewed,
		TemplateEvidenceReviewed:       config.TemplateEvidenceReviewed,
		RequestSigningConfigured:       requestSigningConfigured,
		CallbackVerificationConfigured: callbackVerificationConfigured,
		ReceiptSemanticsReviewed:       config.ReceiptSemanticsReviewed,
		RetryPolicyReviewed:            config.RetryPolicyReviewed,
		FallbackReviewed:               config.FallbackReviewed,
		CostBoundaryReviewed:           config.CostBoundaryReviewed,
		RetryBoundary: notificationres.RetryBoundary{
			MaxAttemptsPerRequest: config.MaxAttemptsPerRequest,
		},
		RateBoundary: notificationres.RateBoundary{
			WindowSeconds: config.RateLimitWindowSeconds,
			MaxDispatches: config.RateLimitCount,
		},
		CostBoundary: notificationres.CostBoundary{
			Currency:            config.CostCurrency,
			EstimatedCostMinor:  config.EstimatedCostMinor,
			DailyCostLimitMinor: config.DailyCostLimitMinor,
		},
		Blockers: []string{},
	}
	checks := []struct {
		ready   bool
		blocker string
	}{
		{contractMode, "本地契约测试模式未启用"},
		{strings.TrimSpace(config.ProviderCode) != "", "供应商标识未配置"},
		{strings.TrimSpace(config.PolicyCode) != "" && config.PolicyVersion > 0, "不可变发送策略版本未配置"},
		{strings.TrimSpace(config.TemplateCode) != "", "模板证据未绑定"},
		{config.QualificationEvidenceReviewed, "主体资质证据未复核"},
		{config.TemplateEvidenceReviewed, "模板证据未复核"},
		{requestSigningConfigured, "请求签名未配置"},
		{callbackVerificationConfigured && config.CallbackMaxSkewSeconds >= 30, "回调验签未配置"},
		{config.ReceiptSemanticsReviewed, "回执语义未复核"},
		{config.RetryPolicyReviewed && config.MaxAttemptsPerRequest > 0, "重试边界未复核"},
		{config.FallbackReviewed, "备用流程未复核"},
		{config.RateLimitWindowSeconds > 0 && config.RateLimitCount > 0, "限流边界未配置"},
		{config.CostBoundaryReviewed && len(config.CostCurrency) == 3 && config.EstimatedCostMinor > 0 && config.DailyCostLimitMinor >= config.EstimatedCostMinor, "费用边界未复核"},
	}
	for _, check := range checks {
		if !check.ready {
			readiness.Blockers = append(readiness.Blockers, check.blocker)
		}
	}
	readiness.CallbackEndpointEnabled = len(readiness.Blockers) == 0
	return readiness
}

func providerDescriptorFromConfig(config careconfig.NotificationProvider) AdapterDescriptor {
	return AdapterDescriptor{
		Channel:                       notificationmodel.ChannelProviderContract,
		ProviderCode:                  strings.TrimSpace(config.ProviderCode),
		UsageScope:                    notificationmodel.ProviderUsageTestOnly,
		PolicyCode:                    strings.TrimSpace(config.PolicyCode),
		PolicyVersion:                 config.PolicyVersion,
		TemplateCode:                  strings.TrimSpace(config.TemplateCode),
		QualificationEvidenceReviewed: config.QualificationEvidenceReviewed,
		TemplateEvidenceReviewed:      config.TemplateEvidenceReviewed,
		ReceiptSemanticsReviewed:      config.ReceiptSemanticsReviewed,
		RetryPolicyReviewed:           config.RetryPolicyReviewed,
		FallbackReviewed:              config.FallbackReviewed,
		CostBoundaryReviewed:          config.CostBoundaryReviewed,
		MaxAttemptsPerRequest:         config.MaxAttemptsPerRequest,
		RateLimitWindowSeconds:        config.RateLimitWindowSeconds,
		RateLimitCount:                config.RateLimitCount,
		CostCurrency:                  strings.TrimSpace(config.CostCurrency),
		EstimatedCostMinor:            config.EstimatedCostMinor,
		DailyCostLimitMinor:           config.DailyCostLimitMinor,
	}
}
