package render

import (
	"testing"

	"github.com/jinzhu/inflection"
	"github.com/stretchr/testify/assert"
)

func TestPlural(t *testing.T) {
	t.Log(inflection.Plural("user"))
	t.Log(inflection.Plural("dict"))
	t.Log(inflection.Plural("admin_login_log"))
	t.Log(inflection.Plural("admin-login-log"))
	t.Log(inflection.Plural("adminLoginLog"))
}

func TestSingular(t *testing.T) {
	t.Log(inflection.Singular("users"))
	t.Log(inflection.Singular("dicts"))
	t.Log(inflection.Singular("admin_login_logs"))
	t.Log(inflection.Singular("admin-login-logs"))
	t.Log(inflection.Singular("adminLoginLogs"))
}

func TestWriteGrpcServiceProto(t *testing.T) {
	data := GrpcProtoTemplateData{
		Module:  "user",
		Name:    "user",
		Comment: "用户",
		Version: "v1",

		Fields: ProtoFieldArray{
			{Name: "id", Type: "int64", Number: 1, Comment: "用户ID"},
			{Name: "last_login_time", Type: "int64", Number: 2, Comment: "最后登录时间"},
		},
	}

	err := WriteGrpcServiceProto("./api/protos", data)
	assert.Nil(t, err)
}

func TestWriteRestServiceProto(t *testing.T) {
	data := RestProtoTemplateData{
		SourceModule: "user",
		TargetModule: "admin",
		Name:         "user",
		Version:      "v1",
		Comment:      "用户",
	}

	err := WriteRestServiceProto("./api/protos", data)
	assert.Nil(t, err)
}
