package kor

import (
	"github.com/yonahd/kor/pkg/filters"
)

var filter filters.Framework

func init() {
	filter = filters.NewNormalFramework(filters.NewDefaultRegistry())
}
