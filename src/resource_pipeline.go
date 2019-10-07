package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"log"
	"strconv"
	"strings"
)

func resourcePipeline() *schema.Resource {
	return &schema.Resource{
		Create: resourcePipelineCreate,
		Read:   resourcePipelineRead,
		Update: resourcePipelineUpdate,
		Delete: resourcePipelineDelete,

		Schema: map[string]*schema.Schema{
			"project_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"revision": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"pipeline_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"repository": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"yml_path": {
							Type:     schema.TypeString,
							Required: true,
						},
						"repo_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"repo_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"branch_name": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "master",
						},
						"service_connection_id": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
		},
	}
}

type buildDefinitionValues struct {
	agentPoolName                 string
	agentPoolID                   int
	pipelineName                  string
	projectID                     string
	projectReference              *core.TeamProjectReference
	repositoryDefaultBranch       string
	repositoryName                string
	repositoryPipelineYmlPath     string
	repositoryType                string
	repositoryURL                 string
	repositoryServiceConnectionID string
}

func resourcePipelineCreate(d *schema.ResourceData, m interface{}) error {
	values := resourceDataToBuildDefinitionValues(d)

	pipelineID, err := createPipeline(m.(*aggregatedClient), values)
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(pipelineID))
	return resourcePipelineRead(d, m)
}

func resourceDataToBuildDefinitionValues(d *schema.ResourceData) *buildDefinitionValues {
	projectID := d.Get("project_id").(string)
	projectUUID := uuid.MustParse(projectID)
	repositories := d.Get("repository").(*schema.Set).List()
	repository := repositories[0].(map[string]interface{})
	repoName := repository["repo_name"].(string)
	pipelineName := d.Get("pipeline_name").(string)
	if pipelineName == "" {
		pipelineName = repoName + "_pipeline"
	}

	return &buildDefinitionValues{
		projectID: projectID,
		projectReference: &core.TeamProjectReference{
			Id: &projectUUID,
		},
		pipelineName:                  pipelineName,
		repositoryName:                repoName,
		repositoryDefaultBranch:       repository["branch_name"].(string),
		repositoryPipelineYmlPath:     repository["yml_path"].(string),
		repositoryType:                repository["repo_type"].(string),
		repositoryURL:                 fmt.Sprintf("https://github.com/%s.git", repoName),
		repositoryServiceConnectionID: repository["service_connection_id"].(string),
		agentPoolName:                 "Hosted Ubuntu 1604",
		agentPoolID:                   224,
	}
}

func createBuildDefinition(values *buildDefinitionValues) *build.BuildDefinition {
	return &build.BuildDefinition{
		Name:    &values.pipelineName,
		Type:    &build.DefinitionTypeValues.Build,
		Quality: &build.DefinitionQualityValues.Definition,
		Queue: &build.AgentPoolQueue{
			Name: &values.agentPoolName,
			Pool: &build.TaskAgentPoolReference{
				Id:   &values.agentPoolID,
				Name: &values.agentPoolName,
			},
		},
		QueueStatus: &build.DefinitionQueueStatusValues.Enabled,
		Repository: &build.BuildRepository{
			Url:           &values.repositoryURL,
			Id:            &values.repositoryName,
			Name:          &values.repositoryName,
			DefaultBranch: &values.repositoryDefaultBranch,
			Type:          &values.repositoryType,
			Properties: &map[string]string{
				"connectedServiceId": values.repositoryServiceConnectionID,
			},
		},
		Process: &build.YamlProcess{
			YamlFilename: &values.repositoryPipelineYmlPath,
		},
		Project: values.projectReference,
	}
}

func createPipeline(clients *aggregatedClient, values *buildDefinitionValues) (int, error) {
	//get info from the client & create a build definition
	createRes, err := clients.BuildClient.CreateDefinition(clients.ctx, build.CreateDefinitionArgs{
		Definition: createBuildDefinition(values),
		Project:    &values.projectID,
	})

	if err != nil {
		return 0, err
	}

	log.Printf("got response: %T", createRes)
	return *(createRes.Id), nil
}

func resourcePipelineRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*aggregatedClient).BuildClient
	projectID := d.Get("project_id").(string)
	pipelineName := d.Get("pipeline_name").(string)

	// Get List of Definitions
	getDefinitionsResponseValue, err := client.GetDefinitions(m.(*aggregatedClient).ctx, build.GetDefinitionsArgs{
		Project: &projectID, // Project ID or project name
	})

	if err != nil {
		return err
	}

	definitionID := -1

	// Find Build with pipelineName, if it exists, save that build's ID
	// TODO: handle ContinuationToken, pagination support for build results...
	for _, buildDefinitionReference := range getDefinitionsResponseValue.Value {
		if strings.TrimRight(*(buildDefinitionReference.Name), "\n") == pipelineName {
			// https://github.com/microsoft/azure-devops-go-api/blob/dev/azuredevops/build/models.go#L451
			definitionID = *(buildDefinitionReference.Id)
			break
		}
	}

	// No existing pipeline definition found.
	if definitionID < 0 {
		d.SetId("")
		return nil
	}

	// Get Build via client, this call has extra data like: properties, tags, jobAuthorizationScope, process, repository
	buildDefinition, err := client.GetDefinition(m.(*aggregatedClient).ctx, build.GetDefinitionArgs{
		Project:      &projectID, // Project ID or project name
		DefinitionId: &definitionID,
	})

	if err != nil {
		return err
	}

	// Save values from buildDefinition into schema, d
	return saveBuildDefinitionToSchema(d, buildDefinition)
}

// Saves passed BuildDefinition values into schema
func saveBuildDefinitionToSchema(d *schema.ResourceData, buildDefinition *build.BuildDefinition) error {
	if buildDefinition.Id != nil {
		d.SetId(strconv.Itoa(*buildDefinition.Id))
	}

	if buildDefinition.Revision != nil {
		d.Set("revision", *buildDefinition.Revision)
	}

	return nil
}

func resourcePipelineDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*aggregatedClient).BuildClient
	if d.Id() != "" {
		projectID := d.Get("project_id").(string)
		definitionID, err := strconv.Atoi(d.Id())

		if err != nil {
			return err
		}

		// returns nil if no error, else returns error
		return client.DeleteDefinition(m.(*aggregatedClient).ctx, build.DeleteDefinitionArgs{
			Project:      &projectID,
			DefinitionId: &definitionID,
		})
	}

	return nil
}

func resourcePipelineUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*aggregatedClient).BuildClient
	values := resourceDataToBuildDefinitionValues(d)

	definitionID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	buildDefinition := createBuildDefinition(values)
	revisionNum := d.Get("revision").(int)
	buildDefinition.Revision = &revisionNum
	buildDefinition.Revision = &revisionNum
	buildDefinition.Id = &definitionID

	_, err = client.UpdateDefinition(m.(*aggregatedClient).ctx, build.UpdateDefinitionArgs{
		Definition:   buildDefinition,
		Project:      &values.projectID,
		DefinitionId: &definitionID,
	})

	if err != nil {
		return err
	}

	return resourcePipelineRead(d, m)
}
