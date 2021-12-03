package typescript

import (
	"bytes"
	"fmt"
	"io/fs"
	"text/template"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/runtime"
	"github.com/airplanedev/cli/pkg/runtime/javascript"
)

// Init register the runtime.
func init() {
	runtime.Register(".ts", Runtime{})
}

// Code template.
var code = template.Must(template.New("ts").Parse(`{{with .Comment -}}
{{.}}

{{end -}}
type Params = {
  {{- range .Params }}
  {{ .Name }}: {{ .Type }}
  {{- end }}
}

// Put the main logic of the task in this function.
export default async function(params: Params) {
  console.log('parameters:', params);

  // You can return data to show outputs to users.
  // Outputs documentation: https://docs.airplane.dev/tasks/outputs
  return [
    {element: 'hydrogen', weight: 1.008},
    {element: 'helium', weight: 4.0026},
  ];
}
`))

// Data represents the data template.
type data struct {
	Comment string
	Params  []param
}

// Param represents the parameter.
type param struct {
	Name string
	Type string
}

// Runtime implementaton.
type Runtime struct {
	javascript.Runtime
}

// Generate implementation.
func (r Runtime) Generate(t *api.Task) ([]byte, fs.FileMode, error) {
	d := data{}
	if t != nil {
		d.Comment = runtime.Comment(r, *t)
		for _, p := range t.Parameters {
			d.Params = append(d.Params, param{
				Name: p.Slug,
				Type: typeof(p.Type),
			})
		}
	}

	var buf bytes.Buffer
	if err := code.Execute(&buf, d); err != nil {
		return nil, 0, fmt.Errorf("typescript: template execute - %w", err)
	}

	return buf.Bytes(), 0644, nil
}

// Typeof translates the given type to typescript.
func typeof(t api.Type) string {
	switch t {
	case api.TypeInteger, api.TypeFloat:
		return "number"
	case api.TypeDate, api.TypeDatetime:
		return "string"
	case api.TypeBoolean:
		return "boolean"
	case api.TypeString:
		return "string"
	case api.TypeUpload:
		return "string"
	default:
		return "unknown"
	}
}
