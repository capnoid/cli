package definitions

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"path"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/lib/pkg/build"
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

type Definition_0_3 struct {
	Name        string                    `json:"name"`
	Slug        string                    `json:"slug"`
	Description string                    `json:"description,omitempty"`
	Parameters  []ParameterDefinition_0_3 `json:"parameters,omitempty"`

	Deno       *DenoDefinition_0_3       `json:"deno,omitempty"`
	Dockerfile *DockerfileDefinition_0_3 `json:"dockerfile,omitempty"`
	Go         *GoDefinition_0_3         `json:"go,omitempty"`
	Image      *ImageDefinition_0_3      `json:"image,omitempty"`
	Node       *NodeDefinition_0_3       `json:"node,omitempty"`
	Python     *PythonDefinition_0_3     `json:"python,omitempty"`
	Shell      *ShellDefinition_0_3      `json:"shell,omitempty"`

	SQL  *SQLDefinition_0_3  `json:"sql,omitempty"`
	REST *RESTDefinition_0_3 `json:"rest,omitempty"`

	Permissions *PermissionDefinition_0_3 `json:"permissions,omitempty"`
	Constraints *api.RunConstraints       `json:"constraints,omitempty"`
	// TODO: default 3600
	Timeout int `json:"timeout,omitempty"`
}

type ImageDefinition_0_3 struct {
	Image   string      `json:"image"`
	Command []string    `json:"command"`
	Root    string      `json:"root,omitempty"`
	Env     api.TaskEnv `json:"env,omitempty"`
}

type DenoDefinition_0_3 struct {
	Entrypoint string `json:"entrypoint"`
	// TODO: default {{JSON.stringify(params)}}
	Arguments []string    `json:"arguments,omitempty"`
	Root      string      `json:"root,omitempty"`
	Env       api.TaskEnv `json:"env,omitempty"`
}

type DockerfileDefinition_0_3 struct {
	Dockerfile string      `json:"dockerfile"`
	Root       string      `json:"root,omitempty"`
	Env        api.TaskEnv `json:"env,omitempty"`
}

type GoDefinition_0_3 struct {
	Entrypoint string `json:"entrypoint"`
	// TODO: default {{JSON.stringify(params)}}
	Arguments []string    `json:"arguments,omitempty"`
	Root      string      `json:"root,omitempty"`
	Env       api.TaskEnv `json:"env,omitempty"`
}

type NodeDefinition_0_3 struct {
	Entrypoint  string `json:"entrypoint"`
	NodeVersion string `json:"nodeVersion"`
	// TODO: default {{JSON.stringify(params)}}
	Arguments []string    `json:"arguments,omitempty"`
	Root      string      `json:"root,omitempty"`
	Env       api.TaskEnv `json:"env,omitempty"`
}

type PythonDefinition_0_3 struct {
	Entrypoint string `json:"entrypoint"`
	// TODO: default {{JSON.stringify(params)}}
	Arguments []string    `json:"arguments,omitempty"`
	Root      string      `json:"root,omitempty"`
	Env       api.TaskEnv `json:"env,omitempty"`
}

type ShellDefinition_0_3 struct {
	Entrypoint string `json:"entrypoint"`
	// TODO: defaults to PARAM1={{params.param1}} PARAM2{{params.param2}} etc.
	Arguments []string    `json:"arguments,omitempty"`
	Root      string      `json:"root,omitempty"`
	Env       api.TaskEnv `json:"env,omitempty"`
}

type SQLDefinition_0_3 struct {
	Resource   string                 `json:"resource"`
	Entrypoint string                 `json:"entrypoint"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type RESTDefinition_0_3 struct {
	Resource  string                 `json:"resource"`
	Method    string                 `json:"method"`
	Path      string                 `json:"path"`
	URLParams map[string]string      `json:"urlParams,omitempty"`
	Headers   map[string]string      `json:"headers,omitempty"`
	BodyType  string                 `json:"bodyType"`
	Body      string                 `json:"body,omitempty"`
	FormData  map[string]interface{} `json:"formData,omitempty"`
}

type ParameterDefinition_0_3 struct {
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	// TODO: default to true
	Required bool                   `json:"required,omitempty"`
	Options  []OptionDefinition_0_3 `json:"options,omitempty"`
}

type OptionDefinition_0_3 struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

var _ json.Unmarshaler = &OptionDefinition_0_3{}

func (o *OptionDefinition_0_3) UnmarshalJSON(b []byte) error {
	// If it's just a string, dump it in the value field.
	var value string
	if err := json.Unmarshal(b, &value); err == nil {
		o.Value = value
		return nil
	}

	// Otherwise, perform a normal unmarshal operation.
	// Note we need a new type, otherwise we recursively call this
	// method and end up stack overflowing.
	type option OptionDefinition_0_3
	var opt option
	if err := json.Unmarshal(b, &opt); err != nil {
		return err
	}
	*o = OptionDefinition_0_3(opt)

	return nil
}

type PermissionDefinition_0_3 struct {
	Viewers    []string `json:"viewers,omitempty"`
	Requesters []string `json:"requesters,omitempty"`
	Executers  []string `json:"executers,omitempty"`
	Admins     []string `json:"admins,omitempty"`
}

func (d PermissionDefinition_0_3) isEmpty() bool {
	return len(d.Viewers) == 0 && len(d.Requesters) == 0 && len(d.Executers) == 0 && len(d.Admins) == 0
}

//go:embed schema_0_3.json
var schemaStr string

func NewDefinition_0_3(name string, slug string, kind build.TaskKind, entrypoint string) (Definition_0_3, error) {
	def := Definition_0_3{
		Name: name,
		Slug: slug,
	}

	switch kind {
	case build.TaskKindDeno:
		def.Deno = &DenoDefinition_0_3{
			Entrypoint: entrypoint,
		}
	case build.TaskKindDockerfile:
		def.Dockerfile = &DockerfileDefinition_0_3{
			Dockerfile: entrypoint,
		}
	case build.TaskKindGo:
		def.Go = &GoDefinition_0_3{
			Entrypoint: entrypoint,
		}
	case build.TaskKindImage:
		def.Image = &ImageDefinition_0_3{}
	case build.TaskKindNode:
		def.Node = &NodeDefinition_0_3{
			Entrypoint: entrypoint,
		}
	case build.TaskKindPython:
		def.Python = &PythonDefinition_0_3{
			Entrypoint: entrypoint,
		}
	case build.TaskKindShell:
		def.Shell = &ShellDefinition_0_3{
			Entrypoint: entrypoint,
		}
	case build.TaskKindSQL:
		def.SQL = &SQLDefinition_0_3{
			Entrypoint: entrypoint,
		}
	case build.TaskKindREST:
		def.REST = &RESTDefinition_0_3{}
	default:
		return Definition_0_3{}, errors.Errorf("unknown kind: %s", kind)
	}

	return def, nil
}

func (d Definition_0_3) Marshal(format TaskDefFormat) ([]byte, error) {
	buf, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		return nil, err
	}

	switch format {
	case TaskDefFormatYAML:
		buf, err = yaml.JSONToYAML(buf)
		if err != nil {
			return nil, err
		}
	case TaskDefFormatJSON:
		// nothing
	default:
		return nil, errors.Errorf("unknown format: %s", format)
	}

	return buf, nil
}

func (d *Definition_0_3) Unmarshal(format TaskDefFormat, buf []byte) error {
	var err error
	switch format {
	case TaskDefFormatYAML:
		buf, err = yaml.YAMLToJSON(buf)
		if err != nil {
			return err
		}
	case TaskDefFormatJSON:
		// nothing
	default:
		return errors.Errorf("unknown format: %s", format)
	}

	schemaLoader := gojsonschema.NewStringLoader(schemaStr)
	docLoader := gojsonschema.NewBytesLoader(buf)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return errors.Wrap(err, "validating schema")
	}

	if !result.Valid() {
		return errors.WithStack(ErrSchemaValidation{Errors: result.Errors()})
	}

	if err = json.Unmarshal(buf, &d); err != nil {
		return err
	}
	return nil
}

func (d Definition_0_3) Kind() (build.TaskKind, error) {
	if d.Deno != nil {
		return build.TaskKindDeno, nil
	} else if d.Dockerfile != nil {
		return build.TaskKindDockerfile, nil
	} else if d.Go != nil {
		return build.TaskKindGo, nil
	} else if d.Image != nil {
		return build.TaskKindImage, nil
	} else if d.Node != nil {
		return build.TaskKindNode, nil
	} else if d.Python != nil {
		return build.TaskKindPython, nil
	} else if d.Shell != nil {
		return build.TaskKindShell, nil
	} else if d.SQL != nil {
		return build.TaskKindSQL, nil
	} else if d.REST != nil {
		return build.TaskKindREST, nil
	} else {
		return "", errors.New("incomplete task definition")
	}
}

func (d Definition_0_3) UpdateTaskRequest(ctx context.Context, client *api.Client, image *string) (api.UpdateTaskRequest, error) {
	req := api.UpdateTaskRequest{
		Slug:        d.Slug,
		Name:        d.Name,
		Description: d.Description,
		Timeout:     d.Timeout,
	}

	if image != nil {
		req.Image = image
	}

	// Convert parameters.
	req.Parameters = make([]api.Parameter, len(d.Parameters))
	for i, pd := range d.Parameters {
		param := api.Parameter{
			Name:    pd.Name,
			Slug:    pd.Slug,
			Desc:    pd.Description,
			Default: pd.Default,
		}

		switch pd.Type {
		case "shorttext":
			param.Type = "string"
		case "longtext":
			param.Type = "string"
			param.Component = api.ComponentTextarea
		case "sql":
			param.Type = "string"
			param.Component = api.ComponentEditorSQL
		case "boolean", "upload", "integer", "float", "date", "datetime", "configvar":
			param.Type = api.Type(pd.Type)
		default:
			return api.UpdateTaskRequest{}, errors.Errorf("unknown parameter type: %s", pd.Type)
		}

		if !pd.Required {
			param.Constraints.Optional = true
		}

		if len(pd.Options) > 0 {
			param.Constraints.Options = make([]api.ConstraintOption, len(pd.Options))
			for j, od := range pd.Options {
				param.Constraints.Options[j].Label = od.Label
				param.Constraints.Options[j].Value = od.Value
			}
		}

		req.Parameters[i] = param
	}

	if d.Permissions != nil && !d.Permissions.isEmpty() {
		req.RequireExplicitPermissions = true
		// TODO: convert permissions.
	}

	if d.Constraints != nil {
		req.Constraints = *d.Constraints
	}

	resourcesByName := map[string]api.Resource{}
	if d.SQL != nil || d.REST != nil {
		// Remap resources from ref -> name to ref -> id.
		resp, err := client.ListResources(ctx)
		if err != nil {
			return api.UpdateTaskRequest{}, errors.Wrap(err, "fetching resources")
		}
		for _, resource := range resp.Resources {
			resourcesByName[resource.Name] = resource
		}
	}

	// Convert kind-specific things.
	if kind, err := d.Kind(); err != nil {
		return api.UpdateTaskRequest{}, err
	} else {
		req.Kind = kind
	}
	switch req.Kind {
	case build.TaskKindDeno:
		req.KindOptions = build.KindOptions{
			"entrypoint": d.Deno.Entrypoint,
		}
		req.Arguments = d.Deno.Arguments
		req.Env = d.Deno.Env
	case build.TaskKindDockerfile:
		req.KindOptions = build.KindOptions{
			"dockerfile": d.Dockerfile.Dockerfile,
		}
		req.Env = d.Dockerfile.Env
	case build.TaskKindGo:
		req.KindOptions = build.KindOptions{
			"entrypoint": d.Go.Entrypoint,
		}
		req.Arguments = d.Go.Arguments
		req.Env = d.Go.Env
	case build.TaskKindImage:
		req.KindOptions = build.KindOptions{}
		req.Image = &d.Image.Image
		req.Command = d.Image.Command
		req.Env = d.Image.Env
	case build.TaskKindNode:
		req.KindOptions = build.KindOptions{
			"entrypoint":  d.Node.Entrypoint,
			"nodeVersion": d.Node.NodeVersion,
		}
		req.Arguments = d.Node.Arguments
		req.Env = d.Node.Env
	case build.TaskKindPython:
		req.KindOptions = build.KindOptions{
			"entrypoint": d.Python.Entrypoint,
		}
		req.Arguments = d.Python.Arguments
	case build.TaskKindShell:
		req.KindOptions = build.KindOptions{
			"entrypoint": d.Shell.Entrypoint,
		}
		req.Env = d.Shell.Env
		req.Arguments = d.Shell.Arguments
	case build.TaskKindSQL:
		queryBytes, err := os.ReadFile(d.SQL.Entrypoint)
		if err != nil {
			return api.UpdateTaskRequest{}, errors.Wrapf(err, "reading SQL entrypoint %s", d.SQL.Entrypoint)
		}
		req.KindOptions = build.KindOptions{
			"query":     string(queryBytes),
			"queryArgs": d.SQL.Parameters,
		}
		if res, ok := resourcesByName[d.SQL.Resource]; ok {
			req.Resources = map[string]string{
				"db": res.ID,
			}
		} else {
			return api.UpdateTaskRequest{}, errors.Errorf("unknown resource: %s", d.SQL.Resource)
		}
	case build.TaskKindREST:
		req.KindOptions = build.KindOptions{
			"method":    d.REST.Method,
			"path":      d.REST.Path,
			"urlParams": d.REST.URLParams,
			"headers":   d.REST.Headers,
			"bodyType":  d.REST.BodyType,
			"body":      d.REST.Body,
			"formData":  d.REST.FormData,
		}
		if res, ok := resourcesByName[d.REST.Resource]; ok {
			req.Resources = map[string]string{
				"rest": res.ID,
			}
		} else {
			return api.UpdateTaskRequest{}, errors.Errorf("unknown resource: %s", d.REST.Resource)
		}
	default:
		return api.UpdateTaskRequest{}, errors.Errorf("unhandled kind: %s", req.Kind)
	}

	return req, nil
}

func (d Definition_0_3) Root(dir string) (string, error) {
	kind, err := d.Kind()
	if err != nil {
		return "", err
	}

	var root string

	switch kind {
	case build.TaskKindDeno:
		root = d.Deno.Root
	case build.TaskKindDockerfile:
		root = d.Dockerfile.Root
	case build.TaskKindGo:
		root = d.Go.Root
	case build.TaskKindImage:
		root = d.Image.Root
	case build.TaskKindNode:
		root = d.Node.Root
	case build.TaskKindPython:
		root = d.Python.Root
	case build.TaskKindShell:
		root = d.Shell.Root
	case build.TaskKindSQL, build.TaskKindREST:
		return "", nil
	default:
		return "", errors.Errorf("unhandled kind: %s", kind)
	}

	return path.Join(dir, root), nil
}

var ErrNoEntrypoint = errors.New("No entrypoint")

func (d Definition_0_3) Entrypoint() (string, error) {
	kind, err := d.Kind()
	if err != nil {
		return "", err
	}

	switch kind {
	case build.TaskKindDeno:
		return d.Deno.Entrypoint, nil
	case build.TaskKindGo:
		return d.Go.Entrypoint, nil
	case build.TaskKindNode:
		return d.Node.Entrypoint, nil
	case build.TaskKindPython:
		return d.Python.Entrypoint, nil
	case build.TaskKindShell:
		return d.Shell.Entrypoint, nil
	case build.TaskKindSQL:
		return d.SQL.Entrypoint, nil
	case build.TaskKindDockerfile, build.TaskKindImage, build.TaskKindREST:
		return "", ErrNoEntrypoint
	default:
		return "", errors.Errorf("unhandled kind: %s", kind)
	}
}

func (d *Definition_0_3) UpgradeJST() error {
	kind, err := d.Kind()
	if err != nil {
		return err
	}

	switch kind {
	case build.TaskKindDeno:
		d.Deno.Arguments = upgradeArguments(d.Deno.Arguments)
		return nil
	case build.TaskKindGo:
		d.Go.Arguments = upgradeArguments(d.Go.Arguments)
		return nil
	case build.TaskKindNode:
		d.Node.Arguments = upgradeArguments(d.Node.Arguments)
		return nil
	case build.TaskKindPython:
		d.Python.Arguments = upgradeArguments(d.Python.Arguments)
		return nil
	case build.TaskKindDockerfile, build.TaskKindImage, build.TaskKindShell,
		build.TaskKindSQL, build.TaskKindREST:
		return nil
	default:
		return errors.Errorf("unhandled kind: %s", kind)
	}
}
