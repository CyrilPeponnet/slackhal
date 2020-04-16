package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

var help string = `
*RBAC management*

By default it's not set, to start use it type the *behave* command.
This will give your the rbac role with permissions on everything

Then you can manage the RBAC as follow:

- *rbac-list*: To list the current RBAC
- *rbac-add-role <name> <description> withParent:<parent1>,<parent2>*: To add a new role with optionnals parents
- *rbac-del-role <name>*: To delete a role
- *rbac-add-permission <name> <description>*: To add a permission
- *rbac-del-permission <name>*: To delete a permission
- *rbac-attach-permission <p1,p2> to <role1,role2>*: To attach permission on a role
- *rbac-dettach-permission <p1,p2> from <role1,role2>*: To detach permssion from a role
- *rbac-bind <kind> <value> to <role1,role2>*: To bind an identity to a role
- *rbac-unbind <value> from <role1,role2>* : To unbind an identity from a role
- *rbac-dump*: Will dump current rbac as a json file
- *rbac-load*: Will load the given json base64 encoded blob

A permission is a trigger (for instance *cat*).

When binding identity to role:
- <kind> can be either user, channel, memberOf or a slack Custom Field Name
- <value> is the value of the user, channel or channel appartenance or slack Custom Field value

Special permission:
- *: mean everything
`

// AuthzHandleChat handle the chat messages
func AuthzHandleChat(msg *slack.MessageEvent) (response string) {

	txt := strings.ToLower(msg.Msg.Text)

	// get user info
	user, err := bot.GetCachedUserInfos(msg.User)
	if err != nil {
		zap.L().Error("Failed to get user info from cache", zap.Error(err))
	}

	switch {
	case strings.HasPrefix(txt, "rbac-help"):
		return help

	case strings.HasPrefix(txt, "rbac-add-role"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			// extract parents first
			line := strings.Replace(msg.Text, "rbac-add-role ", "", 1)
			// get parents
			content := strings.Split(line, "withParent:")
			var parents []string
			if len(content) == 2 {
				parents = strings.Split(strings.TrimSpace(content[1]), ",")
			}

			aRole := strings.SplitN(content[0], " ", 2)

			if len(aRole) != 2 {
				return "Invalid syntax" + help
			}

			err := authz.AddRole(aRole[0], aRole[1], parents...)
			if err != nil {
				return err.Error()
			}
			return "Role " + aRole[0] + " created."
		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-del-role"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			line := strings.TrimSpace(strings.Replace(msg.Text, "rbac-del-role ", "", 1))
			if line == "" {
				return "Please provide a role name."
			}
			err := authz.RemoveRole(line)
			if err != nil {
				return line + " " + err.Error()
			}
			return "Role " + line + " has been removed."
		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-add-permission"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			line := strings.TrimSpace(strings.Replace(msg.Text, "rbac-add-permission ", "", 1))

			aPerm := strings.SplitN(line, " ", 2)

			if len(aPerm) != 2 {
				return "Invalid syntax" + help
			}

			err := authz.AddPermission(aPerm[0], aPerm[1])
			if err != nil {
				return err.Error()
			}
			return "Permission " + aPerm[0] + " created."
		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-del-permission"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {

			line := strings.TrimSpace(strings.Replace(msg.Text, "rbac-del-permission ", "", 1))
			if line == "" {
				return "Please provide a permission name."
			}
			err := authz.RemovePermission(line)
			if err != nil {
				return line + " " + err.Error()
			}
			return "Permission " + line + " has been removed."
		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-attach-permission"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			line := strings.TrimSpace(strings.Replace(msg.Text, "rbac-attach-permission ", "", 1))
			parts := strings.Split(line, " to ")
			if len(parts) != 2 {
				return "Invalid syntax " + help
			}

			perms := strings.Split(parts[0], ",")
			roles := strings.Split(parts[1], ",")

			if len(perms) == 0 || len(roles) == 0 {
				return "Invalid syntax " + help
			}

			for _, r := range roles {
				for _, p := range perms {
					err := authz.AttachPermission(strings.TrimSpace(p), strings.TrimSpace(r))
					if err != nil {
						return "Failed to attach " + p + " to role " + r + " error: " + err.Error()
					}
				}
			}

			return "Permissions " + strings.Join(perms, ",") + " attached to " + strings.Join(roles, ",") + "."

		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-dettach-permission"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			line := strings.TrimSpace(strings.Replace(msg.Text, "rbac-dettach-permission ", "", 1))
			parts := strings.Split(line, " from ")
			if len(parts) != 2 {
				return "Invalid syntax " + help
			}

			perms := strings.Split(parts[0], ",")
			roles := strings.Split(parts[1], ",")

			if len(perms) == 0 || len(roles) == 0 {
				return "Invalid syntax " + help
			}

			for _, r := range roles {
				for _, p := range perms {
					err := authz.DettachPermission(strings.TrimSpace(p), strings.TrimSpace(r))
					if err != nil {
						return "Failed to dettach " + p + " from role " + r
					}
				}
			}

			return "Permissions " + strings.Join(perms, ",") + " dettached from " + strings.Join(roles, ",") + "."

		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-bind"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			line := strings.TrimSpace(strings.Replace(msg.Text, "rbac-bind ", "", 1))

			parts := strings.Split(line, " to ")
			if len(parts) != 2 {
				return "Invalid syntax " + help
			}

			features := strings.SplitN(parts[0], " ", 2)
			roles := strings.Split(parts[1], ",")

			if len(features) != 2 || len(roles) == 0 {
				return "Invalid feature " + help
			}

			kind := strings.TrimSpace(features[0])
			value := strings.TrimSpace(features[1])

			ID := value
			if value != "all" {
				// Extract feature from value
				f := bot.ExtractFeaturesFromMessage(value)
				if len(f) == 0 || len(f) > 1 {
					return "Failed to determine the value for kind " + kind
				}
				ID = f[0].ID
			}

			err := authz.BindToRole(kind, ID, roles...)
			if err != nil {
				return "Failed to bind " + kind + ": " + value + " to " + strings.Join(roles, ",")
			}

			return "Feature " + kind + ": " + value + " binded to " + strings.Join(roles, ",") + "."

		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-unbind"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			line := strings.TrimSpace(strings.Replace(msg.Text, "rbac-unbind ", "", 1))

			parts := strings.Split(line, " from ")
			if len(parts) != 2 {
				return "Invalid syntax " + help
			}

			value := strings.TrimSpace(parts[0])
			roles := strings.Split(parts[1], ",")

			if value == "" || len(roles) == 0 {
				return "Invalid syntax " + help
			}

			// Extract feature from value
			ID := value
			if value != "all" {
				f := bot.ExtractFeaturesFromMessage(value)
				if len(f) == 0 || len(f) > 1 {
					return "Failed to determine the feature of the value"
				}
				ID = f[0].ID
			}

			for _, r := range roles {
				err := authz.UnBindFromRole(ID, r)
				if err != nil {
					return "Failed to unbind " + ": " + value + " from " + r
				}
			}

			return "Feature with value " + value + " unbinded from " + strings.Join(roles, ",") + "."

		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-inspect-indenity"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			line := strings.TrimSpace(strings.Replace(msg.Text, "rbac-inspect-indenity ", "", 1))

			name := strings.TrimSpace(line)
			if name == "" {
				return "Please provide a user name."
			}
			data := ""
			re := regexp.MustCompile(`<@(\S+)>`)
			for _, m := range re.FindAllStringSubmatch(line, -1) {
				user, err := bot.GetCachedUserInfos(m[1])
				if err != nil {
					return "Unable to get user information " + err.Error()
				}
				pjson, err := json.MarshalIndent(user.Profile, "", "    ")
				if err != nil {
					return "Unable to decode profile structure " + err.Error()
				}
				data += string(pjson)
			}

			return data

		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-dump"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {

			data := authz.Dump()
			pjson, err := json.MarshalIndent(data, "", "    ")
			if err != nil {
				return "Failed to marshal current data " + err.Error()
			}
			return string(pjson)

		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "rbac-load"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			line := strings.ReplaceAll(strings.TrimSpace(strings.Replace(msg.Text, "rbac-load ", "", 1)), "\n", "")

			//TODO: this is ugly as hell but that will be enough for now
			// We should use a snipet or a file
			data, err := base64.StdEncoding.DecodeString(line)
			if err != nil {
				return
			}
			err = authz.Load(data)
			if err != nil {
				return "Failed to load data " + err.Error()
			}

			return "Data successfully loaded."

		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	// List the roles permission and bindings
	case strings.HasPrefix(txt, "rbac-list"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {

			tpl := `
  {{ if .Bindings }}
  *Role bindings*:
  {{- range .Bindings}}
  {{- if eq .Kind "user" }}
  - _{{.Kind}}=<@{{.Name}}>_: {{.Roles}}
  {{- end }}
  {{- if eq .Kind "channel" }}
  - _{{.Kind}}=<#{{.Name}}>_: {{.Roles}}
  {{- end }}
  {{- if eq .Kind "memberOf" }}
  - _{{.Kind}}=<#{{.Name}}>_: {{.Roles}}
  {{- end }}
  {{- end}}
  {{- end}}

  {{ if .Roles }}
  *Roles*:
  {{- range .Roles}}
  - _{{.Name}}_: {{.Description}}{{if .Parents}}, Parents:{{.Parents}}{{end}}{{if .Permissions}}, Permissions: {{range .Permissions}}{{.Name}} {{end}}{{end}}
  {{- end}}
  {{- end}}

  {{ if .Permissions }}
  *Permissions*:
  {{- range .Permissions}}
  - {{.Name}}: {{.Description}}
  {{- end}}
  {{- end}}
  `

			data := authz.Dump()
			if data != nil {

				t, err := template.New("output").Parse(tpl)
				if err != nil {
					return ""
				}
				buf := new(bytes.Buffer)
				err = t.Execute(buf, data)
				if err != nil {
					return ""
				}

				return buf.String()
			}
			return "No RBAC set yet, start with behave keyword."
		}
		return fmt.Sprintf("I'm sorry, %s I'm afraid I can't do that.", user.RealName)

	case strings.HasPrefix(txt, "behave"):
		if authz.IsGranted("rbac", msg.User, msg.Channel, "") {
			err := authz.AddPermission("*", "Can do everything")
			if err != nil {
				return fmt.Sprintf("Error while creating the permission: %s", err.Error())
			}
			err = authz.AddPermission("rbac", "Can manage rbac")
			if err != nil {
				return fmt.Sprintf("Error while creating the permission: %s", err.Error())
			}
			err = authz.AddRole("rbac", "RBAC management role")
			if err != nil {
				return fmt.Sprintf("Error while creating the rbac role: %s", err.Error())
			}
			err = authz.AttachPermission("rbac", "rbac")
			if err != nil {
				return fmt.Sprintf("Error while attaching the permission rbac to the rbac role: %s", err.Error())
			}
			err = authz.BindToRole("user", user.ID, "rbac")
			if err != nil {
				return fmt.Sprintf("Error while creating the binding to rbac role: %s", err.Error())
			}
			return "You are the boss now."
		}
	}

	return ""
}
