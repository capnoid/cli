package deploy

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/build"
	"github.com/airplanedev/cli/pkg/cmd/tasks/deploy/discover"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
	"github.com/airplanedev/cli/pkg/utils/pointers"
	libBuild "github.com/airplanedev/lib/pkg/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployTasks(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	fixturesPath, _ := filepath.Abs("./fixtures")
	testCases := []struct {
		desc          string
		taskConfigs   []discover.TaskConfig
		existingTasks map[string]api.Task
		changedFiles  []string
		updatedTasks  map[string]api.Task
	}{
		{
			desc: "no tasks",
		},
		{
			desc: "tasks filtered out by changed files",
			taskConfigs: []discover.TaskConfig{
				{
					TaskRoot: "some/other/root.js",
				},
			},
			changedFiles: []string{"some/random/path.js"},
		},
		{
			desc: "deploys and updates a task",
			taskConfigs: []discover.TaskConfig{
				{
					TaskRoot: fixturesPath,
					Def: &definitions.Definition_0_3{
						Name: "My Task",
						Slug: "my_task",
						Node: &definitions.NodeDefinition_0_3{},
					},
					Task: api.Task{
						Slug: "my_task",
						Name: "My Task",
					},
				},
			},
			existingTasks: map[string]api.Task{"my_task": {Slug: "my_task", Name: "My Task"}},
			updatedTasks: map[string]api.Task{
				"my_task": {
					Name:       "My Task",
					Slug:       "my_task",
					Image:      pointers.String("imageURL"),
					Parameters: api.Parameters{},
					Kind:       "node",
					KindOptions: libBuild.KindOptions{
						"entrypoint":  "",
						"nodeVersion": "",
					},
				},
			},
		},
		{
			desc: "deploys and updates a task that doesn't need to be built",
			taskConfigs: []discover.TaskConfig{
				{
					TaskRoot: fixturesPath,
					Def: &definitions.Definition_0_3{
						Name:  "My Task",
						Slug:  "my_task",
						Image: &definitions.ImageDefinition_0_3{Image: "myImage"},
					},
					Task: api.Task{
						Slug: "my_task",
						Name: "My Task",
					},
				},
			},
			existingTasks: map[string]api.Task{"my_task": {Slug: "my_task", Name: "My Task"}},
			updatedTasks: map[string]api.Task{
				"my_task": {
					Name:       "My Task",
					Slug:       "my_task",
					Image:      pointers.String("myImage"),
					Parameters: api.Parameters{},
					Kind:       "image",
				},
			},
		},
		// TODO uncomment when sql deploys work.
		// {
		// 	desc: "deploys and updates an SQL task",
		// 	taskConfigs: []discover.TaskConfig{
		// 		{
		// 			TaskRoot: fixturesPath,
		// 			Def: &definitions.Definition_0_3{
		// 				Name: "My Task",
		// 				Slug: "my_task",
		// 				SQL: &definitions.SQLDefinition_0_3{
		// 					Entrypoint: "./fixtures/test.sql",
		// 				},
		// 			},
		// 			Task: api.Task{
		// 				Slug: "my_task",
		// 				Name: "My Task",
		// 			},
		// 		},
		// 	},
		// 	existingTasks: map[string]api.Task{"my_task": {Slug: "my_task", Name: "My Task"}},
		// 	updatedTasks: map[string]api.Task{
		// 		"my_task": {
		// 			Name:       "My Task",
		// 			Slug:       "my_task",
		// 			Parameters: api.Parameters{},
		// 			Kind:       "sql",
		// 		},
		// 	},
		// },
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			client := &api.MockClient{
				Tasks: tC.existingTasks,
			}
			cfg := config{
				changedFiles: tC.changedFiles,
				client:       client,
			}
			d := NewDeployer(cfg, &logger.MockLogger{}, DeployerOpts{
				BuildCreator: &build.MockBuildCreator{},
			})
			err := d.DeployTasks(context.Background(), tC.taskConfigs)
			require.NoError(err)

			assert.Equal(tC.updatedTasks, client.Tasks)
		})
	}
}
