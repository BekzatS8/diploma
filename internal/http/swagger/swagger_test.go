package swagger

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSpecIsSerializableOpenAPI3Document(t *testing.T) {
	spec := Spec()

	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}

	if got := decoded["openapi"]; got != "3.0.3" {
		t.Fatalf("openapi = %v, want 3.0.3", got)
	}
	if len(asMap(t, decoded["paths"], "paths")) == 0 {
		t.Fatal("paths must not be empty")
	}
	if len(specSchemas(t, spec)) == 0 {
		t.Fatal("components.schemas must not be empty")
	}
}

func TestSchemaReferencesResolve(t *testing.T) {
	spec := Spec()
	schemas := specSchemas(t, spec)
	refs := map[string]struct{}{}
	collectSchemaRefs(spec, refs)

	for refName := range refs {
		if _, ok := schemas[refName]; !ok {
			t.Errorf("unresolved schema ref %q", refName)
		}
	}
}

func TestOperationsHaveCompleteMetadata(t *testing.T) {
	paths := asMap(t, Spec()["paths"], "paths")
	operationIDs := map[string]string{}

	for pathName, item := range paths {
		for methodName, operation := range asMap(t, item, "path item "+pathName) {
			if !isHTTPMethod(methodName) {
				continue
			}

			op := asMap(t, operation, methodName+" "+pathName)
			requireString(t, op, "summary", methodName+" "+pathName)
			requireString(t, op, "description", methodName+" "+pathName)
			operationID := requireString(t, op, "operationId", methodName+" "+pathName)
			if previous, exists := operationIDs[operationID]; exists {
				t.Errorf("duplicate operationId %q for %s %s and %s", operationID, methodName, pathName, previous)
			}
			operationIDs[operationID] = methodName + " " + pathName

			tags := asSlice(t, op["tags"], "tags for "+methodName+" "+pathName)
			if len(tags) == 0 {
				t.Errorf("%s %s has no tags", methodName, pathName)
			}

			responses := asMap(t, op["responses"], "responses for "+methodName+" "+pathName)
			if !hasSuccessResponse(responses) {
				t.Errorf("%s %s has no 2xx response", methodName, pathName)
			}
			for _, status := range []string{"400", "401", "403", "404", "409", "500"} {
				if _, ok := responses[status]; !ok {
					t.Errorf("%s %s is missing default %s response", methodName, pathName, status)
				}
			}
		}
	}
}

func TestDocumentedResponseShapesMatchHandlers(t *testing.T) {
	spec := Spec()
	schemas := specSchemas(t, spec)

	authProps := schemaProperties(t, schemas, "AuthResponse")
	for _, field := range []string{"user_id", "email", "role", "verification_status", "access_token", "refresh_token"} {
		if _, ok := authProps[field]; !ok {
			t.Errorf("AuthResponse is missing property %q", field)
		}
	}
	for _, staleField := range []string{"user", "tokens"} {
		if _, ok := authProps[staleField]; ok {
			t.Errorf("AuthResponse still documents stale nested property %q", staleField)
		}
	}

	meProps := schemaProperties(t, schemas, "MeResponse")
	for _, field := range []string{"id", "email", "role", "verification_status", "profile"} {
		if _, ok := meProps[field]; !ok {
			t.Errorf("MeResponse is missing property %q", field)
		}
	}

	errorBodyProps := schemaProperties(t, schemas, "ErrorBody")
	for _, staleField := range []string{"error_trace", "stack_trace"} {
		if _, ok := errorBodyProps[staleField]; ok {
			t.Errorf("ErrorBody should document stable client errors, found debug-only field %q", staleField)
		}
	}

	for _, tc := range []struct {
		path   string
		method string
		status string
		schema string
	}{
		{"/api/v1/auth/register", "post", "201", "AuthResponse"},
		{"/api/v1/auth/login", "post", "200", "AuthResponse"},
		{"/api/v1/auth/refresh", "post", "200", "TokenPair"},
		{"/api/v1/profile", "patch", "200", "ProfileResponse"},
		{"/api/v1/profile/avatar", "patch", "200", "ProfileResponse"},
		{"/api/v1/profile/avatar", "delete", "200", "ProfileResponse"},
	} {
		if got := responseSchemaRef(t, spec, tc.path, tc.method, tc.status); got != tc.schema {
			t.Errorf("%s %s response %s = %s, want %s", tc.method, tc.path, tc.status, got, tc.schema)
		}
	}
}

func collectSchemaRefs(node any, refs map[string]struct{}) {
	switch value := node.(type) {
	case map[string]any:
		if ref, ok := value["$ref"].(string); ok && strings.HasPrefix(ref, "#/components/schemas/") {
			refs[strings.TrimPrefix(ref, "#/components/schemas/")] = struct{}{}
		}
		for _, child := range value {
			collectSchemaRefs(child, refs)
		}
	case []any:
		for _, child := range value {
			collectSchemaRefs(child, refs)
		}
	}
}

func specSchemas(t *testing.T, spec map[string]any) map[string]any {
	t.Helper()
	components := asMap(t, spec["components"], "components")
	return asMap(t, components["schemas"], "components.schemas")
}

func schemaProperties(t *testing.T, schemas map[string]any, name string) map[string]any {
	t.Helper()
	schema, ok := schemas[name]
	if !ok {
		t.Fatalf("schema %q not found", name)
	}
	return asMap(t, asMap(t, schema, name)["properties"], name+".properties")
}

func responseSchemaRef(t *testing.T, spec map[string]any, pathName, methodName, status string) string {
	t.Helper()
	paths := asMap(t, spec["paths"], "paths")
	pathItem := asMap(t, paths[pathName], pathName)
	operation := asMap(t, pathItem[methodName], methodName+" "+pathName)
	responses := asMap(t, operation["responses"], "responses for "+methodName+" "+pathName)
	response := asMap(t, responses[status], "response "+status+" for "+methodName+" "+pathName)
	content := asMap(t, response["content"], "content for "+methodName+" "+pathName)
	jsonContent := asMap(t, content["application/json"], "application/json for "+methodName+" "+pathName)
	schema := asMap(t, jsonContent["schema"], "schema for "+methodName+" "+pathName)
	ref := requireString(t, schema, "$ref", "schema ref for "+methodName+" "+pathName)
	return strings.TrimPrefix(ref, "#/components/schemas/")
}

func asMap(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s is %T, want map[string]any", name, value)
	}
	return out
}

func asSlice(t *testing.T, value any, name string) []any {
	t.Helper()
	out, ok := value.([]any)
	if !ok {
		t.Fatalf("%s is %T, want []any", name, value)
	}
	return out
}

func requireString(t *testing.T, values map[string]any, key, context string) string {
	t.Helper()
	value, ok := values[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		t.Fatalf("%s missing non-empty string %q", context, key)
	}
	return value
}

func hasSuccessResponse(responses map[string]any) bool {
	for status := range responses {
		if strings.HasPrefix(status, "2") {
			return true
		}
	}
	return false
}

func isHTTPMethod(value string) bool {
	switch value {
	case "get", "post", "put", "patch", "delete", "options", "head":
		return true
	default:
		return false
	}
}
