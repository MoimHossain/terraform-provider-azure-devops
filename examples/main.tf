
# Make sure to set the following environment variables:
#   AZDO_PERSONAL_ACCESS_TOKEN
#   AZDO_ORG_SERVICE_URL
#   AZDO_GITHUB_SERVICE_CONNECTION_PAT
provider "azuredevops" {
 
}

resource "azuredevops_project" "project" {
  project_name       = "Test Project"
  description        = "Test Project Description"
  visibility         = "private"
  version_control    = "Git"
  work_item_template = "Agile"
}

resource "azuredevops_team" "mytestteam" {
  name               = "Killer Project"
  description        = "Team description"
  project_id         = azuredevops_project.project.id

  members {
    name             = "mohossa@microsoft.com"
    admin            = true
  }
}

