syntax = "proto3";

package {{.TargetPackage}};

import "gnostic/openapi/v3/annotations.proto";
import "google/protobuf/empty.proto";
import "google/api/annotations.proto";
import "pagination/v1/pagination.proto";

import "{{.SourceProto}}";

// {{.ModelName}}管理服务
service {{pascal .Model}}Service {
  // 查询{{.ModelName}}列表
  rpc List (pagination.PagingRequest) returns ({{.SourcePackage}}.List{{pascal .Model}}Response) {
    option (google.api.http) = {
        get: "{{.Path}}"
    };
  }

  // 查询{{.ModelName}}详情
  rpc Get ({{.SourcePackage}}.Get{{pascal .Model}}Request) returns ({{.SourcePackage}}.{{pascal .Model}}) {
    option (google.api.http) = {
        get: "{{.Path}}/{id}"
    };
  }

  // 创建{{.ModelName}}
  rpc Create ({{.SourcePackage}}.Create{{pascal .Model}}Request) returns ({{.SourcePackage}}.{{pascal .Model}}) {
    option (google.api.http) = {
        post: "{{.Path}}"
        body: "*"
    };
  }

  // 更新{{.ModelName}}
  rpc Update ({{.SourcePackage}}.Update{{pascal .Model}}Request) returns (google.protobuf.Empty) {
    option (google.api.http) = {
        put: "{{.Path}}/{id}"
        body: "*"
    };
  }

  // 删除{{.ModelName}}
  rpc Delete ({{.SourcePackage}}.Delete{{pascal .Model}}Request) returns (google.protobuf.Empty) {
    option (google.api.http) = {
        delete: "{{.Path}}/{id}"
    };
  }

  // 批量创建{{.ModelName}}
  rpc BatchCreate ({{.SourcePackage}}.BatchCreate{{pascal .Model}}Request) returns ({{.SourcePackage}}.BatchCreate{{pascal .Model}}Response) {
    option (google.api.http) = {
        post: "{{.Path}}/batch"
        body: "*"
    };
  }
}
