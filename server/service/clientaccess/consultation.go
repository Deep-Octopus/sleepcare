package clientaccess

import (
	"context"

	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	caseworkservice "github.com/flipped-aurora/gin-vue-admin/server/service/casework"
)

func (s *ClientAccessService) CreateConsultation(
	ctx context.Context,
	key string,
	req caseworkreq.CreateConsultation,
) (caseworkres.ConsultationActionResult, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return caseworkres.ConsultationActionResult{}, err
	}
	result, err := s.caseWork().CreateClientConsultation(ctx, consultationIdentity(identity), key, req)
	return result, normalizeCaseWorkError(err)
}

func (s *ClientAccessService) ListConsultations(
	ctx context.Context,
	req caseworkreq.ClientConsultationSearch,
) ([]caseworkres.ClientConsultationSummary, int64, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	list, total, err := s.caseWork().ListClientConsultations(ctx, consultationIdentity(identity), req)
	return list, total, normalizeCaseWorkError(err)
}

func (s *ClientAccessService) GetConsultation(
	ctx context.Context,
	id uint,
) (caseworkres.ClientConsultationDetail, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return caseworkres.ClientConsultationDetail{}, err
	}
	result, err := s.caseWork().GetClientConsultation(ctx, consultationIdentity(identity), id)
	return result, normalizeCaseWorkError(err)
}

func (s *ClientAccessService) AddConsultationMessage(
	ctx context.Context,
	id uint,
	key string,
	req caseworkreq.AddClientConsultationMessage,
) (caseworkres.ConsultationActionResult, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return caseworkres.ConsultationActionResult{}, err
	}
	result, err := s.caseWork().AddClientConsultationMessage(ctx, consultationIdentity(identity), id, key, req)
	return result, normalizeCaseWorkError(err)
}

func (s *ClientAccessService) caseWork() *caseworkservice.CaseWorkService {
	return &caseworkservice.CaseWorkService{
		DB:                       s.db(),
		Now:                      s.Now,
		SyntheticFixturesEnabled: s.SyntheticFixturesEnabled,
	}
}

func consultationIdentity(identity SessionIdentity) caseworkservice.ClientConsultationIdentity {
	return caseworkservice.ClientConsultationIdentity{
		CareClientID: identity.CareClientID,
		DeptID:       identity.DeptID,
		Synthetic:    identity.Synthetic,
	}
}
