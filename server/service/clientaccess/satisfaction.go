package clientaccess

import (
	"context"

	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	supervisionservice "github.com/flipped-aurora/gin-vue-admin/server/service/supervision"
)

func (s *ClientAccessService) ListSatisfactionRequests(
	ctx context.Context,
	req supervisionreq.ClientSatisfactionSearch,
) ([]supervisionres.ClientSatisfactionSummary, int64, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	list, total, err := s.supervision().ListClientSatisfactionRequests(ctx, satisfactionIdentity(identity), req)
	return list, total, normalizeSupervisionError(err)
}

func (s *ClientAccessService) GetSatisfactionRequest(
	ctx context.Context,
	id uint,
) (supervisionres.ClientSatisfactionDetail, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return supervisionres.ClientSatisfactionDetail{}, err
	}
	result, err := s.supervision().GetClientSatisfactionRequest(ctx, satisfactionIdentity(identity), id)
	return result, normalizeSupervisionError(err)
}

func (s *ClientAccessService) SubmitSatisfactionResponse(
	ctx context.Context,
	id uint,
	key string,
	req supervisionreq.SubmitSatisfactionResponse,
) (supervisionres.SubmitSatisfactionResult, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return supervisionres.SubmitSatisfactionResult{}, err
	}
	result, err := s.supervision().SubmitClientSatisfactionResponse(
		ctx,
		satisfactionIdentity(identity),
		id,
		key,
		req,
	)
	return result, normalizeSupervisionError(err)
}

func (s *ClientAccessService) supervision() *supervisionservice.SupervisionService {
	return &supervisionservice.SupervisionService{
		DB:                       s.db(),
		Now:                      s.Now,
		SyntheticFixturesEnabled: s.SyntheticFixturesEnabled,
	}
}

func satisfactionIdentity(identity SessionIdentity) supervisionservice.ClientSatisfactionIdentity {
	return supervisionservice.ClientSatisfactionIdentity{
		CareClientID: identity.CareClientID,
		DeptID:       identity.DeptID,
		Synthetic:    identity.Synthetic,
	}
}
