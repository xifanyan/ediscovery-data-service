package service

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/xifanyan/adp"
	"github.com/xuri/excelize/v2"
)

func getUsers(rows [][]string) []adp.UserDefinition {
	var users []adp.UserDefinition

	for _, row := range rows {
		if row[0] == "UserName" {
			continue
		}

		var externalUserFalg bool = false
		if strings.ToLower(row[2]) == "true" {
			externalUserFalg = true
		}
		users = append(users, adp.UserDefinition{
			Enabled:      true,
			UserName:     row[0],
			Password:     row[1],
			ExternalUser: externalUserFalg,
		})
	}

	return users
}

func getGroups(rows [][]string) []adp.GroupDefinition {
	var groups []adp.GroupDefinition

	for _, row := range rows {
		//  skip the first row
		if row[0] == "GroupName" {
			continue
		}

		groups = append(groups, adp.GroupDefinition{
			GroupName: row[0],
			Enabled:   true,
		})
	}

	return groups
}

func getUserToGroup(rows [][]string) []adp.UserToGroup {
	var userToGroup []adp.UserToGroup

	for _, row := range rows {
		//  skip the first row
		if (row[0] == "GroupName") && (row[1] == "UserName") {
			continue
		}

		userToGroup = append(userToGroup, adp.UserToGroup{
			Enabled:   true,
			GroupName: row[0],
			UserName:  row[1],
		})
	}
	return userToGroup
}

func getApplicationRoles(rows [][]string) []adp.ApplicationRoles {
	var applicationRoles []adp.ApplicationRoles

	for _, row := range rows {
		//  skip the first row
		if (row[0] == "GroupOrUserName") && (row[1] == "Application identifier") {
			continue
		}

		applicationRoles = append(applicationRoles, adp.ApplicationRoles{
			Enabled:               true,
			GroupOrUserName:       row[0],
			ApplicationIdentifier: row[1],
			Roles:                 "Standard User",
		})
	}
	return applicationRoles
}

type UserGroupInput struct {
	Users            []adp.UserDefinition
	Groups           []adp.GroupDefinition
	UserToGroups     []adp.UserToGroup
	ApplicationRoles []adp.ApplicationRoles
}

// GetUsers reads user data from an Excel file specified by the given filename.
// It expects the file to have a sheet named "Data" where each row represents
// a user with columns for UserName, Password, and a boolean flag indicating
// if the user is external. The first row is assumed to be a header and is skipped.
// It returns a slice of adp.UserDefinition and any error encountered during
// file reading or processing.
func GetUsersGroupsRoles(fn string) (*UserGroupInput, error) {
	var err error

	f, err := excelize.OpenFile(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// users
	rows, err := f.GetRows("Users")
	if err != nil {
		return nil, err
	}

	users := getUsers(rows)
	log.Debug().Msgf("users: %+v", users)

	// groups
	rows, err = f.GetRows("Groups")
	if err != nil {
		return nil, err
	}

	groups := getGroups(rows)
	log.Debug().Msgf("groups: %+v", groups)

	// userToGroup
	rows, err = f.GetRows("UserToGroup")
	if err != nil {
		return nil, err
	}

	userToGroup := getUserToGroup(rows)
	log.Debug().Msgf("userToGroup: %+v", userToGroup)

	// applicationRoles
	rows, err = f.GetRows("ApplicationRoles")
	if err != nil {
		return nil, err
	}

	applicationRoles := getApplicationRoles(rows)
	log.Debug().Msgf("applicationRoles: %+v", applicationRoles)

	return &UserGroupInput{
		Users:            users,
		Groups:           groups,
		UserToGroups:     userToGroup,
		ApplicationRoles: applicationRoles,
	}, nil
}

func VerifyApplications(documentHolds []adp.Entity, applicationRoles []adp.ApplicationRoles) error {
	var allowedApplications map[string]struct{} = make(map[string]struct{})

	for _, role := range applicationRoles {
		allowedApplications[strings.TrimSpace(role.ApplicationIdentifier)] = struct{}{} // role.ApplicationIdentifier
	}

	for _, documentHold := range documentHolds {
		if _, ok := allowedApplications[documentHold.ID]; !ok {
			return fmt.Errorf("access to application %s is not allowed", documentHold.ID)
		}
	}

	return nil
}

func SetupManageUsersAndGroupsOptions(input *UserGroupInput) []func(*adp.ManageUsersAndGruopsConfiguration) {
	var opts []func(*adp.ManageUsersAndGruopsConfiguration) = []func(*adp.ManageUsersAndGruopsConfiguration){}

	if len(input.Users) > 0 {
		opts = append(opts, adp.WithManageUsersAndGroupsUserDefinition(input.Users))
	}
	if len(input.Groups) > 0 {
		opts = append(opts, adp.WithManageUsersAndGroupsGroupDefinition(input.Groups))
	}
	if len(input.UserToGroups) > 0 {
		opts = append(opts, adp.WithManageUsersAndGroupsAssignmentUserToGroup(input.UserToGroups))
	}
	if len(input.ApplicationRoles) > 0 {
		opts = append(opts, adp.WithManageUsersAndGroupsAddApplicationRoles(input.ApplicationRoles))
	}
	return opts
}
