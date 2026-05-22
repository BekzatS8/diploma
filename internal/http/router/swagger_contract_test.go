package router

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"buhpro/internal/http/swagger"
)

func TestSwaggerDocumentsRegisteredRoutes(t *testing.T) {
	registered := parseRegisteredRoutes(t)
	documented := documentedOperations(t)

	missing := make([]string, 0)
	for route := range registered {
		if _, ok := documented[route]; !ok {
			missing = append(missing, route)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("routes missing from swagger:\n%s", strings.Join(missing, "\n"))
	}

	allowedDocumentedOnly := map[string]struct{}{
		"get /metrics": {},
	}
	extra := make([]string, 0)
	for operation := range documented {
		if _, ok := registered[operation]; ok {
			continue
		}
		if _, ok := allowedDocumentedOnly[operation]; ok {
			continue
		}
		extra = append(extra, operation)
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		t.Fatalf("swagger operations not found in router:\n%s", strings.Join(extra, "\n"))
	}
}

func parseRegisteredRoutes(t *testing.T) map[string]struct{} {
	t.Helper()

	raw, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}

	groups := map[string]string{"r": ""}
	routes := map[string]struct{}{}
	groupRE := regexp.MustCompile(`^\s*(\w+) := (\w+)\.Group\("([^"]*)"\)`)
	routeRE := regexp.MustCompile(`^\s*(\w+)\.(GET|POST|PATCH|DELETE)\("([^"]*)"`)
	paramRE := regexp.MustCompile(`:([A-Za-z0-9_]+)`)

	for _, line := range strings.Split(string(raw), "\n") {
		if match := groupRE.FindStringSubmatch(line); match != nil {
			groupName, parentName, suffix := match[1], match[2], match[3]
			groups[groupName] = groups[parentName] + suffix
			continue
		}

		match := routeRE.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		groupName, methodName, suffix := match[1], strings.ToLower(match[2]), match[3]
		path := paramRE.ReplaceAllString(groups[groupName]+suffix, `{$1}`)
		routes[methodName+" "+path] = struct{}{}
	}

	if len(routes) == 0 {
		t.Fatal("no routes parsed from router.go")
	}
	return routes
}

func documentedOperations(t *testing.T) map[string]struct{} {
	t.Helper()

	spec := swagger.Spec()
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("swagger paths is %T, want map[string]any", spec["paths"])
	}

	operations := map[string]struct{}{}
	for pathName, pathItemValue := range paths {
		pathItem, ok := pathItemValue.(map[string]any)
		if !ok {
			t.Fatalf("swagger path %s is %T, want map[string]any", pathName, pathItemValue)
		}
		for methodName := range pathItem {
			if !isDocumentedHTTPMethod(methodName) {
				continue
			}
			operations[methodName+" "+pathName] = struct{}{}
		}
	}
	return operations
}

func isDocumentedHTTPMethod(value string) bool {
	switch value {
	case "get", "post", "put", "patch", "delete", "options", "head":
		return true
	default:
		return false
	}
}
