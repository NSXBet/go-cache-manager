package cmd

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Generate() error {
	var err error

	protoOpt := protogen.Options{}

	protoOpt.Run(func(gen *protogen.Plugin) error {
		err = g.GenerateFiles(gen)

		return err
	})

	if err != nil {
		return fmt.Errorf("generating cache manager code: %w", err)
	}

	return nil
}

func (g *Generator) GenerateFiles(gen *protogen.Plugin) error {
	for _, f := range gen.Files {
		if f.Generate {
			if !g.hasCacheService(f) {
				continue
			}

			if ferr := g.generateFile(gen, f); ferr != nil {
				return fmt.Errorf("generating file %s: %w", f.Desc.Path(), ferr)
			}
		}
	}

	return nil
}

func (g *Generator) hasCacheService(file *protogen.File) bool {
	for _, service := range file.Services {
		if strings.HasSuffix(service.GoName, "Cache") {
			return true
		}
	}

	return false
}

func (g *Generator) generateFile(gen *protogen.Plugin, file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_cache_manager.pb.go"

	gf := gen.NewGeneratedFile(filename, file.GoImportPath)
	gf.P("// Code generated by protoc-gen-go-cache-manager. DO NOT EDIT.")
	gf.P()
	gf.P("package ", file.GoPackageName)
	gf.P()

	gf.P("import (")
	gf.P("	\"context\"")
	gf.P("	\"fmt\"")
	gf.P()
	gf.P("  \"github.com/NSXBet/go-cache-manager/pkg/gocachemanager\"")
	gf.P(")")

	for _, service := range file.Services {
		if !strings.HasSuffix(service.GoName, "Cache") {
			continue
		}

		if serr := g.generateService(gf, service); serr != nil {
			return serr
		}
	}

	return nil
}

func (g *Generator) managerName(structname string) string {
	if len(structname) == 0 {
		return ""
	}

	return fmt.Sprintf("%sManager", structname)
}

func (g *Generator) privateManagerName(structname string) string {
	s := g.managerName(structname)

	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func (g *Generator) methodName(managerName, methodName string) string {
	return fmt.Sprintf(
		"%s_%s",
		managerName,
		methodName,
	)
}

func (g *Generator) generateService(gf *protogen.GeneratedFile, service *protogen.Service) error {
	managerName := g.managerName(service.GoName)

	comments := ""
	if service.Comments.Leading != "" {
		comments = fmt.Sprintf(
			"// %s for every operation related to this service:\n%s",
			managerName,
			service.Comments.Leading.String(),
		)
	}

	gf.P(comments, "type ", managerName, " struct {")

	for _, method := range service.Methods {
		gf.P(
			"	",
			g.methodName(g.privateManagerName(method.GoName), method.GoName),
			" *gocachemanager.CacheManager[*",
			method.Input.GoIdent.GoName,
			", *",
			method.Output.GoIdent.GoName,
			"]",
		)
	}

	gf.P("}")
	gf.P()

	var constructedManagers []string

	comments = ""
	if service.Comments.Leading != "" {
		comments = fmt.Sprintf(
			"// New%s is the constructor method for this service:\n%s",
			managerName,
			service.Comments.Leading.String(),
		)
	}

	refreshMethods := []string{}
	for _, method := range service.Methods {
		refreshMethods = append(refreshMethods, fmt.Sprintf(
			" 	update%sFn func(context.Context, *%s) (*%s, error)",
			method.GoName,
			method.Input.GoIdent.GoName,
			method.Output.GoIdent.GoName,
		))
	}

	gf.P(
		comments,
		"func New",
		g.managerName(service.GoName),
		"(",
	)

	for _, refreshMethod := range refreshMethods {
		gf.P(refreshMethod, ",")
	}

	gf.P("  options ...gocachemanager.CacheOption,")
	gf.P(
		") (*",
		service.GoName,
		"Manager, error) {",
	)
	for _, method := range service.Methods {
		mgrs, merr := g.generateConstructorManager(gf, method)
		if merr != nil {
			return merr
		}

		constructedManagers = append(constructedManagers, mgrs...)
	}
	gf.P("	return &", g.managerName(service.GoName), " {")

	for _, mgr := range constructedManagers {
		gf.P("		", mgr, ": ", mgr, ",")
	}

	gf.P("	}, nil")
	gf.P("}")
	gf.P()

	for _, method := range service.Methods {
		if merr := g.generateMethod(gf, method); merr != nil {
			return merr
		}
	}

	return nil
}

func (g *Generator) generateConstructorManager(
	gf *protogen.GeneratedFile,
	method *protogen.Method,
) ([]string, error) {
	var mgrs []string

	gf.P(
		g.methodName(g.privateManagerName(method.GoName), method.GoName),
		", err := gocachemanager.NewCacheManager[*",
		method.Input.GoIdent.GoName,
		", *",
		method.Output.GoIdent.GoName,
		"](",
		"\"",
		strings.ToLower(method.GoName),
		"\",",
		"func() *",
		method.Output.GoIdent.GoName,
		" { return &",
		method.Output.GoIdent.GoName,
		"{} },",
		"update", method.GoName, "Fn,",
		"options...",
		")",
	)

	gf.P("if err != nil {")
	gf.P(
		"	return nil, fmt.Errorf(\"creating cache manager %s: %w\", \"",
		method.GoName,
		"\", err)",
	)
	gf.P("}")
	gf.P()

	mgrs = append(mgrs, g.methodName(g.privateManagerName(method.GoName), method.GoName))

	return mgrs, nil
}

func (g *Generator) generateMethod(gf *protogen.GeneratedFile, method *protogen.Method) error {
	managerName := g.managerName(method.Parent.GoName)
	fieldName := g.methodName(g.privateManagerName(method.GoName), method.GoName)

	// Get cache
	comments := ""
	if method.Comments.Leading != "" {
		comment := strings.TrimSuffix(
			strings.ReplaceAll(
				strings.ReplaceAll(method.Comments.Leading.String(), "// ", ""),
				"//", "",
			),
			" ",
		)
		comments = fmt.Sprintf(
			"// Get%s",
			comment,
		)
	}

	gf.P(
		comments,
		"func (cm *",
		managerName,
		") Get",
		method.GoName,
		"(",
	)
	gf.P("  ctx context.Context,")
	gf.P("	input *", method.Input.GoIdent.GoName, ",")
	gf.P(") (*", method.Output.GoIdent.GoName, ", error) {")
	gf.P("	return cm.", fieldName, ".Get(ctx, input)")
	gf.P("}")
	gf.P()

	// Set cache
	comments = ""
	if method.Comments.Leading != "" {
		comments = fmt.Sprintf(
			"// Eagerly refresh the cache for the method that:\n%s",
			method.Comments.Leading.String(),
		)
	}

	gf.P(
		comments,
		"func (cm *",
		managerName,
		") Refresh",
		method.GoName,
		"(",
	)
	gf.P("  ctx context.Context,")
	gf.P("	input *", method.Input.GoIdent.GoName, ",")
	gf.P(") (*", method.Output.GoIdent.GoName, ", error) {")
	gf.P("	return cm.", fieldName, ".Refresh(ctx, input)")
	gf.P("}")
	gf.P()

	return nil
}
