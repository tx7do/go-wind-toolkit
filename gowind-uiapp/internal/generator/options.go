package generator

type Option struct {
	ID           uint32 `json:"id"`
	TableName    string `json:"tableName"`
	Service      string `json:"service"`
	Exclude      bool   `json:"exclude"`
	ProtoPackage string `json:"protoPackage"`
}

type GeneratorOptions []*Option
type GeneratorOptionMap map[string]*Option
