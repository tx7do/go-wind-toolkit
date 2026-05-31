package generators

import (
	"regexp"
	"strings"

	"github.com/tx7do/go-utils/stringcase"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// SnakeToPascal converts a snake_case string to PascalCase.
func SnakeToPascal(snake string) string {
	return stringcase.ToPascalCase(snake)
}

func SnakeToPascalPlus(snake string) string {
	parts := strings.Split(snake, "_")
	titleCaser := cases.Title(language.Und)
	for i := 0; i < len(parts); i++ {
		part := titleCaser.String(parts[i])
		if strings.ToLower(part) == "id" {
			part = "ID"
		}
		parts[i] = part
	}
	return strings.Join(parts, "")
}

func MakeEntSetNillableFunc(fieldName string) string {
	inputVar := "req.Data." + SnakeToPascal(fieldName)
	return "SetNillable" + SnakeToPascalPlus(fieldName) + "(" + inputVar + ")"
}

func MakeEntSetNillableFuncWithTransfer(fieldName string, transFunc string) string {
	inputVar := "req.Data." + SnakeToPascal(fieldName)
	return "SetNillable" + SnakeToPascalPlus(fieldName) + "(" + transFunc + "(" + inputVar + "))"
}

func MakeEntSetFunc(fieldName string) string {
	inputVar := "req.Data.Get" + SnakeToPascal(fieldName) + "()"
	return "Set" + SnakeToPascalPlus(fieldName) + "(" + inputVar + ")"
}

func RemoveTableCommentSuffix(input string) string {
	re := regexp.MustCompile(`(表|table)$`)
	return re.ReplaceAllString(input, "")
}

// ServerImportPaths returns the import paths for the given server types.
func ServerImportPaths(servers []string) []string {
	var paths []string
	seen := make(map[string]bool)

	for _, server := range servers {
		switch strings.TrimSpace(strings.ToLower(server)) {
		case "grpc":
			if p := "github.com/go-kratos/kratos/v2/transport/grpc"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "rest":
			if p := "github.com/go-kratos/kratos/v2/transport/http"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "activemq":
			if p := "github.com/tx7do/kratos-transport/transport/activemq"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "asynq":
			if p := "github.com/tx7do/kratos-transport/transport/asynq"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "fasthttp":
			if p := "github.com/tx7do/kratos-transport/transport/fasthttp"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "gin":
			if p := "github.com/tx7do/kratos-transport/transport/gin"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "gozero":
			if p := "github.com/tx7do/kratos-transport/transport/gozero"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "graphql":
			if p := "github.com/tx7do/kratos-transport/transport/graphql"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "hertz":
			if p := "github.com/tx7do/kratos-transport/transport/hertz"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "iris":
			if p := "github.com/tx7do/kratos-transport/transport/iris"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "kafka":
			if p := "github.com/tx7do/kratos-transport/transport/kafka"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "machinery":
			if p := "github.com/tx7do/kratos-transport/transport/machinery"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "mqtt":
			if p := "github.com/tx7do/kratos-transport/transport/mqtt"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "nats":
			if p := "github.com/tx7do/kratos-transport/transport/nats"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "nsq":
			if p := "github.com/tx7do/kratos-transport/transport/nsq"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "pulsar":
			if p := "github.com/tx7do/kratos-transport/transport/pulsar"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "rabbitmq":
			if p := "github.com/tx7do/kratos-transport/transport/rabbitmq"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "redis":
			if p := "github.com/tx7do/kratos-transport/transport/redis"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "rocketmq":
			if p := "github.com/tx7do/kratos-transport/transport/rocketmq"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "signalr":
			if p := "github.com/tx7do/kratos-transport/transport/signalr"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "socketio":
			if p := "github.com/tx7do/kratos-transport/transport/socketio"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "sse":
			if p := "github.com/tx7do/kratos-transport/transport/sse"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "tcp":
			if p := "github.com/tx7do/kratos-transport/transport/tcp"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "thrift":
			if p := "github.com/tx7do/kratos-transport/transport/thrift"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "trpc":
			if p := "github.com/tx7do/kratos-transport/transport/trpc"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		case "websocket":
			if p := "github.com/tx7do/kratos-transport/transport/websocket"; !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		}
	}

	return paths
}

// ServerFormalParameters 形参
func ServerFormalParameters(servers []string) []string {
	var params []string
	seen := make(map[string]bool)

	for _, server := range servers {
		switch strings.TrimSpace(strings.ToLower(server)) {
		case "grpc":
			if p := "gs *grpc.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "rest":
			if p := "hs *http.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "activemq":
			if p := "ts *activemq.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "asynq":
			if p := "as *asynq.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "fasthttp":
			if p := "fs *fasthttp.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "gin":
			if p := "is *gin.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "gozero":
			if p := "os *gozero.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "graphql":
			if p := "qs *graphql.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "hertz":
			if p := "zs *hertz.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "iris":
			if p := "rs *iris.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "kafka":
			if p := "ks *kafka.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "machinery":
			if p := "ys *machinery.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "mqtt":
			if p := "ms *mqtt.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "nats":
			if p := "nats *nats.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "nsq":
			if p := "nsqs *nsq.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "pulsar":
			if p := "pulsars *pulsar.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "rabbitmq":
			if p := "rabbitmqs *rabbitmq.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "redis":
			if p := "rediss *redis.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "rocketmq":
			if p := "rocketmqs *rocketmq.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "signalr":
			if p := "signalrs *signalr.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "socketio":
			if p := "socketios *socketio.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "sse":
			if p := "ss *sse.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "tcp":
			if p := "tcps *tcp.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "thrift":
			if p := "thrifts *thrift.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "trpc":
			if p := "trpcs *trpc.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		case "websocket":
			if p := "ws *websocket.Server"; !seen[p] {
				params = append(params, p)
				seen[p] = true
			}
		}
	}

	return params
}

// ServerTransferParameters 实参
func ServerTransferParameters(servers []string) []string {
	var out []string
	seen := make(map[string]bool)

	for _, server := range servers {
		switch strings.TrimSpace(strings.ToLower(server)) {
		case "grpc":
			if v := "gs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "rest":
			if v := "hs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "activemq":
			if v := "ts"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "asynq":
			if v := "as"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "fasthttp":
			if v := "fs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "gin":
			if v := "is"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "gozero":
			if v := "os"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "graphql":
			if v := "qs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "hertz":
			if v := "zs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "iris":
			if v := "rs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "kafka":
			if v := "ks"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "machinery":
			if v := "ys"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "mqtt":
			if v := "ms"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "nats":
			if v := "nats"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "nsq":
			if v := "nsqs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "pulsar":
			if v := "pulsars"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "rabbitmq":
			if v := "rabbitmqs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "redis":
			if v := "rediss"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "rocketmq":
			if v := "rocketmqs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "signalr":
			if v := "signalrs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "socketio":
			if v := "socketios"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "sse":
			if v := "ss"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "tcp":
			if v := "tcps"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "thrift":
			if v := "thrifts"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "trpc":
			if v := "trpcs"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		case "websocket":
			if v := "ws"; !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		}
	}

	return out
}
