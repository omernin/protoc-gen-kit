package endpoint

import (
	"path"
	"strconv"
	"strings"
	"unicode"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/omernin/protoc-gen-kit/generator"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath            = "context"
	osPkgPath                 = "os"
	osSignalPkgPath           = "os/signal"
	syscallPkgPath            = "syscall"
	timePkgPath               = "time"
	netPkgPath                = "net"
	netHTTPPkgPath            = "net/http"
	goKitPkgPath              = "github.com/go-kit/kit/endpoint"
	goKitGRPCPkgPath          = "github.com/go-kit/kit/transport/grpc"
	goKitLogPkgPath           = "github.com/go-kit/kit/log"
	groupLogPkgPath           = "github.com/oklog/oklog/pkg/group"
	grpcPkgPath               = "google.golang.org/grpc"
	grpcInsecurePkgPath       = "google.golang.org/grpc/credentials/insecure"
	promHTTPPkgPath           = "github.com/prometheus/client_golang/prometheus/promhttp"
	grpcGatewayRuntimePkgPath = "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func init() {
	generator.RegisterPlugin(new(kit))
}

// kit is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for go-kit support.
type kit struct {
	gen *generator.Generator
}

// Name returns the name of this plugin, "kit".
func (g *kit) Name() string {
	return "kit"
}

// The names for packages imported in the generated code.
// They may vary from the final path component of the import path
// if the name is used by other packages.
var (
	contextPkg   string
	goKitPkg     string
	goKitGRPCPkg string
	goKitLogPkg  string
)

// Init initializes the plugin.
func (g *kit) Init(gen *generator.Generator) {
	g.gen = gen
	contextPkg = generator.RegisterUniquePackageName("context", nil)
	goKitPkg = generator.RegisterUniquePackageName("kitendpoint", nil)
	goKitGRPCPkg = generator.RegisterUniquePackageName("kitgrpc", nil)
	goKitLogPkg = generator.RegisterUniquePackageName("kitlog", nil)
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *kit) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *kit) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *kit) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *kit) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P("var _ ", contextPkg, ".Context")
	g.P()

	for i, service := range file.FileDescriptorProto.Service {
		g.generateMiddleware(file, service, i)
		g.P()
		g.generateEndpoints(file, service, i)
		g.P()
		g.generateGRPCServer(file, service, i)
		g.P()
		g.generateMainHelperFunctions(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *kit) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.P("import (")
	g.P(contextPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, contextPkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, timePkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, netPkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, netHTTPPkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, osPkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, osSignalPkgPath)))
	g.P(goKitPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, goKitPkgPath)))
	g.P(goKitGRPCPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, goKitGRPCPkgPath)))
	g.P(goKitLogPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, goKitLogPkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, groupLogPkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, grpcPkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, grpcInsecurePkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, promHTTPPkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, grpcGatewayRuntimePkgPath)))
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, syscallPkgPath)))
	g.P(")")
	g.P()
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
	// TODO: do we need any in go-micro?
}

func unexport(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func lowerCaseName(s string) string {
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// generateMIddleware generates standard middlewares for the go-kit services
func (g *kit) generateMiddleware(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	originalServiceName := service.GetName()
	capitalServiceName := generator.CamelCase(originalServiceName)
	lowerCaseServiceName := lowerCaseName(capitalServiceName)

	g.P("//////////////////////////////////////////////////////////")
	g.P("// Go-kit middlewares for ", capitalServiceName, " service")
	g.P("//////////////////////////////////////////////////////////")
	g.P()

	g.P("type ", capitalServiceName, "Middleware func(", capitalServiceName, "Server) ", capitalServiceName, "Server")
	g.P()
	g.P("type ", lowerCaseServiceName, "LoggingMiddleware struct {")
	g.P("logger ", goKitLogPkg, ".Logger")
	g.P("next ", capitalServiceName, "Server")
	g.P("Unimplemented", capitalServiceName, "Server")
	g.P("}")
	g.P()
	g.P("// ", originalServiceName, "LoggingMiddleware takes a logger as a dependency")
	g.P("// and returns a ", capitalServiceName, "Server Middleware.")
	g.P("func ", capitalServiceName, "LoggingMiddleware(logger ", goKitLogPkg, ".Logger)", capitalServiceName, "Middleware {")
	g.P("return func(next ", capitalServiceName, "Server) ", capitalServiceName, "Server {")
	g.P("return &", lowerCaseServiceName, "LoggingMiddleware{logger:logger, next:next}")
	g.P("}")
	g.P("}")
	for _, method := range service.Method {
		g.P("func (l ", lowerCaseServiceName, "LoggingMiddleware) ", method.Name, "(ctx context.Context, request *", g.typeName(method.GetInputType()), ") (response *", g.typeName(method.GetOutputType()), ", err error) {")
		g.P("defer func(begin time.Time) {")
		g.P("l.logger.Log(")
		g.P("\"method\", \"", method.Name, "\",")
		g.P("\"request\", request,")
		g.P("\"response\", response,")
		g.P("\"error\", err,")
		g.P("\"took\", time.Since(begin))")
		g.P("}(time.Now())")
		g.P("return l.next.", method.Name, "(ctx, request)")
		g.P("}")
	}

	g.P()
	g.P("type ", lowerCaseServiceName, "RecoveryMiddleware struct {")
	g.P("logger ", goKitLogPkg, ".Logger")
	g.P("next ", capitalServiceName, "Server")
	g.P("Unimplemented", capitalServiceName, "Server")
	g.P("}")
	g.P()
	g.P("// ", originalServiceName, "RecoveryMiddleware takes a logger as a dependency")
	g.P("// and returns a ", capitalServiceName, "Server Middleware.")
	g.P("func ", capitalServiceName, "RecoveryMiddleware(logger ", goKitLogPkg, ".Logger)", capitalServiceName, "Middleware {")
	g.P("return func(next ", capitalServiceName, "Server) ", capitalServiceName, "Server {")
	g.P("return &", lowerCaseServiceName, "RecoveryMiddleware{logger:logger, next:next}")
	g.P("}")
	g.P("}")
	for _, method := range service.Method {
		g.P("func (l ", lowerCaseServiceName, "RecoveryMiddleware) ", method.Name, "(ctx context.Context, request *", g.typeName(method.GetInputType()), ") (response *", g.typeName(method.GetOutputType()), ", err error) {")
		g.P("defer func() {")
		g.P("if r := recover(); r != nil {")
		g.P("l.logger.Log(")
		g.P("\"method\", \"", method.Name, "\",")
		g.P("\"message\", r)")
		g.P("err = fmt.Errorf(\"%v\", r)")
		g.P("}")
		g.P("}()")
		g.P("return l.next.", method.Name, "(ctx, request)")
		g.P("}")
	}
	/*
	 */
}

// generateEndpoints generates endpoints for the go-kit services
func (g *kit) generateEndpoints(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	originalServiceName := service.GetName()
	capitalServiceName := generator.CamelCase(originalServiceName)

	g.P("////////////////////////////////////////////////////////")
	g.P("// Go-kit endpoints for ", capitalServiceName, " service")
	g.P("////////////////////////////////////////////////////////")
	g.P()

	// Client structure.
	g.P("//", capitalServiceName, "Endpoints stores all the enpoints of the service")
	g.P("type ", capitalServiceName, "Endpoints struct {")
	for _, method := range service.Method {
		g.P(method.Name, "Endpoint ", goKitPkg, ".Endpoint")
	}
	g.P("}")
	g.P()

	for _, method := range service.Method {
		g.P("func make", capitalServiceName, method.Name, "Endpoint(handler ", capitalServiceName, "Server)", goKitPkg, ".Endpoint {")
		g.P("return func(ctx ", contextPkg, ".Context, r interface{}) (interface{}, error) {")
		g.P("request := r.(*", g.typeName(method.GetInputType()), ")")
		g.P("response, err := handler.", method.Name, "(ctx, request)")
		g.P("return response, err")
		g.P("}")
		g.P("}")
		g.P()
	}

	g.P("// New returns a Endpoints struct that wraps the provided service, and wires in all of the")
	g.P("// expected endpoint middlewares")
	g.P("func New", capitalServiceName, "Endpoints(handler ", capitalServiceName, "Server, middlewares map[string][]", goKitPkg, ".Middleware) ", capitalServiceName, "Endpoints {")
	g.P("endpoints := ", capitalServiceName, "Endpoints{")

	for _, method := range service.Method {
		g.P(method.Name, "Endpoint: make", capitalServiceName, method.Name, "Endpoint(handler),")
	}

	g.P("}")
	g.P()
	for _, method := range service.Method {
		g.P("for _, middleware := range middlewares[\"", method.Name, "\"] {")
		g.P("endpoints.", method.Name, "Endpoint = middleware(endpoints.", method.Name, "Endpoint)")
		g.P("}")
		g.P()
	}
	g.P("return endpoints")
	g.P("}")
}

// generateGRPCServer generates grpc integration for go-kit using go-kit grpc transport. It reuses the protobuf request/respose as the domain request response
func (g *kit) generateGRPCServer(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	originalServiceName := service.GetName()
	capitalServiceName := generator.CamelCase(originalServiceName)
	lowerCaseServiceName := lowerCaseName(capitalServiceName)

	g.P("/////////////////////////////////////////////////////////////")
	g.P("// Go-kit grpc transport for ", capitalServiceName, " service")
	g.P("/////////////////////////////////////////////////////////////")
	g.P()

	g.P("//", capitalServiceName, "RequestDecoder empty request decoder just returns the same request")
	g.P("func ", capitalServiceName, "RequestDecoder(ctx context.Context, r interface{}) (interface{}, error) {")
	g.P("return r, nil")
	g.P("}")
	g.P()

	g.P("//", capitalServiceName, "ResponseEncoder empty response encoder just returns the same response")
	g.P("func ", capitalServiceName, "ResponseEncoder(_ context.Context, r interface{}) (interface{}, error) {")
	g.P("return r, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerCaseServiceName, "GrpcServer struct {")
	for _, method := range service.Method {
		g.P(strings.ToLower(*method.Name), "transport ", goKitGRPCPkg, ".Handler")
	}
	g.P("Unimplemented", capitalServiceName, "Server")
	g.P("}")
	g.P()

	g.P("// implement ", capitalServiceName, "Server Interface")
	for _, method := range service.Method {
		g.P("//", method.Name, " implementation")
		g.P("func (s *", lowerCaseServiceName, "GrpcServer) ", method.Name, "(ctx context.Context, r *", g.typeName(method.GetInputType()), ") (*", g.typeName(method.GetOutputType()), ", error) {")
		g.P("_, response, err := s.", strings.ToLower(*method.Name), "transport.ServeGRPC(ctx, r)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return response.(*", g.typeName(method.GetOutputType()), "), nil")
		g.P("}")
		g.P()
	}

	g.P("//NewGRPCServer create new grpc server")
	g.P("func New", capitalServiceName, "GRPCServer(endpoints ", capitalServiceName, "Endpoints, options map[string][]", goKitGRPCPkg, ".ServerOption) ", capitalServiceName, "Server {")
	g.P("return &", lowerCaseServiceName, "GrpcServer{")
	for _, method := range service.Method {
		g.P(strings.ToLower(*method.Name), "transport: ", goKitGRPCPkg, ".NewServer(")
		g.P("endpoints.", method.Name, "Endpoint,")
		g.P(capitalServiceName, "RequestDecoder,")
		g.P(capitalServiceName, "ResponseEncoder,")
		g.P("options[\"", method.Name, "\"]...,")
		g.P("),")
	}
	g.P("}")
	g.P("}")
}

// generateMainHelperFunctions creates the helper methods for main which uses the default values. If you need to customise this you need to set
// everything manually
func (g *kit) generateMainHelperFunctions(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	originalServiceName := service.GetName()
	capitalServiceName := generator.CamelCase(originalServiceName)
	lowerCaseServiceName := lowerCaseName(capitalServiceName)

	g.P("/////////////////////////////////////////////////////////////////////")
	g.P("// Go-kit grpc main helper functions ", capitalServiceName, " service")
	g.P("/////////////////////////////////////////////////////////////////////")
	g.P()

	g.P("func Run", capitalServiceName, "Server(logger, errorLogger ", goKitLogPkg, ".Logger, grpcAddr, httpAddr, debugAddr string, handler ", capitalServiceName, "Server, middlewares map[string][]", goKitPkg, ".Middleware) {")
	g.P("endpoints := New", capitalServiceName, "Endpoints(handler, middlewares)")
	g.P("group := ", lowerCaseServiceName, "CreateService(endpoints, logger, errorLogger, grpcAddr, httpAddr)")
	g.P("init", capitalServiceName, "MetricsEndpoint(debugAddr, logger, errorLogger, group)")
	g.P("init", capitalServiceName, "CancelInterrupt(group)")
	g.P("logger.Log(\"exit\", group.Run())")
	g.P("}")
	g.P()

	g.P(`func Run`, capitalServiceName, `ServerWithDefaults(logger, errorLogger `, goKitLogPkg, `.Logger, grpcAddr, httpAddr, debugAddr string, handler `, capitalServiceName, `Server) {
	middlewares := Get`, capitalServiceName, `ServiceMiddlewares(logger)
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	Run`, capitalServiceName, `Server(logger, errorLogger, grpcAddr, httpAddr, debugAddr, handler, nil)
	}
	`)

	g.P("func Get", capitalServiceName, "Client(address string, isInsecure bool, timeout time.Duration) (", capitalServiceName, "Client, *grpc.ClientConn, error) {")
	g.P("var conn *grpc.ClientConn")
	g.P("var err error")
	g.P("ctx, cancel := context.WithTimeout(context.Background(), timeout * time.Second)")
	g.P("defer cancel()")
	g.P()
	g.P("if isInsecure {")
	g.P("conn, err = grpc.DialContext(ctx, address, grpc.WithTransportCredentials(insecure.NewCredentials()))")
	g.P("} else {")
	g.P("conn, err = grpc.DialContext(ctx, address)")
	g.P("}")
	g.P()
	g.P("if err != nil {")
	g.P("return nil, nil, err")
	g.P("}")
	g.P("return New", capitalServiceName, "Client(conn), conn, nil")
	g.P("}")
	g.P()

	g.P("func Get", capitalServiceName, "ServiceMiddlewares(logger ", goKitLogPkg, ".Logger) (middlewares []", capitalServiceName, "Middleware) {")
	g.P("middlewares = []", capitalServiceName, "Middleware{}")
	g.P("middlewares = append(middlewares, ", capitalServiceName, "LoggingMiddleware(logger))")
	g.P("middlewares = append(middlewares, ", capitalServiceName, "RecoveryMiddleware(logger))")
	g.P("return middlewares")
	g.P("}")
	g.P()

	g.P("func ", lowerCaseServiceName, "CreateService(endpoints ", capitalServiceName, "Endpoints, logger, errorLogger ", goKitLogPkg, ".Logger, grpcAddr, httpAddr string) (g *group.Group) {")
	g.P("g = &group.Group{}")
	g.P()
	g.P("init", capitalServiceName, "GRPCHandler(endpoints, logger, errorLogger, grpcAddr, g)")
	g.P("init", capitalServiceName, "HTTPHandler(endpoints, logger, errorLogger, grpcAddr, httpAddr, g)")
	g.P("return g")
	g.P("}")
	g.P()

	g.P("func default", capitalServiceName, "GRPCOptions(errorLogger ", goKitLogPkg, ".Logger) map[string][]", goKitGRPCPkg, ".ServerOption {")
	g.P("options := map[string][]kitgrpc.ServerOption{")
	for _, method := range service.Method {
		g.P("\"", method.Name, "\":   {kitgrpc.ServerErrorLogger(errorLogger)},")
	}
	g.P("}")
	g.P("return options")
	g.P("}")
	g.P()

	g.P("func init", capitalServiceName, "GRPCHandler(endpoints ", capitalServiceName, "Endpoints, logger, errorLogger ", goKitLogPkg, ".Logger, serviceAddr string, g *group.Group) {")
	g.P("options := default", capitalServiceName, "GRPCOptions(errorLogger)")
	g.P()
	g.P("grpcServer := New", capitalServiceName, "GRPCServer(endpoints, options)")
	g.P("listener, err := net.Listen(\"tcp\", serviceAddr)")
	g.P("if err != nil {")
	g.P("errorLogger.Log(\"transport\", \"gRPC\", \"during\", \"Listen\", \"err\", err)")
	g.P("}")
	g.P()
	g.P("g.Add(func() error {")
	g.P("logger.Log(\"transport\", \"gRPC\", \"addr\", serviceAddr)")
	g.P("baseServer := grpc.NewServer()")
	g.P("Register", capitalServiceName, "Server(baseServer, grpcServer)")
	g.P("return baseServer.Serve(listener)")
	g.P("}, func(error) {")
	g.P("listener.Close()")
	g.P("})")
	g.P("}")
	g.P()

	g.P("func init", capitalServiceName, "HTTPHandler(endpoints ", capitalServiceName, "Endpoints, logger, errorLogger ", goKitLogPkg, ".Logger, grpcAddr, httpAddr string, g *group.Group) {")
	g.P("g.Add(func() error {")
	g.P("logger.Log(\"transport\", \"http\", \"addr\", httpAddr)")
	g.P("ctx, cancel := context.WithCancel(context.Background())")
	g.P("defer cancel()")
	g.P("")
	g.P("mux := runtime.NewServeMux()")
	g.P("opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}")
	g.P("err := Register", capitalServiceName, "HandlerFromEndpoint(ctx, mux, grpcAddr, opts)")
	g.P("if err != nil {")
	g.P("errorLogger.Log(\"transport\", \"http\", \"during\", \"Register", capitalServiceName, "HandlerFromEndpoint\", \"err\", err)")
	g.P("return err")
	g.P("}")
	g.P("return http.ListenAndServe(httpAddr, mux)")
	g.P("}, func(err error) {")
	g.P("errorLogger.Log(\"transport\", \"http\", \"during\", \"ListenAndServe\", \"err\", err)")
	g.P("})")
	g.P("}")
	g.P()

	g.P("func init", capitalServiceName, "MetricsEndpoint(debugAddr string, logger, errorLogger ", goKitLogPkg, ".Logger, g *group.Group) {")
	g.P("http.DefaultServeMux.Handle(\"/metrics\", promhttp.Handler())")
	g.P("debugListener, err := net.Listen(\"tcp\", debugAddr)")
	g.P("if err != nil {")
	g.P("errorLogger.Log(\"transport\", \"debug/HTTP\", \"during\", \"Listen\", \"err\", err)")
	g.P("}")
	g.P("g.Add(func() error {")
	g.P("logger.Log(\"transport\", \"debug/HTTP\", \"addr\", debugAddr)")
	g.P("return http.Serve(debugListener, http.DefaultServeMux)")
	g.P("}, func(error) {")
	g.P("debugListener.Close()")
	g.P("})")
	g.P("}")
	g.P()

	g.P("func init", capitalServiceName, "CancelInterrupt(g *group.Group) {")
	g.P("cancelInterrupt := make(chan struct{})")
	g.P("g.Add(func() error {")
	g.P("c := make(chan os.Signal, 1)")
	g.P("signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)")
	g.P("select {")
	g.P("case sig := <-c:")
	g.P("return fmt.Errorf(\"received signal %s\", sig)")
	g.P("case <-cancelInterrupt:")
	g.P("return nil")
	g.P("}")
	g.P("}, func(error) {")
	g.P("close(cancelInterrupt)")
	g.P("})")
	g.P("}")
}
