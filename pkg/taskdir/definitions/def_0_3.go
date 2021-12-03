package definitions

import (
	_ "embed"
	"encoding/json"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/lib/pkg/build"
	"github.com/pkg/errors"
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

func (d Definition_0_3) Contents(format TaskDefFormat) ([]byte, error) {
	buf, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		return nil, err
	}

	switch format {
	case TaskDefFormatYAML:
		return nil, errors.New("NotImplemented")
	case TaskDefFormatJSON:
		// nothing
	default:
		return nil, errors.Errorf("unknown format: %s", format)
	}

	return buf, nil
}
