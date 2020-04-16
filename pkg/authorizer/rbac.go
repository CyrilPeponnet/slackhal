package authorizer

import (
	"fmt"
	"sync"

	"encoding/json"

	"github.com/asdine/storm"
	"github.com/mikespook/gorbac"
	"go.uber.org/zap"
)

type role struct {
	Name        string `storm:"id"`
	Description string
	Permissions []permission
	Parents     []string
}

type rolebinding struct {
	Name  string `storm:"id"`
	Kind  string
	Roles []string
}

type permission struct {
	Name        string `storm:"id"`
	Description string
}

// Authorizer is a authorizer type
type Authorizer struct {
	db   *storm.DB
	rbac *gorbac.RBAC
	lock sync.Mutex
}

// Init initialize the authorizer
func (a *Authorizer) Init(dbPath string) (err error) {

	// Init the db
	if a.db == nil {
		a.db, err = storm.Open(dbPath)
		if err != nil {
			zap.L().Error("Failed to init database", zap.Error(err))
			return err
		}
	}
	return a.load()
}

// IsGranted return if a context is authorized
func (a *Authorizer) IsGranted(permission, user string, fromChannel string, memberOfChannels ...string) bool {

	var myRoles []string

	// Look for bindings
	rolebindings := []rolebinding{}

	if err := a.db.All(&rolebindings); err != nil {
		zap.L().Error("Error while listing the bindings", zap.Error(err))
		return false
	}

	if len(rolebindings) == 0 {
		zap.L().Warn("No RBAC set, allow all")
		return true
	}

	for _, r := range rolebindings {
		switch r.Kind {
		case "memberOf":
			for _, m := range memberOfChannels {
				if r.Name == m {
					myRoles = append(myRoles, r.Roles...)
				}
			}
		case "channel":
			if fromChannel == r.Name {
				myRoles = append(myRoles, r.Roles...)
			}
		case "user":
			if user == r.Name || r.Name == "all" {
				myRoles = append(myRoles, r.Roles...)
			}
		}
	}

	// For each roles find any that are granted
	for _, r := range myRoles {

		// Look for wildcard permission in our role if any
		rl, _, err := a.rbac.Get(r)
		if err != nil {
			zap.L().Error("Cannot retrieve role", zap.String("role", r), zap.Error(err))
		} else {
			if rl.Permit(gorbac.NewStdPermission("*")) {
				zap.L().Debug("Authorization Granted",
					zap.String("user", user),
					zap.String("permission", permission),
					zap.String("channel", fromChannel),
					zap.Strings("roles", myRoles),
					zap.Int("memberOf", len(memberOfChannels)),
					zap.String("granted through", rl.ID()))
				return true
			}
		}

		// otherwise look if our role is granted
		if a.rbac.IsGranted(r, gorbac.NewStdPermission(permission), nil) {
			zap.L().Debug("Authorization Granted",
				zap.String("user", user),
				zap.String("permission", permission),
				zap.String("channel", fromChannel),
				zap.Strings("roles", myRoles),
				zap.Int("memberOf", len(memberOfChannels)),
				zap.String("granted through", r))
			return true
		}
	}

	zap.L().Debug("Authorization Refused",
		zap.String("user", user),
		zap.String("permission", permission),
		zap.String("channel", fromChannel),
		zap.Strings("roles", myRoles),
		zap.Int("memberOf", len(memberOfChannels)))

	return false

}

// load db to rabc
func (a *Authorizer) load() (err error) {

	a.lock.Lock()
	defer a.lock.Unlock()

	// Init the rbac
	// save it if we have one already
	var oldrbac *gorbac.RBAC
	if a.rbac != nil {
		oldrbac = a.rbac
	}

	a.rbac = gorbac.New()

	// Load the roles
	roleList := []role{}
	if err := a.db.All(&roleList); err != nil {
		zap.L().Error("Failed to get role list", zap.Error(err))
		if oldrbac != nil {
			a.rbac = oldrbac
		}
		return err
	}

	// Build roles and add them to goRBAC instance
	for _, r := range roleList {
		grole := gorbac.NewStdRole(r.Name)
		for _, p := range r.Permissions {
			err := grole.Assign(gorbac.NewStdPermission(p.Name))
			if err != nil {
				return err
			}
		}
		err := a.rbac.Add(grole)
		if err != nil {
			return err
		}
	}

	// Assign the inheritance relationship
	for _, r := range roleList {
		for _, p := range r.Parents {
			if p != "" {
				if err := a.rbac.SetParent(r.Name, p); err != nil {
					if oldrbac != nil {
						a.rbac = oldrbac
					}
					return fmt.Errorf("Failed to set Role %s as parent (%s)", p, err.Error())
				}
			}
		}
	}
	return err
}

// BindToRole bind an identity to a role
func (a *Authorizer) BindToRole(kind string, name string, roles ...string) (err error) {
	return a.db.Save(&rolebinding{
		Kind:  kind,
		Name:  name,
		Roles: roles,
	})
}

// UnBindFromRole unbind an identity from a role
func (a *Authorizer) UnBindFromRole(name string, role string) (err error) {

	var r rolebinding
	err = a.db.One("Name", name, &r)
	if err != nil {
		return err
	}

	// Remove the role
	for i, rl := range r.Roles {
		if rl == role {
			r.Roles[i] = r.Roles[len(r.Roles)-1]
			r.Roles[len(r.Roles)-1] = ""
			r.Roles = r.Roles[:len(r.Roles)-1]
			break
		}
	}

	if len(r.Roles) == 0 {
		err = a.db.DeleteStruct(&r)
		if err != nil {
			zap.L().Error("Failed to delete rolebinding", zap.Error(err))
			return err
		}
	} else {
		err = a.db.Save(&r)
		if err != nil {
			zap.L().Error("Failed to update rolebinding", zap.Error(err))
			return err
		}
	}

	return err
}

// AddPermission add a permission
func (a *Authorizer) AddPermission(name string, description string) error {

	p := permission{
		Name:        name,
		Description: description,
	}

	err := a.db.Save(&p)
	if err != nil {
		return err
	}

	return a.load()

}

// RemovePermission remove a permission
func (a *Authorizer) RemovePermission(name string) error {

	p := permission{}

	err := a.db.One("Name", name, &p)
	if err != nil {
		return err
	}

	err = a.db.DeleteStruct(&p)
	if err != nil {
		return err
	}

	// Remove permissions from roles
	roleList := []role{}

	if err := a.db.All(&roleList); err != nil {
		zap.L().Error("Failed to get role list", zap.Error(err))
		return err
	}

	for _, r := range roleList {
		for i, p := range r.Permissions {
			if p.Name == name {
				r.Permissions[i] = r.Permissions[len(r.Permissions)-1]
				r.Permissions[len(r.Permissions)-1] = permission{}
				r.Permissions = r.Permissions[:len(r.Permissions)-1]
				break
			}
		}
	}

	for _, r := range roleList {
		err = a.db.Save(&r)
		if err != nil {
			zap.L().Error("Failed to update role", zap.Error(err))
			return err
		}
	}

	return a.load()
}

// AttachPermission add a permission to a role
func (a *Authorizer) AttachPermission(permName string, roleName string) error {

	// Make sure permission exist

	p := permission{}

	err := a.db.One("Name", permName, &p)
	if err != nil {
		return err
	}

	r := role{}
	err = a.db.One("Name", roleName, &r)
	if err != nil {
		return err
	}

	// Look if it's already there
	for _, cp := range r.Permissions {
		if cp.Name == permName {
			return nil
		}
	}

	// Add it
	r.Permissions = append(r.Permissions, p)

	// Save
	err = a.db.Save(&r)
	if err != nil {
		return err
	}

	return a.load()

}

// DettachPermission detach a permission from a role
func (a *Authorizer) DettachPermission(permName string, roleName string) error {

	// Make sure permission exist
	p := permission{}

	err := a.db.One("Name", permName, &p)
	if err != nil {
		return err
	}

	r := role{}
	err = a.db.One("Name", roleName, &r)
	if err != nil {
		return err
	}

	// Look if it's already there
	for i, cp := range r.Permissions {
		if cp.Name == permName {
			r.Permissions[i] = r.Permissions[len(r.Permissions)-1]
			r.Permissions[len(r.Permissions)-1] = permission{}
			r.Permissions = r.Permissions[:len(r.Permissions)-1]
			break
		}
	}

	// Save
	err = a.db.Save(&r)
	if err != nil {
		return err
	}

	return a.load()

}

// AddRole add a new role
func (a *Authorizer) AddRole(name string, description string, parents ...string) error {

	// Make sure parents exists first
	if len(parents) > 0 {
		for _, p := range parents {
			_r := role{}
			if err := a.db.One("Name", p, &_r); err != nil {
				return fmt.Errorf("Parent %s doesnt exist", p)
			}
		}
	}

	// Look for a role that exists
	var r role
	if err := a.db.One("Name", name, &r); err == nil {
		// Update fields
		r.Description = description
		r.Parents = parents
	} else {
		// New role
		r = role{
			Name:        name,
			Description: description,
			Permissions: []permission{},
			Parents:     parents,
		}
	}

	if err := a.db.Save(&r); err != nil {
		return err
	}

	return a.load()
}

// RemoveRole remove a role
func (a *Authorizer) RemoveRole(name string) (err error) {

	var r role
	err = a.db.One("Name", name, &r)
	if err != nil {
		return err
	}

	err = a.db.DeleteStruct(&r)
	if err != nil {
		return err
	}

	// Remove role from parent
	roleList := []role{}

	if err := a.db.All(&roleList); err != nil {
		zap.L().Error("Failed to get role list", zap.Error(err))
		return err
	}

	for _, r := range roleList {
		for i, p := range r.Parents {
			if p == name {
				r.Parents[i] = r.Parents[len(r.Parents)-1]
				r.Parents[len(r.Parents)-1] = ""
				r.Parents = r.Parents[:len(r.Parents)-1]
				break
			}
		}
	}

	for _, r := range roleList {
		err = a.db.Save(&r)
		if err != nil {
			zap.L().Error("Failed to update role", zap.Error(err))
			return err
		}
	}

	return a.load()

}

// Dump will dump current data
func (a *Authorizer) Dump() interface{} {

	roleList := []role{}
	if err := a.db.All(&roleList); err != nil {
		return ""
	}

	// List binding
	rolebindings := []rolebinding{}
	if err := a.db.All(&rolebindings); err != nil {
		return ""
	}

	// List permissions
	permissions := []permission{}
	if err := a.db.All(&permissions); err != nil {
		return ""
	}

	data := struct {
		Roles       []role
		Bindings    []rolebinding
		Permissions []permission
	}{
		roleList,
		rolebindings,
		permissions,
	}

	if len(roleList) == 0 && len(rolebindings) == 0 && len(permissions) == 0 {
		return nil
	}

	return data

}

// Load will Load current data
func (a *Authorizer) Load(raw []byte) error {

	data := struct {
		Roles       []role
		Bindings    []rolebinding
		Permissions []permission
	}{}

	err := json.Unmarshal(raw, &data)
	if err != nil {
		return err
	}

	for _, i := range data.Bindings {
		if err := a.db.Save(&i); err != nil {
			return err
		}
	}

	for _, i := range data.Roles {
		if err := a.db.Save(&i); err != nil {
			return err
		}
	}

	for _, i := range data.Permissions {
		if err := a.db.Save(&i); err != nil {
			return err
		}
	}

	return a.load()

}
