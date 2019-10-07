package main

import (
	"fmt"
	"github.com/microsoft/terraform-provider-azuredevops/utils/converter"
	"github.com/microsoft/terraform-provider-azuredevops/utils/tfhelper"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/operations"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,

		//https://godoc.org/github.com/hashicorp/terraform/helper/schema#Schema
		Schema: map[string]*schema.Schema{
			"project_name": &schema.Schema{
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				DiffSuppressFunc: tfhelper.DiffFuncSupressCaseSensitivity,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"visibility": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "private",
				ValidateFunc: validation.StringInSlice([]string{"private", "public"}, false),
			},
			"version_control": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "Git",
				ValidateFunc: validation.StringInSlice([]string{"Git", "Tfvc"}, true),
			},
			"work_item_template": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Agile",
			},
			"process_template_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceProjectCreate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*aggregatedClient)
	project, err := expandProject(clients, d)
	if err != nil {
		return fmt.Errorf("Error converting terraform data model to AzDO project reference: %+v", err)
	}

	err = createProject(clients, project)
	if err != nil {
		return fmt.Errorf("Error creating project in Azure DevOps: %+v", err)
	}

	d.Set("project_name", *project.Name)
	return resourceProjectRead(d, m)
}

// Make API call to create the project and wait for an async success/fail response from the service
func createProject(clients *aggregatedClient, project *core.TeamProject) error {
	operationRef, err := clients.CoreClient.QueueCreateProject(clients.ctx, core.QueueCreateProjectArgs{ProjectToCreate: project})
	if err != nil {
		return err
	}

	err = waitForAsyncOperationSuccess(clients, operationRef)
	if err != nil {
		return err
	}

	return nil
}

func waitForAsyncOperationSuccess(clients *aggregatedClient, operationRef *operations.OperationReference) error {
	maxAttempts := 30
	currentAttempt := 1

	for currentAttempt <= maxAttempts {
		result, err := clients.OperationsClient.GetOperation(clients.ctx, operations.GetOperationArgs{
			OperationId: operationRef.Id,
			PluginId:    operationRef.PluginId,
		})

		if err != nil {
			return err
		}

		if *result.Status == operations.OperationStatusValues.Succeeded {
			// Sometimes without the sleep, the subsequent operations won't find the project...
			time.Sleep(2 * time.Second)
			return nil
		}

		currentAttempt++
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("Operation was not successful after %d attempts", maxAttempts)
}

func resourceProjectRead(d *schema.ResourceData, m interface{}) error {
	clients := m.(*aggregatedClient)

	projectID := d.Id()
	if projectID == "" {
		// project name can be used as an identifier for the core.Projects API:
		//	https://docs.microsoft.com/en-us/rest/api/azure/devops/core/projects/get?view=azure-devops-rest-5.0
		projectID = d.Get("project_name").(string)
	}
	project, err := clients.CoreClient.GetProject(clients.ctx, core.GetProjectArgs{
		ProjectId:           &projectID,
		IncludeCapabilities: converter.Bool(true),
		IncludeHistory:      converter.Bool(false),
	})
	if err != nil {
		return fmt.Errorf("Error looking up project given ID: %v %v", projectID, err)
	}

	err = flattenProject(clients, d, project)
	if err != nil {
		return fmt.Errorf("Error flattening project: %v", err)
	}
	return nil
}

func resourceProjectUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceProjectRead(d, m)
}

func resourceProjectDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}

// Convert internal Terraform data structure to an AzDO data structure
func expandProject(clients *aggregatedClient, d *schema.ResourceData) (*core.TeamProject, error) {
	workItemTemplate := d.Get("work_item_template").(string)
	processTemplateID, err := lookupProcessTemplateID(clients, workItemTemplate)
	if err != nil {
		return nil, err
	}

	// an "error" is OK here as it is expected in the case that the ID is not set in the resource data
	var projectID *uuid.UUID
	parsedID, err := uuid.Parse(d.Id())
	if err == nil {
		projectID = &parsedID
	}

	visibility := d.Get("visibility").(string)
	project := &core.TeamProject{
		Id:          projectID,
		Name:        converter.String(d.Get("project_name").(string)),
		Description: converter.String(d.Get("description").(string)),
		Visibility:  convertVisibilty(visibility),
		Capabilities: &map[string]map[string]string{
			"versioncontrol": map[string]string{
				"sourceControlType": d.Get("version_control").(string),
			},
			"processTemplate": map[string]string{
				"templateTypeId": processTemplateID,
			},
		},
	}

	return project, nil
}

func convertVisibilty(v string) *core.ProjectVisibility {
	if strings.ToLower(v) == "public" {
		return &core.ProjectVisibilityValues.Public
	}
	return &core.ProjectVisibilityValues.Private
}

func flattenProject(clients *aggregatedClient, d *schema.ResourceData, project *core.TeamProject) error {
	description := converter.ToString(project.Description, "")
	processTemplateID := (*project.Capabilities)["processTemplate"]["templateTypeId"]
	processTemplateName, err := lookupProcessTemplateName(clients, processTemplateID)

	if err != nil {
		return err
	}

	d.SetId(project.Id.String())
	d.Set("project_name", *project.Name)
	d.Set("visibility", *project.Visibility)
	d.Set("description", description)
	d.Set("version_control", (*project.Capabilities)["versioncontrol"]["sourceControlType"])
	d.Set("process_template_id", processTemplateID)
	d.Set("work_item_template", processTemplateName)

	return nil
}

// given a process template name, get the process template ID
func lookupProcessTemplateID(clients *aggregatedClient, templateName string) (string, error) {
	processes, err := clients.CoreClient.GetProcesses(clients.ctx, core.GetProcessesArgs{})
	if err != nil {
		return "", err
	}

	for _, p := range *processes {
		if *p.Name == templateName {
			return p.Id.String(), nil
		}
	}

	return "", fmt.Errorf("No process template found")
}

// given a process template ID, get the process template name
func lookupProcessTemplateName(clients *aggregatedClient, templateID string) (string, error) {
	id, err := uuid.Parse(templateID)
	if err != nil {
		return "", fmt.Errorf("Error parsing Work Item Template ID, got %s: %v", templateID, err)
	}

	process, err := clients.CoreClient.GetProcessById(clients.ctx, core.GetProcessByIdArgs{
		ProcessId: &id,
	})

	if err != nil {
		return "", fmt.Errorf("Error looking up template by ID: %v", err)
	}

	return *process.Name, nil
}
