package endpoint

import (
	"path"
	"strconv"
	"strings"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/serkangunes/protoc-gen-kit/generator"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath   = "context"
	osPkgPath        = "os"
	osSignalPkgPath  = "os/signal"
	syscallPkgPath   = "syscall"
	timePkgPath      = "time"
	netPkgPath       = "net"
	netHTTPPkgPath   = "net/http"
	goKitPkgPath     = "github.com/go-kit/kit/endpoint"
	goKitGRPCPkgPath = "github.com/go-kit/kit/transport/grpc"
	goKitLogPkgPath  = "github.com/go-kit/kit/log"
	groupLogPkgPath  = "github.com/oklog/oklog/pkg/group"
	grpcPkgPath      = "google.golang.org/grpc"
	promHTTPPkgPath  = "github.com/prometheus/client_golang/prometheus/promhttp"
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
	g.P(strconv.Quote(path.Join(g.gen.ImportPrefix, promHTTPPkgPath)))
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

// generateMIddleware generates standard middlewares for the go-kit services
func (g *kit) generateMiddleware(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	originalServiceName := service.GetName()
	capitalServiceName := generator.CamelCase(originalServiceName)

	g.P("//////////////////////////////////////////////////////////")
	g.P("// Go-kit middlewares for ", capitalServiceName, " service")
	g.P("//////////////////////////////////////////////////////////")
	g.P()

	g.P("type Middleware func(", capitalServiceName, "Server) ", capitalServiceName, "Server")
	g.P()
	g.P("type loggingMiddleware struct {")
	g.P("logger ", goKitLogPkg, ".Logger")
	g.P("next ", capitalServiceName, "Server")
	g.P("}")
	g.P()
	g.P("// LoggingMiddleware takes a logger as a dependency")
	g.P("// and returns a ", capitalServiceName, "Server Middleware.")
	g.P("func LoggingMiddleware(logger ", goKitLogPkg, ".Logger) Middleware {")
	g.P("return func(next ", capitalServiceName, "Server) ", capitalServiceName, "Server {")
	g.P("return &loggingMiddleware{logger, next}")
	g.P("}")
	g.P("}")
	for _, method := range service.Method {
		g.P("func (l loggingMiddleware) ", method.Name, "(ctx context.Context, request *", g.typeName(method.GetInputType()), ") (response *", g.typeName(method.GetOutputType()), ", err error) {")
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
	g.P("//Endpoints stores all the enpoints of the service")
	g.P("type Endpoints struct {")
	for _, method := range service.Method {
		g.P(method.Name, "Endpoint ", goKitPkg, ".Endpoint")
	}
	g.P("}")
	g.P()

	for _, method := range service.Method {
		g.P("func make", method.Name, "Endpoint(handler ", capitalServiceName, "Server)", goKitPkg, ".Endpoint {")
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
	g.P("func NewEndpoints(handler ", capitalServiceName, "Server, middlewares map[string][]", goKitPkg, ".Middleware) Endpoints {")
	g.P("endpoints := Endpoints{")

	for _, method := range service.Method {
		g.P(method.Name, "Endpoint: make", method.Name, "Endpoint(handler),")
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

	g.P("/////////////////////////////////////////////////////////////")
	g.P("// Go-kit grpc transport for ", capitalServiceName, " service")
	g.P("/////////////////////////////////////////////////////////////")
	g.P()

	g.P("//RequestDecoder empty request decoder just returns the same request")
	g.P("func RequestDecoder(ctx context.Context, r interface{}) (interface{}, error) {")
	g.P("return r, nil")
	g.P("}")
	g.P()

	g.P("//ResponseEncoder empty response encoder just returns the same response")
	g.P("func ResponseEncoder(_ context.Context, r interface{}) (interface{}, error) {")
	g.P("return r, nil")
	g.P("}")
	g.P()

	g.P("type grpcServer struct {")
	for _, method := range service.Method {
		g.P(strings.ToLower(*method.Name), "transport ", goKitGRPCPkg, ".Handler")
	}
	g.P("}")
	g.P()

	g.P("// implement ", capitalServiceName, "Server Interface")
	for _, method := range service.Method {
		g.P("//", method.Name, " implementation")
		g.P("func (s *grpcServer) ", method.Name, "(ctx context.Context, r *", g.typeName(method.GetInputType()), ") (*", g.typeName(method.GetOutputType()), ", error) {")
		g.P("_, response, err := s.", strings.ToLower(*method.Name), "transport.ServeGRPC(ctx, r)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return response.(*", g.typeName(method.GetOutputType()), "), nil")
		g.P("}")
		g.P()
	}

	g.P("//NewGRPCServer create new grpc server")
	g.P("func NewGRPCServer(endpoints Endpoints, options map[string][]", goKitGRPCPkg, ".ServerOption) ", capitalServiceName, "Server {")
	g.P("return &grpcServer{")
	for _, method := range service.Method {
		g.P(strings.ToLower(*method.Name), "transport: ", goKitGRPCPkg, ".NewServer(")
		g.P("endpoints.", method.Name, "Endpoint,")
		g.P("RequestDecoder,")
		g.P("ResponseEncoder,")
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

	g.P("/////////////////////////////////////////////////////////////////////")
	g.P("// Go-kit grpc main helper functions ", capitalServiceName, " service")
	g.P("/////////////////////////////////////////////////////////////////////")
	g.P()

	g.P("func RunServer(logger, errorLogger ", goKitLogPkg, ".Logger, grpcAddr, debugAddr string, handler ", capitalServiceName, "Server) {")
	g.P("endpoints := NewEndpoints(handler, nil)")
	g.P("group := createService(endpoints, logger, errorLogger, grpcAddr)")
	g.P("initMetricsEndpoint(debugAddr, logger, errorLogger, group)")
	g.P("initCancelInterrupt(group)")
	g.P("logger.Log(\"exit\", group.Run())")
	g.P("}")
	g.P()

	g.P("func Client(address string, insecure bool, timeoutInSeconds time.Duration) (", capitalServiceName, "Client, *grpc.ClientConn, error) {")
	g.P("var conn *grpc.ClientConn")
	g.P("var err error")
	g.P("if insecure {")
	g.P("conn, err = grpc.Dial(address, grpc.WithInsecure(), grpc.WithTimeout(timeoutInSeconds*time.Second))")
	g.P("} else {")
	g.P("conn, err = grpc.Dial(address, grpc.WithTimeout(timeoutInSeconds*time.Second))")
	g.P("}")
	g.P()
	g.P("if err != nil {")
	g.P("return nil, nil, err")
	g.P("}")
	g.P("return New", capitalServiceName, "Client(conn), conn, nil")
	g.P("}")
	g.P()

	g.P("func GetServiceMiddlewares(logger ", goKitLogPkg, ".Logger) (middlewares []Middleware) {")
	g.P("middlewares = []Middleware{}")
	g.P("return append(middlewares, LoggingMiddleware(logger))")
	g.P("}")
	g.P()

	g.P("func createService(endpoints Endpoints, logger, errorLogger ", goKitLogPkg, ".Logger, grpcAddr string) (g *group.Group) {")
	g.P("g = &group.Group{}")
	g.P("initGRPCHandler(endpoints, logger, errorLogger, grpcAddr, g)")
	g.P("return g")
	g.P("}")
	g.P()

	g.P("func defaultGRPCOptions(errorLogger ", goKitLogPkg, ".Logger) map[string][]", goKitGRPCPkg, ".ServerOption {")
	g.P("options := map[string][]kitgrpc.ServerOption{")
	for _, method := range service.Method {
		g.P("\"", method.Name, "\":   {kitgrpc.ServerErrorLogger(errorLogger)},")
	}
	g.P("}")
	g.P("return options")
	g.P("}")
	g.P()

	g.P("func initGRPCHandler(endpoints Endpoints, logger, errorLogger ", goKitLogPkg, ".Logger, grpcAddr string, g *group.Group) {")
	g.P("options := defaultGRPCOptions(errorLogger)")
	g.P()
	g.P("grpcServer := NewGRPCServer(endpoints, options)")
	g.P("grpcListener, err := net.Listen(\"tcp\", grpcAddr)")
	g.P("if err != nil {")
	g.P("errorLogger.Log(\"transport\", \"gRPC\", \"during\", \"Listen\", \"err\", err)")
	g.P("}")
	g.P("g.Add(func() error {")
	g.P("logger.Log(\"transport\", \"gRPC\", \"addr\", grpcAddr)")
	g.P("baseServer := grpc.NewServer()")
	g.P("Register", capitalServiceName, "Server(baseServer, grpcServer)")
	g.P("return baseServer.Serve(grpcListener)")
	g.P("}, func(error) {")
	g.P("grpcListener.Close()")
	g.P("})")
	g.P("}")
	g.P()

	g.P("func initMetricsEndpoint(debugAddr string, logger, errorLogger ", goKitLogPkg, ".Logger, g *group.Group) {")
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

	g.P("func initCancelInterrupt(g *group.Group) {")
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
