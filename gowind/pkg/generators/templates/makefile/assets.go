package makefile

import _ "embed"

var TemplateMap = map[string][]byte{
	"app_makefile.tpl": AppMakefileTemplate,
}

//go:embed app_makefile.tpl
var AppMakefileTemplate []byte
