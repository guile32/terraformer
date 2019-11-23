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
	"github.com/GoogleCloudPlatform/terraformer/terraform_utils"
	"github.com/Trois-Six/terraform-provider-keycloak/keycloak"
)

type RealmGenerator struct {
	KeycloakService
}

var RealmAllowEmptyValues = []string{}

var RequiredActionAllowEmptyValues = []string{}
var RequiredActionAdditionalFields = map[string]interface{}{}

func (g RealmGenerator) createResources(realms []*keycloak.Realm) []terraform_utils.Resource {
	var resources []terraform_utils.Resource
	for _, realm := range realms {
		resources = append(resources, terraform_utils.NewSimpleResource(
			realm.Id,
			"realm_"+normalizeResourceName(realm.Realm),
			"keycloak_realm",
			"keycloak",
			RealmAllowEmptyValues,
		))
	}
	return resources
}

func (g RealmGenerator) createRequiredActionResources(requiredActions []*keycloak.RequiredAction) []terraform_utils.Resource {
	var resources []terraform_utils.Resource
	for _, requiredAction := range requiredActions {
		resources = append(resources, terraform_utils.NewResource(
			requiredAction.RealmId+"/"+requiredAction.Alias,
			"required_action_"+normalizeResourceName(requiredAction.RealmId)+"_"+normalizeResourceName(requiredAction.Alias),
			"keycloak_required_action",
			"keycloak",
			map[string]string{
				"realm_id": requiredAction.RealmId,
				"alias":    requiredAction.Alias,
			},
			RequiredActionAllowEmptyValues,
			RequiredActionAdditionalFields,
		))
	}
	return resources
}

func (g *RealmGenerator) InitResources() error {
	client, err := keycloak.NewKeycloakClient(g.Args["url"].(string), g.Args["client_id"].(string), g.Args["client_secret"].(string), g.Args["realm"].(string), "", "", true, 5)
	if err != nil {
		return err
	}
	realms, err := client.GetRealms()
	if err != nil {
		return err
	}
	g.Resources = g.createResources(realms)
	for _, realm := range realms {
		requiredActions, err := client.GetRequiredActions(realm.Id)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createRequiredActionResources(requiredActions)...)
	}
	return nil
}

func (g *RealmGenerator) PostConvertHook() error {
	mapRealmIDs := map[string]string{}
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "keycloak_realm" {
			continue
		}
		mapRealmIDs[r.InstanceState.ID] = "${" + r.InstanceInfo.Type + "." + r.ResourceName + ".id}"
	}
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "keycloak_required_action" {
			continue
		}
		r.Item["realm_id"] = mapRealmIDs[r.Item["realm_id"].(string)]
	}
	return nil
}
