package baremetal_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/acctest"
	baremetalchecks "github.com/scaleway/terraform-provider-scaleway/v2/internal/services/baremetal/testfuncs"
)

func TestAccDataSourceServer_Basic(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccScalewayDataSourceBaremetalServer_Basic"
	name := "TestAccScalewayDataSourceBaremetalServer_Basic"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "my_os" {
						zone = "%s"
						name = "Ubuntu"
						version = "22.04 LTS (Jammy Jellyfish)"						
					}

					data "scaleway_baremetal_offer" "my_offer" {
						zone = "%s"
						name = "%s"
					}

					resource "scaleway_iam_ssh_key" "main" {
						name       = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "main" {
						name        = "%s"
						zone        = "%s"
						description = "test a description"
						offer       = data.scaleway_baremetal_offer.my_offer.offer_id
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
					}
				`, Zone, Zone, OfferName, SSHKeyName, SSHKeyBaremetal, name, Zone),
			},
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "my_os" {
						zone = "%s"
						name = "Ubuntu"
						version = "22.04 LTS (Jammy Jellyfish)"						
					}

					data "scaleway_baremetal_offer" "my_offer" {
						zone = "%s"
						name = "%s"
					}

					resource "scaleway_iam_ssh_key" "main" {
						name       = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "main" {
						name        = "%s"
						zone        = "%s"
						description = "test a description"
						offer       = data.scaleway_baremetal_offer.my_offer.offer_id
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
					}

					data "scaleway_baremetal_server" "by_name" {
						name = "${scaleway_baremetal_server.main.name}"
						zone = "%s"
					}
					
					data "scaleway_baremetal_server" "by_id" {
						server_id = "${scaleway_baremetal_server.main.id}"
						zone = "%s"
					}
				`, Zone, Zone, OfferName, SSHKeyName, SSHKeyBaremetal, name, Zone, Zone, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "data.scaleway_baremetal_server.by_id"),
					testAccCheckBaremetalServerExists(tt, "data.scaleway_baremetal_server.by_name"),
					resource.TestCheckResourceAttrPair(
						"data.scaleway_baremetal_server.by_name", "name",
						"scaleway_baremetal_server.main", "name"),
					resource.TestCheckResourceAttrPair(
						"data.scaleway_baremetal_server.by_id", "name",
						"scaleway_baremetal_server.main", "name"),
				),
			},
		},
	})
}
