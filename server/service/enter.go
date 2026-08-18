package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/service/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/service/carepath"
	"github.com/flipped-aurora/gin-vue-admin/server/service/casework"
	"github.com/flipped-aurora/gin-vue-admin/server/service/clientaccess"
	"github.com/flipped-aurora/gin-vue-admin/server/service/example"
	"github.com/flipped-aurora/gin-vue-admin/server/service/media"
	"github.com/flipped-aurora/gin-vue-admin/server/service/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/service/supervision"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	CarePathServiceGroup      carepath.ServiceGroup
	CaseWorkServiceGroup      casework.ServiceGroup
	ClientAccessServiceGroup  clientaccess.ServiceGroup
	CareClientServiceGroup    careclient.ServiceGroup
	QuestionnaireServiceGroup questionnaire.ServiceGroup
	SupervisionServiceGroup   supervision.ServiceGroup
	SystemServiceGroup        system.ServiceGroup
	ExampleServiceGroup       example.ServiceGroup
	MediaServiceGroup         media.ServiceGroup
}
