package plugin

import (
	"sort"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http/internal/codegen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// generateApiClient generates a unified ApiClient class that aggregates all
// service clients into a single entry point. Users inject their own
// ClientTransport implementation.
//
// Example generated output:
//
//	class ApiClient {
//	  final ClientTransport _transport;
//	  FreightServiceClient? _freightService;
//
//	  ApiClient(this._transport);
//
//	  FreightServiceClient get freightService {
//	    _freightService ??= FreightServiceClient(_transport);
//	    return _freightService!;
//	  }
//	}
func generateApiClient(f *codegen.File, services []protoreflect.ServiceDescriptor) {
	if len(services) == 0 {
		Warn("no services found in package; skipping ApiClient generation")
		return
	}

	// Sort services alphabetically by name
	sortedServices := make([]protoreflect.ServiceDescriptor, len(services))
	copy(sortedServices, services)
	sort.Slice(sortedServices, func(i, j int) bool {
		return lowerFirst(string(sortedServices[i].Name())) < lowerFirst(string(sortedServices[j].Name()))
	})

	// Class declaration
	f.P("class ApiClient {")
	f.P(t(1), "final ClientTransport _transport;")
	f.P()

	// Private lazy fields
	for _, svc := range sortedServices {
		fieldName := lowerFirst(string(svc.Name()))
		clientName := descriptorTypeName(svc) + "Client"
		f.P(t(1), clientName, "? _", fieldName, ";")
	}
	f.P()

	// Constructor
	f.P(t(1), "ApiClient(this._transport);")
	f.P()

	// Getter for each service
	for i, svc := range sortedServices {
		fieldName := lowerFirst(string(svc.Name()))
		clientName := descriptorTypeName(svc) + "Client"
		f.P(t(1), clientName, " get ", fieldName, " {")
		f.P(t(2), "_", fieldName, " ??= ", clientName, "(_transport);")
		f.P(t(2), "return _", fieldName, "!;")
		f.P(t(1), "}")
		if i < len(sortedServices)-1 {
			f.P()
		}
	}
	f.P()
	f.P(t(1), "/// Closes all service clients and releases resources.")
	f.P(t(1), "void dispose() {")
	for _, svc := range sortedServices {
		fieldName := lowerFirst(string(svc.Name()))
		f.P(t(2), "_", fieldName, " = null;")
	}
	f.P(t(1), "}")
	f.P("}")
	f.P()

	// Convenience factory
	f.P("ApiClient createApiClient(ClientTransport transport) {")
	f.P(t(1), "return ApiClient(transport);")
	f.P("}")
	f.P()
}
