// 独立运行模式下的 proto 加载：用 protocompile 从源码编译 spec 涉及的
// proto 文件（含全部传递依赖），拼装出 CodeGeneratorRequest 交给 protogen
// 构建模型。本生成器不作为 protoc 插件运行，所有输入都经此路径。
package main

import (
	"context"
	"fmt"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// loadRequest 把 protoFiles（相对 importPaths 根）编译为 CodeGeneratorRequest。
// 返回的请求携带 source info（protogen 据此还原 leading comments）。
func loadRequest(ctx context.Context, importPaths, protoFiles []string) (*pluginpb.CodeGeneratorRequest, error) {
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: importPaths,
		}),
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	results, err := compiler.Compile(ctx, protoFiles...)
	if err != nil {
		return nil, fmt.Errorf("compile protos: %w", err)
	}

	// 后序遍历依赖树（传递依赖经 Imports() 逐层下钻），天然满足 protogen
	// 对 ProtoFile 的拓扑序要求（依赖先于引用者）。
	var fdps []*descriptorpb.FileDescriptorProto
	visited := make(map[string]bool)
	var visit func(f protoreflect.FileDescriptor)
	visit = func(f protoreflect.FileDescriptor) {
		if visited[f.Path()] {
			return
		}
		visited[f.Path()] = true
		imports := f.Imports()
		for i := 0; i < imports.Len(); i++ {
			// FileImport 是包装结构，须下钻到内嵌的 FileDescriptor 才能
			// 还原成 linker.Result。
			visit(imports.Get(i).FileDescriptor)
		}
		fdps = append(fdps, fdpOf(f))
	}
	for _, f := range results {
		visit(f)
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate:        protoFiles,
		ProtoFile:             fdps,
		SourceFileDescriptors: fdps,
	}
	return req, nil
}

// fdpOf 取文件的描述符 proto：源码编译产物直接携带（含 source info），
// 标准库 WKT 等以链接描述符形态解析的文件经 protodesc 转换。
func fdpOf(f protoreflect.FileDescriptor) *descriptorpb.FileDescriptorProto {
	if lr, ok := f.(linker.Result); ok {
		return lr.FileDescriptorProto()
	}
	if fi, ok := f.(protoreflect.FileImport); ok {
		return fdpOf(fi.FileDescriptor)
	}
	return protodesc.ToFileDescriptorProto(f)
}
