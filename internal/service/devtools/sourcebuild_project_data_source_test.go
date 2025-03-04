package devtools_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/acctest"
)

func TestAccDataSourceNcloudSourceBuildProject(t *testing.T) {
	name := fmt.Sprintf("test-sourcebuild-project-name-%s", acctest.RandString(5))
	repoName := fmt.Sprintf("test-repo-basic-%s", acctest.RandString(5))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { TestAccPreCheck(t) },
		ProtoV5ProviderFactories: ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNcloudSourceBuildProjectConfig(name, repoName),
				Check: resource.ComposeTestCheckFunc(
					TestAccCheckDataSourceID("data.ncloud_sourcebuild_project.project"),
				),
			},
		},
	})
}

func testAccDataSourceNcloudSourceBuildProjectConfig(name string, repoName string) string {
	return fmt.Sprintf(`
data "ncloud_sourcebuild_project_computes" "computes" {
}

data "ncloud_sourcebuild_project_os" "os" {
}

data "ncloud_sourcebuild_project_os_runtimes" "runtimes" {
	os_id = data.ncloud_sourcebuild_project_os.os.os[0].id
}

data "ncloud_sourcebuild_project_os_runtime_versions" "runtime_versions" {
	os_id      = data.ncloud_sourcebuild_project_os.os.os[0].id
	runtime_id = data.ncloud_sourcebuild_project_os_runtimes.runtimes.runtimes[0].id
}

data "ncloud_sourcebuild_project_docker_engines" "docker_engines" {
}
	  
resource "ncloud_sourcecommit_repository" "test-repo" {
	name = "%[1]s"
}

resource "ncloud_sourcebuild_project" "test-project" {
	name        = "%[2]s"
	description = "my build project"
	source {
		type = "SourceCommit"
		config {
			repository_name = ncloud_sourcecommit_repository.test-repo.name
			branch          = "master"
		}
	}
	env {
		compute {
			id = data.ncloud_sourcebuild_project_computes.computes.computes[0].id
		}
		platform {
			type = "SourceBuild"
			config {
				os {
					id = data.ncloud_sourcebuild_project_os.os.os[0].id
				}
				runtime {
					id = data.ncloud_sourcebuild_project_os_runtimes.runtimes.runtimes[0].id
					version {
						id = data.ncloud_sourcebuild_project_os_runtime_versions.runtime_versions.runtime_versions[0].id
					}
				}
			}
		}
	}
}

data "ncloud_sourcebuild_project" "project" {
	id = ncloud_sourcebuild_project.test-project.id
}
`, repoName, name)
}
