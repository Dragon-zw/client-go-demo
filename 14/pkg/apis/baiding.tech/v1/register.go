// +groupName=baiding.tech

package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// register.go 会将 types.go 注册 APIServer
var (
	Scheme       = runtime.NewScheme()
	GroupVersion = schema.GroupVersion{Group: "baiding.tech", Version: "v1"}
	Codecs       = serializer.NewCodecFactory(Scheme)
)
