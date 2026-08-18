package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/service/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/service/example"
	"github.com/flipped-aurora/gin-vue-admin/server/service/media"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	CareClientServiceGroup careclient.ServiceGroup
	SystemServiceGroup     system.ServiceGroup
	ExampleServiceGroup    example.ServiceGroup
	MediaServiceGroup      media.ServiceGroup
}
