package httprule

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// JSONLeafWalkFunc receives every leaf field of the walked message, as the
// path of proto field names leading to it and the field descriptor itself.
type JSONLeafWalkFunc func(path FieldPath, field protoreflect.FieldDescriptor)

// WalkJSONLeafFields walks message and invokes f on every leaf field: fields
// of non-message kind, and message-kind fields whose type the embedding
// generator treats as a leaf, per isWellKnown. Nested non-leaf messages are
// walked recursively, with the field path extended by each traversal step.
func WalkJSONLeafFields(message protoreflect.MessageDescriptor, isWellKnown func(protoreflect.Descriptor) bool, f JSONLeafWalkFunc) {
	var w jsonWalker
	w.walkMessage(nil, message, isWellKnown, f)
}

type jsonWalker struct {
	// onPath 记录当前这条 DFS 路径上已经进入的消息类型及进入次数。
	onPath map[protoreflect.FullName]int
}

// enter 只拒绝"这条路径上重复",不拒绝"整个 walk 里重复"。同一消息类型挂在不同
// 字段下是常态(diamond),按全局 memo 走第二支会整支不产出叶子,查询参数因此在
// 生成的客户端里凭空少一截;需要截断的只有自引用形成的环。
func (w *jsonWalker) enter(name protoreflect.FullName) bool {
	if w.onPath == nil {
		w.onPath = make(map[protoreflect.FullName]int)
	}
	if w.onPath[name] > 0 {
		return false
	}
	w.onPath[name]++
	return true
}

func (w *jsonWalker) leave(name protoreflect.FullName) {
	w.onPath[name]--
}

func (w *jsonWalker) walkMessage(path FieldPath, message protoreflect.MessageDescriptor, isWellKnown func(protoreflect.Descriptor) bool, f JSONLeafWalkFunc) {
	name := message.FullName()
	if !w.enter(name) {
		return
	}
	defer w.leave(name)

	for i := 0; i < message.Fields().Len(); i++ {
		field := message.Fields().Get(i)
		p := append(FieldPath{}, path...)
		p = append(p, string(field.Name()))
		switch {
		case !field.IsMap() && !field.IsList() && field.Kind() == protoreflect.MessageKind:
			if field.Message() == nil {
				warnf("field %q has message kind but no valid message descriptor; treating as leaf", field.FullName())
				f(p, field)
				continue
			}
			if isWellKnown(field.Message()) {
				f(p, field)
			} else {
				w.walkMessage(p, field.Message(), isWellKnown, f)
			}
		default:
			f(p, field)
		}
	}
}

func warnf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "[httprule] WARN: "+format+"\n", args...)
}
