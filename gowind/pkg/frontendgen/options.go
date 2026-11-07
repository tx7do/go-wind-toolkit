package frontendgen

import "strings"

// Framework 目标前端框架
type Framework string

const (
	FrameworkVueElement Framework = "vue-element"
	FrameworkVueVben    Framework = "vue-vben"
	FrameworkReactAntd  Framework = "react"
)

// ParseFramework 解析框架名（接受 vue-element/vue-vben/react-antd/react）
func ParseFramework(s string) (Framework, bool) {
	switch strings.TrimSpace(s) {
	case "vue-element":
		return FrameworkVueElement, true
	case "vue-vben":
		return FrameworkVueVben, true
	case "react", "react-antd":
		return FrameworkReactAntd, true
	}
	return "", false
}

// 生成的文件类型
const (
	TypeComposable = "composable" // vue hooks 文件
	TypeHooks      = "hooks"      // react hooks 文件
	TypePage       = "page"
	TypeDrawer     = "drawer"
	TypeRouter     = "router"
	TypeLocale     = "locale"
)

// RouterModuleConfig 路由模块分组配置
type RouterModuleConfig struct {
	// ModuleKey 模块标识（如 permission, system）
	ModuleKey string `json:"moduleKey"`
	// ModuleDisplayName 模块中文名
	ModuleDisplayName string `json:"moduleDisplayName"`
	// ModuleIcon 模块图标
	ModuleIcon string `json:"moduleIcon,omitempty"`
	// ModuleOrder 模块排序（0 = 框架默认值）
	ModuleOrder int `json:"moduleOrder,omitempty"`
	// Authority 权限标识列表
	Authority []string `json:"authority,omitempty"`
	// ServiceTags 该模块包含的服务 tag 名列表
	ServiceTags []string `json:"serviceTags"`
}

// Options 生成选项
type Options struct {
	// Spec 已解析的 OpenAPI 规格（必填）
	Spec *Spec
	// Framework 目标框架（必填）
	Framework Framework
	// Tags 要生成的服务 tag 名列表；空 = 全部服务（按文档序）
	Tags []string
	// ServiceName 生成代码的服务名（默认 admin，用于 generated 导入路径）
	ServiceName string
	// ModulePathMap 文件名 -> 模块路径（如 role -> permission/role）
	ModulePathMap map[string]string
	// GenerateTypes 要生成的文件类型；空 = 全部类型
	// vue 框架: composable/page/drawer/router/locale；react: hooks/page/drawer/router/locale
	GenerateTypes []string
	// RouterModules 路由模块分组配置；为空且 AutoRouterModules 为 true 时自动检测
	RouterModules []RouterModuleConfig
	// AutoRouterModules RouterModules 为空时按 basePath 自动检测分组
	AutoRouterModules bool
	// ErrorI18NFiles protoc-gen-wind-errors 产出的错误文案词表
	// (<proto>_errors_i18n.<lang>.json,reason -> {code, message})。
	// 非空时按语言合并,额外产出 locales/<lang>/errors.json——
	// 错误文案后端不出网,前端按 reason 本地化。
	ErrorI18NFiles []string
}

// GeneratedFile 生成的文件
type GeneratedFile struct {
	// Path 文件相对路径（相对前端项目 src 目录）
	Path string `json:"path"`
	// Content 文件内容
	Content string `json:"content"`
	// Description 文件描述
	Description string `json:"description"`
	// ServiceName 所属服务 tag 名
	ServiceName string `json:"serviceName"`
	// Type 文件类型（composable/hooks/page/drawer/router/locale）
	Type string `json:"type"`
}

func (o *Options) serviceName() string {
	if o.ServiceName == "" {
		return "admin"
	}
	return o.ServiceName
}

func (o *Options) hasType(t string) bool {
	for _, gt := range o.GenerateTypes {
		if gt == t {
			return true
		}
	}
	return false
}

// resolveServices 提取并按 Tags 过滤服务
func (o *Options) resolveServices() []*ParsedService {
	services := ExtractServices(o.Spec)
	if len(o.Tags) == 0 {
		return services
	}
	filtered := make([]*ParsedService, 0, len(services))
	for _, svc := range services {
		for _, tag := range o.Tags {
			if svc.TagName == tag {
				filtered = append(filtered, svc)
				break
			}
		}
	}
	return filtered
}

// resolveTypes 归一化文件类型列表（react 用 hooks 表达 composable）
func (o *Options) resolveTypes() []string {
	defaultTypes := []string{TypeComposable, TypePage, TypeDrawer, TypeRouter, TypeLocale}
	if o.Framework == FrameworkReactAntd {
		defaultTypes = []string{TypeHooks, TypePage, TypeDrawer, TypeRouter, TypeLocale}
	}
	if len(o.GenerateTypes) == 0 {
		return defaultTypes
	}
	resolved := make([]string, 0, len(o.GenerateTypes))
	for _, t := range o.GenerateTypes {
		// 兼容把 composable 传给 react / 把 hooks 传给 vue 的写法
		if t == TypeComposable && o.Framework == FrameworkReactAntd {
			t = TypeHooks
		}
		resolved = append(resolved, t)
	}
	return resolved
}

// resolveRouterModules 解析路由模块分组
func (o *Options) resolveRouterModules(services []*ParsedService) []RouterModuleConfig {
	if len(o.RouterModules) > 0 {
		return o.RouterModules
	}
	if o.AutoRouterModules {
		return DetectRouterModules(services)
	}
	return nil
}
