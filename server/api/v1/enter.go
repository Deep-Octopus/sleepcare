package v1

import (
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/carepath"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/casework"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/clientaccess"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/example"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/media"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/supervision"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/system"
)

var ApiGroupApp = new(ApiGroup)

type ApiGroup struct {
	CarePathApiGroup      carepath.ApiGroup
	CaseWorkApiGroup      casework.ApiGroup
	ClientAccessApiGroup  clientaccess.ApiGroup
	CareClientApiGroup    careclient.ApiGroup
	QuestionnaireApiGroup questionnaire.ApiGroup
	SupervisionApiGroup   supervision.ApiGroup
	SystemApiGroup        system.ApiGroup
	ExampleApiGroup       example.ApiGroup
	MediaApiGroup         media.ApiGroup
}
