package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/operations"
	"github.com/microsoft/azure-devops-go-api/azuredevops/serviceendpoint"
	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/graph"
)

// Aggregates all of the underlying clients into a single data
// type. Each client is ready to use and fully configured with the correct
// AzDO PAT/organization
//
// AggregatedClient uses interfaces derived from the underlying client structs to
// allow for mocking to support unit testing of the funcs that invoke the
// Azure DevOps client.
type aggregatedClient struct {
	CoreClient            CoreClient
	BuildClient           BuildClient
	OperationsClient      OperationsClient
	ServiceEndpointClient ServiceEndpointClient
	GraphClient			  GraphClient
	ctx                   context.Context
}

// Use ifacemaker ( https://github.com/vburenin/ifacemaker ) to pull the interfaces for required clients
// from the relevant client.go file under github.com/microsoft/azure-devops-go-api/azuredevops .

// GraphClient was pulled from https://github.com/microsoft/azure-devops-go-api/blob/dev/azuredevops/graph/client.go
type GraphClient interface {
	// DeleteAvatar(ctx context.Context, args graph.DeleteAvatarArgs) error	
	// GetAvatar(ctx context.Context, args graph.GetAvatarArgs) (*profile.Avatar, error)
	// SetAvatar(ctx context.Context, args graph.SetAvatarArgs) error
	GetDescriptor(ctx context.Context, args graph.GetDescriptorArgs) (*graph.GraphDescriptorResult, error)
	CreateGroup(ctx context.Context, args graph.CreateGroupArgs) (*graph.GraphGroup, error)
	DeleteGroup(ctx context.Context, args graph.DeleteGroupArgs) error
	GetGroup(ctx context.Context, args graph.GetGroupArgs) (*graph.GraphGroup, error)
	ListGroups(ctx context.Context, args graph.ListGroupsArgs) (*graph.PagedGraphGroups, error)
	UpdateGroup(ctx context.Context, args graph.UpdateGroupArgs) (*graph.GraphGroup, error)
	AddMembership(ctx context.Context, args graph.AddMembershipArgs) (*graph.GraphMembership, error) 
	CheckMembershipExistence(ctx context.Context, args graph.CheckMembershipExistenceArgs) error 
	GetMembership(ctx context.Context, args graph.GetMembershipArgs) (*graph.GraphMembership, error)
	RemoveMembership(ctx context.Context, args graph.RemoveMembershipArgs) error 
	ListMemberships(ctx context.Context, args graph.ListMembershipsArgs) (*[]graph.GraphMembership, error)
	GetMembershipState(ctx context.Context, args graph.GetMembershipStateArgs) (*graph.GraphMembershipState, error)
	GetProviderInfo(ctx context.Context, args graph.GetProviderInfoArgs) (*graph.GraphProviderInfo, error)
	RequestAccess(ctx context.Context, args graph.RequestAccessArgs) error 
	GetStorageKey(ctx context.Context, args graph.GetStorageKeyArgs) (*graph.GraphStorageKeyResult, error)
	LookupSubjects(ctx context.Context, args graph.LookupSubjectsArgs) (*map[string]graph.GraphSubject, error)
	CreateUser(ctx context.Context, args graph.CreateUserArgs) (*graph.GraphUser, error) 
	DeleteUser(ctx context.Context, args graph.DeleteUserArgs) error 
	GetUser(ctx context.Context, args graph.GetUserArgs) (*graph.GraphUser, error)
	ListUsers(ctx context.Context, args graph.ListUsersArgs) (*graph.PagedGraphUsers, error)
	UpdateUser(ctx context.Context, args graph.UpdateUserArgs) (*graph.GraphUser, error) 	
}

// BuildClient was pulled from https://github.com/microsoft/azure-devops-go-api/blob/dev/azuredevops/build/client.go
type BuildClient interface {
	CreateArtifact(ctx context.Context, args build.CreateArtifactArgs) (*build.BuildArtifact, error)
	GetArtifact(ctx context.Context, args build.GetArtifactArgs) (*build.BuildArtifact, error)
	GetArtifactContentZip(ctx context.Context, args build.GetArtifactContentZipArgs) (io.ReadCloser, error)
	GetArtifacts(ctx context.Context, args build.GetArtifactsArgs) (*[]build.BuildArtifact, error)
	GetFile(ctx context.Context, args build.GetFileArgs) (io.ReadCloser, error)
	GetAttachments(ctx context.Context, args build.GetAttachmentsArgs) (*[]build.Attachment, error)
	GetAttachment(ctx context.Context, args build.GetAttachmentArgs) (io.ReadCloser, error)
	AuthorizeProjectResources(ctx context.Context, args build.AuthorizeProjectResourcesArgs) (*[]build.DefinitionResourceReference, error)
	GetProjectResources(ctx context.Context, args build.GetProjectResourcesArgs) (*[]build.DefinitionResourceReference, error)
	ListBranches(ctx context.Context, args build.ListBranchesArgs) (*[]string, error)
	GetBuildBadge(ctx context.Context, args build.GetBuildBadgeArgs) (*build.BuildBadge, error)
	GetBuildBadgeData(ctx context.Context, args build.GetBuildBadgeDataArgs) (*string, error)
	DeleteBuild(ctx context.Context, args build.DeleteBuildArgs) error
	GetBuild(ctx context.Context, args build.GetBuildArgs) (*build.Build, error)
	GetBuilds(ctx context.Context, args build.GetBuildsArgs) (*build.GetBuildsResponseValue, error)
	QueueBuild(ctx context.Context, args build.QueueBuildArgs) (*build.Build, error)
	UpdateBuild(ctx context.Context, args build.UpdateBuildArgs) (*build.Build, error)
	UpdateBuilds(ctx context.Context, args build.UpdateBuildsArgs) (*[]build.Build, error)
	GetBuildChanges(ctx context.Context, args build.GetBuildChangesArgs) (*build.GetBuildChangesResponseValue, error)
	GetChangesBetweenBuilds(ctx context.Context, args build.GetChangesBetweenBuildsArgs) (*[]build.Change, error)
	GetBuildController(ctx context.Context, args build.GetBuildControllerArgs) (*build.BuildController, error)
	GetBuildControllers(ctx context.Context, args build.GetBuildControllersArgs) (*[]build.BuildController, error)
	CreateDefinition(ctx context.Context, args build.CreateDefinitionArgs) (*build.BuildDefinition, error)
	DeleteDefinition(ctx context.Context, args build.DeleteDefinitionArgs) error
	GetDefinition(ctx context.Context, args build.GetDefinitionArgs) (*build.BuildDefinition, error)
	GetDefinitions(ctx context.Context, args build.GetDefinitionsArgs) (*build.GetDefinitionsResponseValue, error)
	RestoreDefinition(ctx context.Context, args build.RestoreDefinitionArgs) (*build.BuildDefinition, error)
	UpdateDefinition(ctx context.Context, args build.UpdateDefinitionArgs) (*build.BuildDefinition, error)
	GetFileContents(ctx context.Context, args build.GetFileContentsArgs) (io.ReadCloser, error)
	CreateFolder(ctx context.Context, args build.CreateFolderArgs) (*build.Folder, error)
	DeleteFolder(ctx context.Context, args build.DeleteFolderArgs) error
	GetFolders(ctx context.Context, args build.GetFoldersArgs) (*[]build.Folder, error)
	UpdateFolder(ctx context.Context, args build.UpdateFolderArgs) (*build.Folder, error)
	GetLatestBuild(ctx context.Context, args build.GetLatestBuildArgs) (*build.Build, error)
	GetBuildLog(ctx context.Context, args build.GetBuildLogArgs) (io.ReadCloser, error)
	GetBuildLogLines(ctx context.Context, args build.GetBuildLogLinesArgs) (*[]string, error)
	GetBuildLogs(ctx context.Context, args build.GetBuildLogsArgs) (*[]build.BuildLog, error)
	GetBuildLogsZip(ctx context.Context, args build.GetBuildLogsZipArgs) (io.ReadCloser, error)
	GetBuildLogZip(ctx context.Context, args build.GetBuildLogZipArgs) (io.ReadCloser, error)
	GetProjectMetrics(ctx context.Context, args build.GetProjectMetricsArgs) (*[]build.BuildMetric, error)
	GetDefinitionMetrics(ctx context.Context, args build.GetDefinitionMetricsArgs) (*[]build.BuildMetric, error)
	GetBuildOptionDefinitions(ctx context.Context, args build.GetBuildOptionDefinitionsArgs) (*[]build.BuildOptionDefinition, error)
	GetPathContents(ctx context.Context, args build.GetPathContentsArgs) (*[]build.SourceRepositoryItem, error)
	GetBuildProperties(ctx context.Context, args build.GetBuildPropertiesArgs) (interface{}, error)
	UpdateBuildProperties(ctx context.Context, args build.UpdateBuildPropertiesArgs) (interface{}, error)
	GetDefinitionProperties(ctx context.Context, args build.GetDefinitionPropertiesArgs) (interface{}, error)
	UpdateDefinitionProperties(ctx context.Context, args build.UpdateDefinitionPropertiesArgs) (interface{}, error)
	GetPullRequest(ctx context.Context, args build.GetPullRequestArgs) (*build.PullRequest, error)
	GetBuildReport(ctx context.Context, args build.GetBuildReportArgs) (*build.BuildReportMetadata, error)
	GetBuildReportHtmlContent(ctx context.Context, args build.GetBuildReportHtmlContentArgs) (io.ReadCloser, error)
	ListRepositories(ctx context.Context, args build.ListRepositoriesArgs) (*build.SourceRepositories, error)
	AuthorizeDefinitionResources(ctx context.Context, args build.AuthorizeDefinitionResourcesArgs) (*[]build.DefinitionResourceReference, error)
	GetDefinitionResources(ctx context.Context, args build.GetDefinitionResourcesArgs) (*[]build.DefinitionResourceReference, error)
	GetResourceUsage(ctx context.Context, args build.GetResourceUsageArgs) (*build.BuildResourceUsage, error)
	GetDefinitionRevisions(ctx context.Context, args build.GetDefinitionRevisionsArgs) (*[]build.BuildDefinitionRevision, error)
	GetBuildSettings(ctx context.Context, args build.GetBuildSettingsArgs) (*build.BuildSettings, error)
	UpdateBuildSettings(ctx context.Context, args build.UpdateBuildSettingsArgs) (*build.BuildSettings, error)
	ListSourceProviders(ctx context.Context, args build.ListSourceProvidersArgs) (*[]build.SourceProviderAttributes, error)
	GetStatusBadge(ctx context.Context, args build.GetStatusBadgeArgs) (*string, error)
	AddBuildTag(ctx context.Context, args build.AddBuildTagArgs) (*[]string, error)
	AddBuildTags(ctx context.Context, args build.AddBuildTagsArgs) (*[]string, error)
	DeleteBuildTag(ctx context.Context, args build.DeleteBuildTagArgs) (*[]string, error)
	GetBuildTags(ctx context.Context, args build.GetBuildTagsArgs) (*[]string, error)
	GetTags(ctx context.Context, args build.GetTagsArgs) (*[]string, error)
	AddDefinitionTag(ctx context.Context, args build.AddDefinitionTagArgs) (*[]string, error)
	AddDefinitionTags(ctx context.Context, args build.AddDefinitionTagsArgs) (*[]string, error)
	DeleteDefinitionTag(ctx context.Context, args build.DeleteDefinitionTagArgs) (*[]string, error)
	GetDefinitionTags(ctx context.Context, args build.GetDefinitionTagsArgs) (*[]string, error)
	DeleteTemplate(ctx context.Context, args build.DeleteTemplateArgs) error
	GetTemplate(ctx context.Context, args build.GetTemplateArgs) (*build.BuildDefinitionTemplate, error)
	GetTemplates(ctx context.Context, args build.GetTemplatesArgs) (*[]build.BuildDefinitionTemplate, error)
	SaveTemplate(ctx context.Context, args build.SaveTemplateArgs) (*build.BuildDefinitionTemplate, error)
	GetBuildTimeline(ctx context.Context, args build.GetBuildTimelineArgs) (*build.Timeline, error)
	RestoreWebhooks(ctx context.Context, args build.RestoreWebhooksArgs) error
	ListWebhooks(ctx context.Context, args build.ListWebhooksArgs) (*[]build.RepositoryWebhook, error)
	GetBuildWorkItemsRefs(ctx context.Context, args build.GetBuildWorkItemsRefsArgs) (*[]webapi.ResourceRef, error)
	GetBuildWorkItemsRefsFromCommits(ctx context.Context, args build.GetBuildWorkItemsRefsFromCommitsArgs) (*[]webapi.ResourceRef, error)
	GetWorkItemsBetweenBuilds(ctx context.Context, args build.GetWorkItemsBetweenBuildsArgs) (*[]webapi.ResourceRef, error)
}

// CoreClient was pulled from https://github.com/microsoft/azure-devops-go-api/blob/dev/azuredevops/core/client.go
type CoreClient interface {
	RemoveProjectAvatar(ctx context.Context, args core.RemoveProjectAvatarArgs) error
	SetProjectAvatar(ctx context.Context, args core.SetProjectAvatarArgs) error
	CreateConnectedService(ctx context.Context, args core.CreateConnectedServiceArgs) (*core.WebApiConnectedService, error)
	GetConnectedServiceDetails(ctx context.Context, args core.GetConnectedServiceDetailsArgs) (*core.WebApiConnectedServiceDetails, error)
	GetConnectedServices(ctx context.Context, args core.GetConnectedServicesArgs) (*[]core.WebApiConnectedService, error)
	GetTeamMembersWithExtendedProperties(ctx context.Context, args core.GetTeamMembersWithExtendedPropertiesArgs) (*[]webapi.TeamMember, error)
	GetProcessById(ctx context.Context, args core.GetProcessByIdArgs) (*core.Process, error)
	GetProcesses(ctx context.Context, args core.GetProcessesArgs) (*[]core.Process, error)
	GetProjectCollection(ctx context.Context, args core.GetProjectCollectionArgs) (*core.TeamProjectCollection, error)
	GetProjectCollections(ctx context.Context, args core.GetProjectCollectionsArgs) (*[]core.TeamProjectCollectionReference, error)
	GetProject(ctx context.Context, args core.GetProjectArgs) (*core.TeamProject, error)
	GetProjects(ctx context.Context, args core.GetProjectsArgs) (*core.GetProjectsResponseValue, error)
	QueueCreateProject(ctx context.Context, args core.QueueCreateProjectArgs) (*operations.OperationReference, error)
	QueueDeleteProject(ctx context.Context, args core.QueueDeleteProjectArgs) (*operations.OperationReference, error)
	UpdateProject(ctx context.Context, args core.UpdateProjectArgs) (*operations.OperationReference, error)
	GetProjectProperties(ctx context.Context, args core.GetProjectPropertiesArgs) (*[]core.ProjectProperty, error)
	SetProjectProperties(ctx context.Context, args core.SetProjectPropertiesArgs) error
	CreateOrUpdateProxy(ctx context.Context, args core.CreateOrUpdateProxyArgs) (*core.Proxy, error)
	DeleteProxy(ctx context.Context, args core.DeleteProxyArgs) error
	GetProxies(ctx context.Context, args core.GetProxiesArgs) (*[]core.Proxy, error)
	CreateTeam(ctx context.Context, args core.CreateTeamArgs) (*core.WebApiTeam, error)
	DeleteTeam(ctx context.Context, args core.DeleteTeamArgs) error
	GetTeam(ctx context.Context, args core.GetTeamArgs) (*core.WebApiTeam, error)
	GetTeams(ctx context.Context, args core.GetTeamsArgs) (*[]core.WebApiTeam, error)
	UpdateTeam(ctx context.Context, args core.UpdateTeamArgs) (*core.WebApiTeam, error)
	GetAllTeams(ctx context.Context, args core.GetAllTeamsArgs) (*[]core.WebApiTeam, error)
}

// OperationsClient was pulled from https://github.com/microsoft/azure-devops-go-api/blob/dev/azuredevops/build/client.go
type OperationsClient interface {
	GetOperation(ctx context.Context, args operations.GetOperationArgs) (*operations.Operation, error)
}

// ServiceEndpointClient was pulled from https://github.com/microsoft/azure-devops-go-api/blob/dev/azuredevops/serviceendpoint/client.go
type ServiceEndpointClient interface {
	ExecuteServiceEndpointRequest(ctx context.Context, args serviceendpoint.ExecuteServiceEndpointRequestArgs) (*serviceendpoint.ServiceEndpointRequestResult, error)
	CreateServiceEndpoint(ctx context.Context, args serviceendpoint.CreateServiceEndpointArgs) (*serviceendpoint.ServiceEndpoint, error)
	DeleteServiceEndpoint(ctx context.Context, args serviceendpoint.DeleteServiceEndpointArgs) error
	GetServiceEndpointDetails(ctx context.Context, args serviceendpoint.GetServiceEndpointDetailsArgs) (*serviceendpoint.ServiceEndpoint, error)
	GetServiceEndpoints(ctx context.Context, args serviceendpoint.GetServiceEndpointsArgs) (*[]serviceendpoint.ServiceEndpoint, error)
	GetServiceEndpointsByNames(ctx context.Context, args serviceendpoint.GetServiceEndpointsByNamesArgs) (*[]serviceendpoint.ServiceEndpoint, error)
	UpdateServiceEndpoint(ctx context.Context, args serviceendpoint.UpdateServiceEndpointArgs) (*serviceendpoint.ServiceEndpoint, error)
	UpdateServiceEndpoints(ctx context.Context, args serviceendpoint.UpdateServiceEndpointsArgs) (*[]serviceendpoint.ServiceEndpoint, error)
	GetServiceEndpointExecutionRecords(ctx context.Context, args serviceendpoint.GetServiceEndpointExecutionRecordsArgs) (*serviceendpoint.GetServiceEndpointExecutionRecordsResponseValue, error)
	GetServiceEndpointTypes(ctx context.Context, args serviceendpoint.GetServiceEndpointTypesArgs) (*[]serviceendpoint.ServiceEndpointType, error)
}

func getAzdoClient(azdoPAT string, organizationURL string) (*aggregatedClient, error) {
	ctx := context.Background()

	if azdoPAT == "" {
		return nil, fmt.Errorf("the personal access token is required")
	}

	if organizationURL == "" {
		return nil, fmt.Errorf("the url of the Azure DevOps is required")
	}

	connection := azuredevops.NewPatConnection(organizationURL, azdoPAT)

	// client for these APIs (includes CRUD for AzDO projects...):
	//	https://docs.microsoft.com/en-us/rest/api/azure/devops/core/?view=azure-devops-rest-5.1
	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		log.Printf("getAzdoClient(): core.NewClient failed.")
		return nil, err
	}

	// client for these APIs (includes CRUD for AzDO build pipelines...):
	//	https://docs.microsoft.com/en-us/rest/api/azure/devops/build/?view=azure-devops-rest-5.1
	buildClient, err := build.NewClient(ctx, connection)
	if err != nil {
		log.Printf("getAzdoClient(): build.NewClient failed.")
		return nil, err
	}

	// client for these APIs (monitor async operations...):
	//	https://docs.microsoft.com/en-us/rest/api/azure/devops/operations/operations?view=azure-devops-rest-5.1
	operationsClient := operations.NewClient(ctx, connection)

	// client for these APIs (includes CRUD for AzDO service endpoints a.k.a. service connections...):
	//  https://docs.microsoft.com/en-us/rest/api/azure/devops/serviceendpoint/endpoints?view=azure-devops-rest-5.1
	serviceEndpointClient, err := serviceendpoint.NewClient(ctx, connection)
	if err != nil {
		log.Printf("getAzdoClient(): serviceendpoint.NewClient failed.")
		return nil, err
	}

	graphClient, err := graph.NewClient(ctx, connection)
	if err != nil {
		log.Printf("getAzdoClient(): graphClient.NewClient failed.")
		return nil, err
	}

	aggregatedClient := &aggregatedClient{
		CoreClient:            coreClient,
		BuildClient:           buildClient,
		OperationsClient:      operationsClient,
		ServiceEndpointClient: serviceEndpointClient,
		GraphClient:		   graphClient,
		ctx:                   ctx,
	}

	log.Printf("getAzdoClient(): Created core, build, operations, and serviceendpoint clients successfully!")
	return aggregatedClient, nil
}
