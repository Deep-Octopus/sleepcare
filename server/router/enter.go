package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/router/carepath"
	"github.com/flipped-aurora/gin-vue-admin/server/router/casework"
	"github.com/flipped-aurora/gin-vue-admin/server/router/clientaccess"
	"github.com/flipped-aurora/gin-vue-admin/server/router/example"
	"github.com/flipped-aurora/gin-vue-admin/server/router/media"
	"github.com/flipped-aurora/gin-vue-admin/server/router/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	CarePath      carepath.RouterGroup
	CaseWork      casework.RouterGroup
	ClientAccess  clientaccess.RouterGroup
	CareClient    careclient.RouterGroup
	Questionnaire questionnaire.RouterGroup
	System        system.RouterGroup
	Example       example.RouterGroup
	Media         media.RouterGroup
}
