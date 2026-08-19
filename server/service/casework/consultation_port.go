package casework

import (
	"context"

	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	"gorm.io/gorm"
)

// ConsultationClosedProjector is implemented by the quality domain. The port
// keeps casework independent from supervision while allowing both aggregates
// to commit on the caller's transaction.
type ConsultationClosedProjector interface {
	ProjectConsultationClosed(
		context.Context,
		*gorm.DB,
		caseworkmodel.Consultation,
		caseworkmodel.ConsultationInteraction,
	) error
}
