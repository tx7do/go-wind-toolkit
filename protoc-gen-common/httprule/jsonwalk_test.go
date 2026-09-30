package httprule_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/httprule"
)

// jsonwalkFixture 造出三类结构:diamond(Twice.a 与 Twice.b 指向同一个 Mid,
// Mid.leaf 又指向同一个 Leaf)、自引用(Node.child)、以及消息型 repeated 字段。
// 这三类各自钉住一条契约,前两类在按全局 memo 实现的 walk 里都会给出错误答案:
// diamond 的第二支整支消失,叶子少一截。
func jsonwalkFixture(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()

	str := func(s string) *string { return proto.String(s) }
	i32 := func(n int32) *int32 { return proto.Int32(n) }
	opt := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	rep := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	msg := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	str32 := descriptorpb.FieldDescriptorProto_TYPE_STRING

	field := func(name string, num int32, typ descriptorpb.FieldDescriptorProto_Type,
		label descriptorpb.FieldDescriptorProto_Label, typeName string) *descriptorpb.FieldDescriptorProto {
		f := &descriptorpb.FieldDescriptorProto{
			Name: str(name), Number: i32(num), Label: label.Enum(), Type: typ.Enum(),
		}
		if typeName != "" {
			f.TypeName = str(typeName)
		}
		return f
	}
	message := func(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
		return &descriptorpb.DescriptorProto{Name: str(name), Field: fields}
	}

	fdp := &descriptorpb.FileDescriptorProto{
		Name:    str("jsonwalktest.proto"),
		Package: str("jsonwalktest"),
		Syntax:  str("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			message("Twice",
				field("a", 1, msg, opt, ".jsonwalktest.Mid"),
				field("b", 2, msg, opt, ".jsonwalktest.Mid"),
				field("tag", 3, str32, opt, ""),
			),
			message("Mid",
				field("leaf", 1, msg, opt, ".jsonwalktest.Leaf"),
				// repeated 消息按叶子处理,不递归进去。
				field("many", 2, msg, rep, ".jsonwalktest.Leaf"),
			),
			message("Leaf", field("v", 1, str32, opt, "")),
			message("Node",
				field("child", 1, msg, opt, ".jsonwalktest.Node"),
				field("name", 2, str32, opt, ""),
			),
		},
	}

	fd, err := protodesc.NewFile(fdp, nil)
	if err != nil {
		t.Fatalf("protodesc.NewFile: %v", err)
	}
	return fd
}

func TestWalkJSONLeafFields_RepeatedTypes(t *testing.T) {
	fd := jsonwalkFixture(t)

	var got []string
	httprule.WalkJSONLeafFields(fd.Messages().ByName("Twice"), noWellKnown, func(path httprule.FieldPath, field protoreflect.FieldDescriptor) {
		got = append(got, path.String()+"("+field.JSONName()+")")
	})

	// 两条支路都要完整产出。曾经的实现把已走过的消息类型记在全局 seen 里,
	// b 那一支在 enter(Mid) 处就被判成环,整个子树静默消失。
	want := []string{
		"a.leaf.v(v)",
		"a.many(many)",
		"b.leaf.v(v)",
		"b.many(many)",
		"tag(tag)",
	}
	assert.DeepEqual(t, want, got)
}

func TestWalkJSONLeafFields_SelfReferenceTerminates(t *testing.T) {
	fd := jsonwalkFixture(t)

	var got []string
	httprule.WalkJSONLeafFields(fd.Messages().ByName("Node"), noWellKnown, func(path httprule.FieldPath, field protoreflect.FieldDescriptor) {
		got = append(got, path.String())
	})

	// 环只截断那一条边,其余字段照常产出;walk 必须自己收得住,不能靠全局 memo。
	assert.DeepEqual(t, []string{"name"}, got)
}

func TestHasQueryParams_SecondBranchCounts(t *testing.T) {
	fd := jsonwalkFixture(t)

	// a.leaf.v 与 tag 都被路径变量绑定掉了,Mid.many 是消息型集合字段,
	// 于是唯一还能作查询参数的叶子只剩第二支上的 b.leaf.v。
	rule := httprule.Rule{Method: "GET", Template: mustParse(t, "/v1/twice/{a.leaf.v}/{tag}")}
	assert.Assert(t, httprule.HasQueryParams(fd.Messages().ByName("Twice"), rule, noWellKnown),
		"第二支上的 b.leaf.v 是合法查询参数,不能因为 b 与 a 同类型就不被枚举")
}

func noWellKnown(protoreflect.Descriptor) bool { return false }
