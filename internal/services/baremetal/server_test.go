package baremetal_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	baremetalSDK "github.com/scaleway/scaleway-sdk-go/api/baremetal/v1"
	baremetalV3SDK "github.com/scaleway/scaleway-sdk-go/api/baremetal/v3"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/acctest"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/baremetal"
	baremetalchecks "github.com/scaleway/terraform-provider-scaleway/v2/internal/services/baremetal/testfuncs"
)

const SSHKeyBaremetal = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIM7HUxRyQtB2rnlhQUcbDGCZcTJg7OvoznOiyC9W6IxH opensource@scaleway.com"

var jsonConfigPartitioning = "{\"disks\":[{\"device\":\"/dev/nvme0n1\",\"partitions\":[{\"label\":\"uefi\",\"number\":1,\"size\":536870912},{\"label\":\"swap\",\"number\":2,\"size\":4294967296},{\"label\":\"boot\",\"number\":3,\"size\":1073741824},{\"label\":\"root\",\"number\":4,\"size\":1017827045376}]},{\"device\":\"/dev/nvme1n1\",\"partitions\":[{\"label\":\"swap\",\"number\":1,\"size\":4294967296},{\"label\":\"boot\",\"number\":2,\"size\":1073741824},{\"label\":\"root\",\"number\":3,\"size\":1017827045376}]}],\"filesystems\":[{\"device\":\"/dev/nvme0n1p1\",\"format\":\"fat32\",\"mountpoint\":\"/boot/efi\"},{\"device\":\"/dev/md0\",\"format\":\"ext4\",\"mountpoint\":\"/boot\"},{\"device\":\"/dev/md1\",\"format\":\"ext4\",\"mountpoint\":\"/\"}],\"raids\":[{\"devices\":[\"/dev/nvme0n1p3\",\"/dev/nvme1n1p2\"],\"level\":\"raid_level_1\",\"name\":\"/dev/md0\"},{\"devices\":[\"/dev/nvme0n1p4\",\"/dev/nvme1n1p3\"],\"level\":\"raid_level_1\",\"name\":\"/dev/md1\"}],\"zfs\":{\"pools\":[]}}"

func TestAccServer_Basic(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccServer_Basic"
	name := "TestAccServer_Basic"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "my_os" {
					  zone    = "%s"
					  name    = "Ubuntu"
					  version = "22.04 LTS (Jammy Jellyfish)"
					}

					resource "scaleway_iam_ssh_key" "main" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						description = "test a description"
						offer       = "%s"
						os    = data.scaleway_baremetal_os.my_os.os_id
						
						tags = [ "terraform-test", "scaleway_baremetal_server", "minimal" ]
						ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
					}
				`, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone, OfferName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "name", name),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "offer_name", OfferName),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "os", Zone+"/96e5f0f2-d216-4de2-8a15-68730d877885"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "description", "test a description"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.1", "scaleway_baremetal_server"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.2", "minimal"),
					acctest.CheckResourceAttrUUID("scaleway_baremetal_server.base", "ssh_key_ids.0"),
				),
			},
			{
				// Trigger a reinstall and update tags
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "my_os" {
					  zone    = "%s"
					  name    = "Ubuntu"
					  version = "22.04 LTS (Jammy Jellyfish)"
					}

					resource "scaleway_iam_ssh_key" "main" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						description = "test a description"
						offer       = "%s"
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						tags = [ "terraform-test", "scaleway_baremetal_server", "minimal", "edited" ]
						ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
					}
				`, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone, OfferName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "name", name),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "offer_name", OfferName),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "os", Zone+"/96e5f0f2-d216-4de2-8a15-68730d877885"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "description", "test a description"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.#", "4"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.1", "scaleway_baremetal_server"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.2", "minimal"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.3", "edited"),
					acctest.CheckResourceAttrUUID("scaleway_baremetal_server.base", "ssh_key_ids.0"),
				),
			},
		},
	})
}

func TestAccServer_RequiredInstallConfig(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_baremetal_server" "base" {
						name        = "TestAccServer_RequiredInstallConfig"
						zone        = "%s"
						offer       = "%s"
						os          = "7e865c16-1a63-4dc7-8181-dabc020fc21b" // Proxmox

						ssh_key_ids = []
					}`, Zone, OfferName),
				ExpectError: regexp.MustCompile("attribute is required"),
			},
		},
	})
}

func TestAccServer_WithoutInstallConfig(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_offer" "my_offer" {
					  zone = "%s"
					  name = "%s"
					}

					resource "scaleway_baremetal_server" "base" {
					  name 			             = "TestAccScalewayBaremetalServer_WithoutInstallConfig"
                      zone     			         = "%s"
					  offer     				 = data.scaleway_baremetal_offer.my_offer.offer_id
					  install_config_afterward   = true
					}`, Zone, OfferName, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "name", "TestAccScalewayBaremetalServer_WithoutInstallConfig"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "offer_name", OfferName),
					resource.TestCheckNoResourceAttr("scaleway_baremetal_server.base", "os"),
				),
			},
		},
	})
}

func TestAccServer_CreateServerWithCustomInstallConfig(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccServer_CreateServerWithCustomInstallConfig"
	name := "TestAccServer_CreateServerWithCustomInstallConfig"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "my_os" {
					  zone    = "%s"
					  name    = "Ubuntu"
					  version = "22.04 LTS (Jammy Jellyfish)"
					}

					resource "scaleway_iam_ssh_key" "main" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						description = "test a description"
						offer       = "%s"
						os    = data.scaleway_baremetal_os.my_os.os_id
						partitioning = "%s"
						
						tags = [ "terraform-test", "scaleway_baremetal_server", "minimal" ]
						ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
					}
				`, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone, OfferName, strings.ReplaceAll(jsonConfigPartitioning, "\"", "\\\"")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "name", name),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "offer_name", OfferName),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "os", Zone+"/96e5f0f2-d216-4de2-8a15-68730d877885"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "description", "test a description"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.1", "scaleway_baremetal_server"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "tags.2", "minimal"),
					testAccCheckPartitioning(tt, "scaleway_baremetal_server.base", jsonConfigPartitioning),
					acctest.CheckResourceAttrUUID("scaleway_baremetal_server.base", "ssh_key_ids.0"),
				),
			},
		},
	})
}

func TestAccServer_CreateServerWithServicePassword(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	SSHKeyName := "TestAccServer_CreateServerWithServicePassword"
	password := "HelloWorld678!"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "by_id" {
					  zone    = "%s"
					  name    = "Proxmox"
					  version = "VE 8 | Debian 12 (Bookworm)"
					}

					data "scaleway_baremetal_offer" "server_model" {
					  zone                = "%s"
					  name                = "%s"
					  subscription_period = "hourly"
					}

					resource "scaleway_iam_ssh_key" "main" {
						name 	   = "%s"
						public_key = "%s"
					}

					resource "scaleway_baremetal_server" "TP" {
					  zone             = "%s"
					  ssh_key_ids      = [scaleway_iam_ssh_key.main.id]
					  offer            = data.scaleway_baremetal_offer.server_model.offer_id
					  os               = data.scaleway_baremetal_os.by_id.os_id
					  service_password = "%s"
					}
				`, Zone, Zone, OfferName, SSHKeyName, SSHKeyBaremetal, Zone, password),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.TP"),
					resource.TestCheckResourceAttrSet("scaleway_baremetal_server.TP", "id"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.TP", "zone", Zone),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.TP", "offer_name", OfferName),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.TP", "os", "fr-par-2/a5c00c1b-95b1-4c08-8a33-79cc079f9dac"), // Replace with actual os_id if needed
					resource.TestCheckResourceAttrSet("scaleway_baremetal_server.TP", "service_user"),
					acctest.CheckResourceAttrUUID("scaleway_baremetal_server.TP", "ssh_key_ids.0"),
				),
			},
		},
	})
}

func TestAccServer_CreateServerWithOption(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccScalewayBaremetalServer_CreateServerWithOption"
	name := "TestAccScalewayBaremetalServer_CreateServerWithOption"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "scaleway_baremetal_os" "my_os" {
				  zone    = "%s"
				  name    = "Ubuntu"
				  version = "22.04 LTS (Jammy Jellyfish)"
				}
				
				data "scaleway_baremetal_offer" "my_offer" {
				  zone = "%s"
				  name = "%s"
				}
				
				data "scaleway_baremetal_option" "private_network" {
				  zone = "%s"
				  name = "Private Network"
				}
				
				resource "scaleway_iam_ssh_key" "base" {
				  name       = "%s"
				  public_key = "%s"
				}
				
				resource "scaleway_baremetal_server" "base" {
				  name  = "%s"
				  zone  = "%s"
				  offer = data.scaleway_baremetal_offer.my_offer.offer_id
				  os    = data.scaleway_baremetal_os.my_os.os_id
				
				  ssh_key_ids = [scaleway_iam_ssh_key.base.id]
				  options {
					id = data.scaleway_baremetal_option.private_network.option_id
				  }
				}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasOptions(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.base", "options.0.id", "data.scaleway_baremetal_option.private_network", "option_id"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "ips.#", "2"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "ipv4.#", "1"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "ipv4.0.version", "IPv4"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "ipv6.#", "1"),
					resource.TestCheckResourceAttr("scaleway_baremetal_server.base", "ipv6.0.version", "IPv6"),
				),
			},
		},
	})
}

func TestAccServer_AddOption(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccScalewayBaremetalServer_AddOption"
	name := "TestAccScalewayBaremetalServer_AddOption"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "by_id" {
					  zone    = "%s"
					  name    = "Ubuntu"
					  version = "22.04 LTS (Jammy Jellyfish)"
					}
					
					data "scaleway_baremetal_offer" "my_offer" {
					  zone = "%s"
					  name = "%s"
					}
					
					resource "scaleway_iam_ssh_key" "base" {
					  name       = "%s"
					  public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
					  name  = "%s"
					  zone  = "%s"
					  offer = data.scaleway_baremetal_offer.my_offer.offer_id
					  os    = data.scaleway_baremetal_os.by_id.os_id
					
					  ssh_key_ids = [scaleway_iam_ssh_key.base.id]
					}
				`, Zone, Zone, OfferName, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
				),
			},
			{
				Config: fmt.Sprintf(`
				data "scaleway_baremetal_os" "my_os" {
				  zone    = "%s"
				  name    = "Ubuntu"
				  version = "22.04 LTS (Jammy Jellyfish)"
				}
				
				data "scaleway_baremetal_offer" "my_offer" {
				  zone = "%s"
				  name = "%s"
				}
				
				data "scaleway_baremetal_option" "private_network" {
				  zone = "%s"
				  name = "Private Network"
				}
				
				resource "scaleway_iam_ssh_key" "base" {
				  name       = "%s"
				  public_key = "%s"
				}
				
				resource "scaleway_baremetal_server" "base" {
				  name  = "%s"
				  zone  = "%s"
				  offer = data.scaleway_baremetal_offer.my_offer.offer_id
				  os    = data.scaleway_baremetal_os.my_os.os_id
				
				  ssh_key_ids = [scaleway_iam_ssh_key.base.id]
				  options {
					id = data.scaleway_baremetal_option.private_network.option_id
				  }
				}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasOptions(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.base", "options.0.id", "data.scaleway_baremetal_option.private_network", "option_id"),
				),
			},
		},
	})
}

func TestAccServer_AddTwoOptionsThenDeleteOne(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccScalewayBaremetalServer_AddTwoOptionsThenDeleteOne"
	name := "TestAccScalewayBaremetalServer_AddTwoOptionsThenDeleteOne"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      baremetalchecks.CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "by_id" {
					  zone    = "%s"
					  name    = "Ubuntu"
					  version = "22.04 LTS (Jammy Jellyfish)"
					}
					
					data "scaleway_baremetal_offer" "my_offer" {
					  zone = "%s"
					  name = "%s"
					}
					
					resource "scaleway_iam_ssh_key" "base" {
					  name       = "%s"
					  public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
					  name  = "%s"
					  zone  = "%s"
					  offer = data.scaleway_baremetal_offer.my_offer.offer_id
					  os    = data.scaleway_baremetal_os.by_id.os_id
					
					  ssh_key_ids = [scaleway_iam_ssh_key.base.id]
					}
				`, Zone, Zone, OfferName, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "my_os" {
					  zone    = "%s"
					  name    = "Ubuntu"
					  version = "22.04 LTS (Jammy Jellyfish)"
					}
					
					data "scaleway_baremetal_offer" "my_offer" {
					  zone = "%s"
					  name = "%s"
					}
					
					data "scaleway_baremetal_option" "remote_access" {
					  zone = "%s"
					  name = "Remote Access"
					}
					
					data "scaleway_baremetal_option" "private_network" {
					  zone = "%s"
					  name = "Private Network"
					}
					
					resource "scaleway_iam_ssh_key" "base" {
					  name       = "%s"
					  public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
					  name        = "%s"
					  zone        = "%s"
					  offer       = data.scaleway_baremetal_offer.my_offer.offer_id
					  os          = data.scaleway_baremetal_os.my_os.os_id
					  ssh_key_ids = [scaleway_iam_ssh_key.base.id]
					
					  options {
						id = data.scaleway_baremetal_option.private_network.option_id
					  }
					  options {
						id         = data.scaleway_baremetal_option.remote_access.option_id
						expires_at = "2026-07-06T09:00:00Z"
					  }
					}
				`, Zone, Zone, OfferName, Zone, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasOptions(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckTypeSetElemAttrPair("scaleway_baremetal_server.base", "options.*.id", "data.scaleway_baremetal_option.remote_access", "option_id"),
					resource.TestCheckTypeSetElemAttrPair("scaleway_baremetal_server.base", "options.*.id", "data.scaleway_baremetal_option.private_network", "option_id"),
					resource.TestCheckTypeSetElemNestedAttrs("scaleway_baremetal_server.base", "options.*", map[string]string{
						"id":         Zone + "/931df052-d713-4674-8b58-96a63244c8e2",
						"expires_at": "2026-07-06T09:00:00Z",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("scaleway_baremetal_server.base", "options.*", map[string]string{
						"id": Zone + "/cd4158d7-2d65-49be-8803-c4b8ab6f760c",
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_os" "my_os" {
					  zone    = "%s"
					  name    = "Ubuntu"
					  version = "22.04 LTS (Jammy Jellyfish)"
					}
					
					data "scaleway_baremetal_offer" "my_offer" {
					  zone = "%s"
					  name = "%s"
					}
					
					data "scaleway_baremetal_option" "remote_access" {
					  zone = "%s"
					  name = "Remote Access"
					}
					
					resource "scaleway_iam_ssh_key" "base" {
					  name       = "%s"
					  public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
					  name        = "%s"
					  zone        = "%s"
					  offer       = data.scaleway_baremetal_offer.my_offer.offer_id
					  os          = data.scaleway_baremetal_os.my_os.os_id
					  ssh_key_ids = [scaleway_iam_ssh_key.base.id]
					
					  options {
						id         = data.scaleway_baremetal_option.remote_access.option_id
						expires_at = "2026-07-06T09:00:00Z"
					  }
					}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasOptions(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.base", "options.0.id", "data.scaleway_baremetal_option.remote_access", "option_id"),
					resource.TestCheckTypeSetElemNestedAttrs("scaleway_baremetal_server.base", "options.*", map[string]string{
						"id":         Zone + "/931df052-d713-4674-8b58-96a63244c8e2",
						"expires_at": "2026-07-06T09:00:00Z",
					}),
				),
			},
		},
	})
}

func TestAccServer_CreateServerWithPrivateNetwork(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccScalewayBaremetalServer_CreateServerWithPrivateNetwork"
	name := "TestAccScalewayBaremetalServer_CreateServerWithPrivateNetwork"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			baremetalchecks.CheckServerDestroy(tt),
		),
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

					data "scaleway_baremetal_option" "private_network" {
						zone = "%s"
						name = "Private Network"
					}

					resource "scaleway_vpc_private_network" "pn" {
						name = "baremetal_private_network"
					} 

					resource "scaleway_iam_ssh_key" "base" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						offer       = data.scaleway_baremetal_offer.my_offer.offer_id
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						ssh_key_ids = [ scaleway_iam_ssh_key.base.id ]
						options {
						  id = data.scaleway_baremetal_option.private_network.option_id
						}
						private_network {
						  id = scaleway_vpc_private_network.pn.id
						}
					}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasPrivateNetwork(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.base", "private_network.0.id", "scaleway_vpc_private_network.pn", "id"),
					resource.TestCheckResourceAttrSet("scaleway_baremetal_server.base", "private_ips.0.id"),
					resource.TestCheckResourceAttrSet("scaleway_baremetal_server.base", "private_ips.0.address"),
				),
			},
		},
	})
}

func TestAccServer_AddPrivateNetwork(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccScalewayBaremetalServer_AddPrivateNetwork"
	name := "TestAccScalewayBaremetalServer_AddPrivateNetwork"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			baremetalchecks.CheckServerDestroy(tt),
		),
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

					data "scaleway_baremetal_option" "private_network" {
						zone = "%s"
						name = "Private Network"
					}

					resource "scaleway_vpc_private_network" "pn" {
						name = "baremetal_private_network"
					} 

					resource "scaleway_iam_ssh_key" "base" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						offer       = data.scaleway_baremetal_offer.my_offer.offer_id
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						ssh_key_ids = [ scaleway_iam_ssh_key.base.id ]
						options {
						  id = data.scaleway_baremetal_option.private_network.option_id
						}
					}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
				),
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

					data "scaleway_baremetal_option" "private_network" {
						zone = "%s"
						name = "Private Network"
					}

					resource "scaleway_vpc_private_network" "pn" {
						name = "baremetal_private_network"
					} 

					resource "scaleway_iam_ssh_key" "base" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						offer       = data.scaleway_baremetal_offer.my_offer.offer_id
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						ssh_key_ids = [ scaleway_iam_ssh_key.base.id ]
						options {
						  id = data.scaleway_baremetal_option.private_network.option_id
						}
						private_network {
						  id = scaleway_vpc_private_network.pn.id
						}
					}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasPrivateNetwork(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.base", "private_network.0.id", "scaleway_vpc_private_network.pn", "id"),
				),
			},
		},
	})
}

func TestAccServer_AddAnotherPrivateNetwork(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccScalewayBaremetalServer_AddAnotherPrivateNetwork"
	name := "TestAccScalewayBaremetalServer_AddAnotherPrivateNetwork"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			baremetalchecks.CheckServerDestroy(tt),
		),
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

					data "scaleway_baremetal_option" "private_network" {
						zone = "%s"
						name = "Private Network"
					}

					resource "scaleway_vpc_private_network" "pn" {
						name = "baremetal_private_network"
					}

					resource "scaleway_iam_ssh_key" "base" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						offer       = data.scaleway_baremetal_offer.my_offer.offer_id
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						ssh_key_ids = [ scaleway_iam_ssh_key.base.id ]
						options {
						  id = data.scaleway_baremetal_option.private_network.option_id
						}
						private_network {
						  id = scaleway_vpc_private_network.pn.id
						}
					}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasPrivateNetwork(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.base", "private_network.0.id", "scaleway_vpc_private_network.pn", "id"),
				),
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

					data "scaleway_baremetal_option" "private_network" {
						zone = "%s"
						name = "Private Network"
					}

					resource "scaleway_vpc_private_network" "pn" {
						name = "baremetal_private_network"
					} 

					resource "scaleway_vpc_private_network" "pn2" {
						name = "baremetal_private_network2"
					} 

					resource "scaleway_iam_ssh_key" "base" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						offer       = data.scaleway_baremetal_offer.my_offer.offer_id
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						ssh_key_ids = [ scaleway_iam_ssh_key.base.id ]
						options {
						  id = data.scaleway_baremetal_option.private_network.option_id
						}
						private_network {
						  id = scaleway_vpc_private_network.pn.id
						}
						private_network {
						  id = scaleway_vpc_private_network.pn2.id
						}
					}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasPrivateNetwork(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckTypeSetElemAttrPair("scaleway_baremetal_server.base", "private_network.*.id", "scaleway_vpc_private_network.pn", "id"),
					resource.TestCheckTypeSetElemAttrPair("scaleway_baremetal_server.base", "private_network.*.id", "scaleway_vpc_private_network.pn2", "id"),
				),
			},
		},
	})
}

func TestAccServer_WithIPAMPrivateNetwork(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	SSHKeyName := "TestAccScalewayBaremetalServer_WithIPAMPrivateNetwork"
	name := "TestAccScalewayBaremetalServer_WithIPAMPrivateNetwork"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			baremetalchecks.CheckServerDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_vpc" "vpc01" {
					  name = "TestAccScalewayBaremetalIPAM"
					}
					
					resource "scaleway_vpc_private_network" "pn01" {
					  name = "TestAccScalewayBaremetalIPAM"
					  ipv4_subnet {
						subnet = "172.16.64.0/22"
					  }
					  vpc_id = scaleway_vpc.vpc01.id
					}
					
					resource "scaleway_ipam_ip" "ip01" {
					  address = "172.16.64.7"
					  source {
						private_network_id = scaleway_vpc_private_network.pn01.id
					  }
					}

					resource "scaleway_ipam_ip" "ip02" {
					  address = "172.16.64.9"
					  source {
						private_network_id = scaleway_vpc_private_network.pn01.id
					  }
					}

					data "scaleway_baremetal_os" "my_os" {
						zone = "%s"
						name = "Ubuntu"
						version = "22.04 LTS (Jammy Jellyfish)"						
					}

					data "scaleway_baremetal_offer" "my_offer" {
						zone = "%s"
						name = "%s"
					}

					data "scaleway_baremetal_option" "private_network" {
						zone = "%s"
						name = "Private Network"
					}

					resource "scaleway_iam_ssh_key" "base" {
						name 	   = "%s"
						public_key = "%s"
					}
					
					resource "scaleway_baremetal_server" "base" {
						name        = "%s"
						zone        = "%s"
						offer       = data.scaleway_baremetal_offer.my_offer.offer_id
						os          = data.scaleway_baremetal_os.my_os.os_id
					
						ssh_key_ids = [ scaleway_iam_ssh_key.base.id ]
						options {
						  id = data.scaleway_baremetal_option.private_network.option_id
						}
						private_network {
						  id = scaleway_vpc_private_network.pn01.id
						  ipam_ip_ids = [scaleway_ipam_ip.ip01.id]
						}
					}

					data "scaleway_ipam_ip" "base" {
					  resource {
						name = scaleway_baremetal_server.base.name
						type = "baremetal_private_nic"
					  }
					  type = "ipv4"
					}
				`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasPrivateNetwork(tt, "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_ipam_ip.ip01", "address", "data.scaleway_ipam_ip.base", "address_cidr"),
				),
			},
			{
				Config: fmt.Sprintf(`
						resource "scaleway_vpc" "vpc01" {
						  name = "TestAccScalewayBaremetalIPAM"
						}
			
						resource "scaleway_vpc_private_network" "pn01" {
						  name = "TestAccScalewayBaremetalIPAM"
						  ipv4_subnet {
							subnet = "172.16.64.0/22"
						  }
						  vpc_id = scaleway_vpc.vpc01.id
						}
			
						resource "scaleway_ipam_ip" "ip01" {
						  address = "172.16.64.7"
						  source {
							private_network_id = scaleway_vpc_private_network.pn01.id
						  }
						}
			
						resource "scaleway_ipam_ip" "ip02" {
						  address = "172.16.64.9"
						  source {
							private_network_id = scaleway_vpc_private_network.pn01.id
						  }
						}
			
						data "scaleway_baremetal_os" "my_os" {
							zone = "%s"
							name = "Ubuntu"
							version = "22.04 LTS (Jammy Jellyfish)"
						}
			
						data "scaleway_baremetal_offer" "my_offer" {
							zone = "%s"
							name = "%s"
						}
			
						data "scaleway_baremetal_option" "private_network" {
							zone = "%s"
							name = "Private Network"
						}
			
						resource "scaleway_iam_ssh_key" "base" {
							name 	   = "%s"
							public_key = "%s"
						}
			
						resource "scaleway_baremetal_server" "base" {
							name        = "%s"
							zone        = "%s"
							offer       = data.scaleway_baremetal_offer.my_offer.offer_id
							os          = data.scaleway_baremetal_os.my_os.os_id
			
							ssh_key_ids = [ scaleway_iam_ssh_key.base.id ]
							options {
							  id = data.scaleway_baremetal_option.private_network.option_id
							}
							private_network {
							  id = scaleway_vpc_private_network.pn01.id
							  ipam_ip_ids = [scaleway_ipam_ip.ip01.id, scaleway_ipam_ip.ip02.id]
							}
						}
			
						data "scaleway_ipam_ips" "base" {
						  resource {
							name = scaleway_baremetal_server.base.name
							type = "baremetal_private_nic"
						  }
						  type = "ipv4"
						}
					`, Zone, Zone, OfferName, Zone, SSHKeyName, SSHKeyBaremetal, name, Zone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaremetalServerExists(tt, "scaleway_baremetal_server.base"),
					testAccCheckBaremetalServerHasPrivateNetwork(tt, "scaleway_baremetal_server.base"),
					testIPAMIPs(tt, "scaleway_ipam_ip", "data.scaleway_ipam_ips.base"),
				),
			},
		},
	})
}

func TestAccServer_UpdateSubscriptionPeriod(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	newOffer := "EM-B320E-NVME"

	if !IsOfferAvailable(OfferName, scw.Zone(Zone), tt) {
		t.Skip("Offer is out of stock")
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			baremetalchecks.CheckServerDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_offer" "my_offer" {
						zone = "%s"
						name = "%s"
						subscription_period = "hourly"
					}

					resource "scaleway_baremetal_server" "server01" {
						name = "TestAccServer_UpdateSubscriptionPeriod"
						offer = data.scaleway_baremetal_offer.my_offer.offer_id
						zone = "%s"
						install_config_afterward = true
					}`, Zone, OfferName, Zone),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_baremetal_server.server01", "zone", Zone),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.server01", "offer_id", "data.scaleway_baremetal_offer.my_offer", "offer_id"),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_offer" "my_offer" {
						zone = "%s"
						name = "%s"
						subscription_period = "monthly"
					}

					resource "scaleway_baremetal_server" "server01" {
						name = "TestAccServer_UpdateSubscriptionPeriod"
						offer = data.scaleway_baremetal_offer.my_offer.offer_id
						zone = "%s"
						install_config_afterward = true
					}`, Zone, OfferName, Zone),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_baremetal_server.server01", "zone", Zone),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.server01", "offer_id", "data.scaleway_baremetal_offer.my_offer", "offer_id"),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_offer" "my_offer" {
						zone = "%s"
						name = "%s"
						subscription_period = "hourly"
					}

					resource "scaleway_baremetal_server" "server01" {
						name = "TestAccServer_UpdateSubscriptionPeriod"
						offer = data.scaleway_baremetal_offer.my_offer.offer_id
						zone = "%s"
						install_config_afterward = true
					}`, Zone, OfferName, Zone),
				ExpectError: regexp.MustCompile(`invalid plan transition: you cannot transition from a monthly plan to an hourly plan. Only the reverse \(hourly to monthly\) is supported. Please update your configuration accordingly`),
			},
			{
				Config: fmt.Sprintf(`
					data "scaleway_baremetal_offer" "my_offer" {
						zone = "%s"
						name = "%s"
						subscription_period = "hourly"
					}

					resource "scaleway_baremetal_server" "server01" {
						name = "Test_UpdateSubscriptionPeriod"
						offer = data.scaleway_baremetal_offer.my_offer.offer_id
						zone = "%s"
						install_config_afterward = true
					}`, Zone, newOffer, Zone),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_baremetal_server.server01", "zone", Zone),
					resource.TestCheckResourceAttrPair("scaleway_baremetal_server.server01", "offer_id", "data.scaleway_baremetal_offer.my_offer", "offer_id"),
				),
			},
		},
	})
}

func testAccCheckBaremetalServerExists(tt *acctest.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		baremetalAPI, zonedID, err := baremetal.NewAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = baremetalAPI.GetServer(&baremetalSDK.GetServerRequest{
			ServerID: zonedID.ID,
			Zone:     zonedID.Zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckPartitioning(tt *acctest.TestTools, n string, source string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		baremetalAPI, zonedID, err := baremetal.NewAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		server, err := baremetalAPI.GetServer(&baremetalSDK.GetServerRequest{
			ServerID: zonedID.ID,
			Zone:     zonedID.Zone,
		})
		if err != nil {
			return err
		}

		if server.Install.PartitioningSchema == nil {
			return fmt.Errorf("server %s has no partitioning schema", n)
		}

		schema := baremetalSDK.Schema{}

		err = json.Unmarshal([]byte(source), &schema)
		if err != nil {
			return err
		}

		if !reflect.DeepEqual(&schema, server.Install.PartitioningSchema) {
			return fmt.Errorf("server %s has not custom partitioning install", n)
		}

		return nil
	}
}

func testAccCheckBaremetalServerHasOptions(tt *acctest.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		baremetalAPI, zonedID, err := baremetal.NewAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		server, err := baremetalAPI.GetServer(&baremetalSDK.GetServerRequest{
			ServerID: zonedID.ID,
			Zone:     zonedID.Zone,
		})
		if err != nil {
			return err
		}

		if len(server.Options) == 0 {
			return fmt.Errorf("server (%s) has no options enabled", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckBaremetalServerHasPrivateNetwork(tt *acctest.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		_, zonedID, err := baremetal.NewAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		baremetalPrivateNetworkAPI, _, err := baremetal.NewPrivateNetworkAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		listPrivateNetworks, err := baremetalPrivateNetworkAPI.ListServerPrivateNetworks(&baremetalV3SDK.PrivateNetworkAPIListServerPrivateNetworksRequest{
			Zone:     zonedID.Zone,
			ServerID: &zonedID.ID,
		})
		if err != nil {
			return err
		}

		if len(listPrivateNetworks.ServerPrivateNetworks) == 0 {
			return fmt.Errorf("server (%s) has no private networks attached to it", rs.Primary.ID)
		}

		return nil
	}
}

func testIPAMIPs(_ *acctest.TestTools, ipamResourcePrefix, ipamDataSource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ipamData, ok := s.RootModule().Resources[ipamDataSource]
		if !ok {
			return fmt.Errorf("not found: %s", ipamDataSource)
		}

		ips := ipamData.Primary.Attributes
		expectedIPs := make(map[string]bool)

		for i := 0; ; i++ {
			key := fmt.Sprintf("ips.%d.address", i)

			ip, ok := ips[key]
			if !ok {
				break
			}

			expectedIPs[ip] = true
		}

		for y := 1; ; y++ {
			resourceName := fmt.Sprintf("%s.ip0%d", ipamResourcePrefix, y)

			rs, ok := s.RootModule().Resources[resourceName]
			if !ok {
				break
			}

			ip := rs.Primary.Attributes["address"]
			if !expectedIPs[ip] {
				return fmt.Errorf("IP %q from resource %s not found in data source %s", ip, resourceName, ipamDataSource)
			}
		}

		return nil
	}
}
