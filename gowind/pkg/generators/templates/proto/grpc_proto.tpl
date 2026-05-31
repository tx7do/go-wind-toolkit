syntax = "proto3";

package {{.Package}};

import "gnostic/openapi/v3/annotations.proto";

import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/field_mask.proto";

import "pagination/v1/pagination.proto";

// {{.ModelName}}服务
service {{pascal .Model}}Service {
  // 查询{{.ModelName}}列表
  rpc List (pagination.PagingRequest) returns (List{{pascal .Model}}Response) {}

  // 统计{{.ModelName}}数量
  rpc Count (pagination.PagingRequest) returns (Count{{pascal .Model}}Response) {}

  // 查询{{.ModelName}}详情
  rpc Get (Get{{pascal .Model}}Request) returns ({{pascal .Model}}) {}

  // 创建{{.ModelName}}
  rpc Create (Create{{pascal .Model}}Request) returns ({{pascal .Model}}) {}

  // 更新{{.ModelName}}
  rpc Update (Update{{pascal .Model}}Request) returns (google.protobuf.Empty) {}

  // 删除{{.ModelName}}
  rpc Delete (Delete{{pascal .Model}}Request) returns (google.protobuf.Empty) {}

  // 批量创建{{.ModelName}}
  rpc BatchCreate (BatchCreate{{pascal .Model}}Request) returns (BatchCreate{{pascal .Model}}Response) {}
}

// {{.ModelName}}
message {{pascal .Model}} {
{{range .Fields}}  optional {{.Type}} {{snake .Name}} = {{.Number}} [
    json_name = "{{camel .Name}}",
    (gnostic.openapi.v3.property) = {description: "{{.Comment}}"}
  ]; // {{.Comment}}

{{end -}}
}

message List{{pascal .Model}}Response {
  repeated {{pascal .Model}} items = 1;
  uint64 total = 2;
}

message Get{{pascal .Model}}Request {
  oneof query_by {
    uint32 id = 1 [
      (gnostic.openapi.v3.property) = {description: "ID", read_only: true},
      json_name = "id"
    ]; // ID
  }

  optional google.protobuf.FieldMask view_mask = 100 [
    json_name = "viewMask",
    (gnostic.openapi.v3.property) = {
      description: "视图字段过滤器，用于控制返回的字段"
    }
  ]; // 视图字段过滤器，用于控制返回的字段
}

message Create{{pascal .Model}}Request {
  {{pascal .Model}} data = 1;
}

message Update{{pascal .Model}}Request {
  uint32 id = 1;

  {{pascal .Model}} data = 2;

  google.protobuf.FieldMask update_mask = 100 [
    (gnostic.openapi.v3.property) = {
      description: "要更新的字段列表",
      example: {yaml : "id,realname,username"}
    },
    json_name = "updateMask"
  ]; // 要更新的字段列表

  optional bool allow_missing = 101 [
    (gnostic.openapi.v3.property) = {description: "如果设置为true的时候，资源不存在则会新增(插入)，并且在这种情况下`updateMask`字段将会被忽略。"},
    json_name = "allowMissing"
  ]; // 如果设置为true的时候，资源不存在则会新增(插入)，并且在这种情况下`updateMask`字段将会被忽略。
}

message Delete{{pascal .Model}}Request {
  oneof delete_by {
    uint32 id = 1 [
      (gnostic.openapi.v3.property) = {description: "ID", read_only: true},
      json_name = "id"
    ]; // ID
  }
}

message BatchCreate{{pascal .Model}}Request {
  repeated {{pascal .Model}} data = 1;
}
message BatchCreate{{pascal .Model}}Response {
  repeated {{pascal .Model}} data = 1;
}

message Count{{pascal .Model}}Response {
  uint64 count = 1;
}
