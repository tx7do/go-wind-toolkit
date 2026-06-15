package plugin

import (
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http/internal/codegen"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type commentGenerator struct {
	descriptor protoreflect.Descriptor
}

// tsToDartTerms maps TypeScript/JavaScript ecosystem terms to their Dart
// equivalents so that proto comments referencing TS concepts are adapted for
// generated Dart code.
var tsToDartTerms = []struct{ from, to string }{
	{"fetch", "package:http"},
	{"XMLHttpRequest", "package:http"},
	{"TypeScript", "Dart"},
	{"typescript", "dart"},
	{"node.js", "Dart"},
	{"Node.js", "Dart"},
	{"npm", "pub"},
	{"require(", "import "},
	{"interface ", "abstract class "},
	{"Promise<", "Future<"},
	{"Observable<", "Stream<"},
}

// adaptComment replaces TypeScript-specific terms in a comment line with their
// Dart equivalents.
func adaptComment(line string) string {
	for _, r := range tsToDartTerms {
		line = strings.ReplaceAll(line, r.from, r.to)
	}
	return line
}

func (c commentGenerator) generateLeading(f *codegen.File, indent int) {
	loc := c.descriptor.ParentFile().SourceLocations().ByDescriptor(c.descriptor)
	lines := strings.Split(loc.LeadingComments, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		f.P(t(indent), "/// ", adaptComment(strings.TrimSpace(line)))
	}
	if field, ok := c.descriptor.(protoreflect.FieldDescriptor); ok {
		if behaviorComment := fieldBehaviorComment(field); len(behaviorComment) > 0 {
			f.P(t(indent), "///")
			f.P(t(indent), "/// ", behaviorComment)
		}
	}
}

func fieldBehaviorComment(field protoreflect.FieldDescriptor) string {
	behaviors := getFieldBehaviors(field)
	if len(behaviors) == 0 {
		return ""
	}

	behaviorStrings := make([]string, 0, len(behaviors))
	for _, b := range behaviors {
		behaviorStrings = append(behaviorStrings, b.String())
	}
	return "Behaviors: " + strings.Join(behaviorStrings, ", ")
}

func getFieldBehaviors(field protoreflect.FieldDescriptor) []annotations.FieldBehavior {
	if behaviors, ok := proto.GetExtension(
		field.Options(), annotations.E_FieldBehavior,
	).([]annotations.FieldBehavior); ok {
		return behaviors
	}
	return nil
}
