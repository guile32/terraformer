// Copyright 2018 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package keycloak

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/terraformer/terraform_utils"
	"github.com/Trois-Six/terraform-provider-keycloak/keycloak"
)

type OpenIDClientGenerator struct {
	KeycloakService
}

var OpenIDClientAllowEmptyValues = []string{"web_origins"}
var OpenIDClientAdditionalFields = map[string]interface{}{}

func (g OpenIDClientGenerator) createResources(openIDClients []*keycloak.OpenidClient) []terraform_utils.Resource {
	var resources []terraform_utils.Resource
	for _, openIDClient := range openIDClients {
		resources = append(resources, terraform_utils.NewResource(
			openIDClient.Id,
			"openid_client_"+normalizeResourceName(openIDClient.RealmId)+"_"+normalizeResourceName(openIDClient.ClientId),
			"keycloak_openid_client",
			"keycloak",
			map[string]string{
				"realm_id": openIDClient.RealmId,
			},
			OpenIDClientAllowEmptyValues,
			OpenIDClientAdditionalFields,
		))
	}
	return resources
}

func (g OpenIDClientGenerator) createServiceAccountClientRolesResources(realmId string, serviceAccountClient []string, serviceAccountUserId string, client *keycloak.OpenidClient, roles []*keycloak.OpenidClientServiceAccountRole) []terraform_utils.Resource {
	var resources []terraform_utils.Resource
	for _, role := range roles {
		resources = append(resources, terraform_utils.NewResource(
			realmId+"/"+serviceAccountUserId+"/"+serviceAccountClient[1]+"/"+role.Id,
			"openid_client_service_account_role_"+normalizeResourceName(realmId)+"_"+normalizeResourceName(serviceAccountClient[1])+"_"+normalizeResourceName(client.ClientId)+"_"+normalizeResourceName(role.Name),
			"keycloak_openid_client_service_account_role",
			"keycloak",
			map[string]string{
				"realm_id":                realmId,
				"service_account_user_id": serviceAccountUserId,
				"client_id":               client.Id,
				"role":                    role.Name,
			},
			OpenIDClientAllowEmptyValues,
			OpenIDClientAdditionalFields,
		))
	}
	return resources
}

func (g *OpenIDClientGenerator) InitResources() error {
	var openIDClientsFull []*keycloak.OpenidClient
	client, err := keycloak.NewKeycloakClient(g.Args["url"].(string), g.Args["client_id"].(string), g.Args["client_secret"].(string), g.Args["realm"].(string), "", "", true, 5)
	if err != nil {
		return err
	}
	realms, err := client.GetRealms()
	if err != nil {
		return err
	}
	for _, realm := range realms {
		openIDClients, err := client.GetOpenidClients(realm.Id, true)
		if err != nil {
			return err
		}
		mapServiceAccountIds := map[string][]string{}
		for _, openIDClient := range openIDClients {
			if !openIDClient.ServiceAccountsEnabled {
				continue
			}
			serviceAccountUser, err := client.GetOpenidClientServiceAccountUserId(realm.Id, openIDClient.Id)
			if err != nil {
				return err
			}

			fmt.Printf("serviceAccountUser: %+v\n", serviceAccountUser)
			mapServiceAccountIds[serviceAccountUser.Id] = []string{openIDClient.Id, openIDClient.ClientId}

			/*
			openidClientWithProcolMappers, err := client.GetGenericClientProtocolMappers(realm.Id, openIDClient.Id)
			for _, genericClientProtocolMapper := range openidClientWithProcolMappers.ProtocolMappers {
				fmt.Printf("genericClientProtocolMapper: %+v\n", genericClientProtocolMapper.ProtocolMapper)
			}
			*/
		}
		openIDClientsFull = append(openIDClientsFull, openIDClients...)

		/*
		clientRoles, err := client.GetClientRoles(realm.Id, openIDClients)
		if err != nil {
			return err
		}
		fmt.Printf("clientRoles: %+v\n", clientRoles)

		usersInRole, err := client.GetClientRoleUsers(realm.Id, clientRoles)
		if err != nil {
			return err
		}
		fmt.Printf("usersInRole: %+v\n", usersInRole)
		*/

	}
	g.Resources = append(g.Resources, g.createResources(openIDClientsFull)...)
	return nil
}

func (g *OpenIDClientGenerator) PostConvertHook() error {
	mapClientIDs := map[string]string{}
	mapServiceAccountUserIDs := map[string]string{}
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "keycloak_openid_client" {
			continue
		}
		mapClientIDs[r.InstanceState.ID] = "${" + r.InstanceInfo.Type + "." + r.ResourceName + ".id}"
		if _, exist := r.InstanceState.Attributes["service_account_user_id"]; exist {
			mapServiceAccountUserIDs[r.InstanceState.Attributes["service_account_user_id"]] = "${" + r.InstanceInfo.Type + "." + r.ResourceName + ".service_account_user_id}"
		}
	}
	for i, r := range g.Resources {
		if r.InstanceInfo.Type != "keycloak_openid_client" && r.InstanceInfo.Type != "keycloak_openid_client_service_account_role" {
			continue
		}
		if _, exist := r.Item["client_id"]; exist && r.InstanceInfo.Type == "keycloak_openid_client_service_account_role" {
			r.Item["client_id"] = mapClientIDs[r.Item["client_id"].(string)]
		}
		if _, exist := r.Item["service_account_user_id"]; exist && r.InstanceInfo.Type == "keycloak_openid_client_service_account_role" {
			r.Item["service_account_user_id"] = mapServiceAccountUserIDs[r.Item["service_account_user_id"].(string)]
		}
		if strings.Contains(r.InstanceState.Attributes["name"], "$") {
			g.Resources[i].Item["name"] = strings.ReplaceAll(r.InstanceState.Attributes["name"], "$", "$$")
		}
	}
	return nil
}
