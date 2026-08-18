package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/router/example"
	"github.com/flipped-aurora/gin-vue-admin/server/router/media"
	"github.com/flipped-aurora/gin-vue-admin/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	CareClient careclient.RouterGroup
	System     system.RouterGroup
	Example    example.RouterGroup
	Media      media.RouterGroup
}
