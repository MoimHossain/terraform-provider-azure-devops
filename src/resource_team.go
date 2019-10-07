package main

import (
	"fmt"
	"github.com/microsoft/terraform-provider-azuredevops/utils/converter"
	"github.com/microsoft/terraform-provider-azuredevops/utils/tfhelper"
	"os"
	"log"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/graph"
)

func resourceTeam() *schema.Resource {
	return &schema.Resource{
		Create: resourceTeamCreate,
		Read:   resourceTeamRead,
		Update: resourceTeamUpdate,
		Delete: resourceTeamDelete,

		//https://godoc.org/github.com/hashicorp/terraform/helper/schema#Schema
		Schema: map[string]*schema.Schema{
			"project_id": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: tfhelper.DiffFuncSupressCaseSensitivity,
			},
			"name": &schema.Schema{
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
			"members": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"admin": {
							Type:      schema.TypeBool,
							Required:  true,
							Sensitive: true,
						},
					},
				},
			},			
		},
	}
}

func resourceTeamCreate(d *schema.ResourceData, m interface{}) error {	
	clients := m.(*aggregatedClient)
	team, err := expandTeam(clients, d)
	if err != nil {
		return fmt.Errorf("Error converting terraform data model to AzDO team reference: %+v", err)
	}

	team, err = createTeam(clients, team)
	if err != nil {
		return fmt.Errorf("Error creating team in Azure DevOps: %+v", err)
	}

	err = expandMemberSet(d, clients)

	d.SetId(team.Id.String())
	return resourceTeamRead(d, m)
}

// Make API call to create the team and wait for an async success/fail response from the service
func createTeam(clients *aggregatedClient, team *core.WebApiTeam) (*core.WebApiTeam, error) {

	pid := team.ProjectId.String()


	teamRetVal, err := clients.CoreClient.CreateTeam(clients.ctx, core.CreateTeamArgs{Team: team, ProjectId: &pid})
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return teamRetVal, nil
}

func resourceTeamRead(d *schema.ResourceData, m interface{}) error {
	clients := m.(*aggregatedClient)

	teamID := d.Id()
	projectID := d.Get("project_id").(string)

	team, err := clients.CoreClient.GetTeam(clients.ctx, core.GetTeamArgs{
		TeamId:           &teamID,
		ProjectId:        &projectID,
		ExpandIdentity:   converter.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("Error looking up team given ID: %v %v", projectID, err)
	}

	err = flattenTeam(clients, d, team)
	if err != nil {
		return fmt.Errorf("Error flattening team: %v", err)
	}
	return nil
}

func resourceTeamUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceTeamRead(d, m)
}

func resourceTeamDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}

// Convert internal Terraform data structure to an AzDO data structure
func expandTeam(clients *aggregatedClient, d *schema.ResourceData) (*core.WebApiTeam, error) {
	var projectID *uuid.UUID
	parsedID, err := uuid.Parse(d.Get("project_id").(string))
	if err == nil {
		projectID = &parsedID
	}

	team := &core.WebApiTeam{
		ProjectId:   projectID,
		Name:        converter.String(d.Get("name").(string)),
		Description: converter.String(d.Get("description").(string)),
	}
	return team, nil
}


func flattenTeam(clients *aggregatedClient, d *schema.ResourceData, team *core.WebApiTeam) error {

	d.SetId(team.Id.String())
	d.Set("project_name", converter.ToString(team.ProjectName, ""))
	d.Set("project_id", team.ProjectId)
	d.Set("name", converter.ToString(team.Name, ""))
	d.Set("description", converter.ToString(team.Description, ""))

	return nil
}

func expandMemberSet(d *schema.ResourceData, clients *aggregatedClient) error {

	f, _ := os.OpenFile("text.log",
	os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)	
	defer f.Close()
	logger := log.New(f, "prefix", log.LstdFlags)


	response, err := clients.GraphClient.ListGroups(clients.ctx, graph.ListGroupsArgs{	})
	if err != nil {
		logger.Println(fmt.Sprintf("Error %v\n", err))
		return fmt.Errorf("Error looking up Groups. ")
	}

	var p []graph.GraphGroup
	p = *response.GraphGroups

	
	logger.Println(fmt.Sprintf("Inside loop %d\n", len(p)))
	for _, item := range *response.GraphGroups {
		logger.Println(fmt.Sprintf("Inside loop %s\n", item.Descriptor))
	}

	input := d.Get("members").(*schema.Set).List()	

	logger.Println(fmt.Sprintf("%d\n", len(input)))


	for _, v := range input {
		vals := v.(map[string]interface{})

		upn := vals["name"].(string)
		admin := vals["admin"].(bool)


		logger.Println(fmt.Sprintf("%s- %t\n", upn, admin))
		
	}
	return nil
}
