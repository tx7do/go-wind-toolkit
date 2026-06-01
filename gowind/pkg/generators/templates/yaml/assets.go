package yaml

import _ "embed"

var TemplateMap = map[string][]byte{
	"client_yaml.tpl": ClientYamlTemplate,
	"server_yaml.tpl": ServerYamlTemplate,
	"data_yaml.tpl":   DataYamlTemplate,
	"logger_yaml.tpl": LoggerYamlTemplate,
}

//go:embed client_yaml.tpl
var ClientYamlTemplate []byte

//go:embed server_yaml.tpl
var ServerYamlTemplate []byte

//go:embed data_yaml.tpl
var DataYamlTemplate []byte

//go:embed logger_yaml.tpl
var LoggerYamlTemplate []byte
