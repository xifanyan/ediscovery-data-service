package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/xifanyan/ediscovery-data-service/service"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/xifanyan/adp"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) SetupRouter(e *echo.Echo) {

	e.GET("/getTemplates", h.getTemplates)
	e.GET("/getWorkspaces", h.getWorkspaces)
	e.GET("/getHosts", h.getHosts)
	e.GET("/getApplications", h.getDocumentHolds)
	e.GET("/getRnaApplications", h.getAxcelerates)
	e.GET("/getEngines", h.getEngines)
	e.GET("/getDataSourceTemplates", h.getDataSourceTemplates)
	e.GET("/getCustodians", h.getCustodians)
	e.GET("/getGlobalSearches", h.getGlobalSearches)
	e.GET("/getFieldProperties", h.getFieldProperties)
	e.GET("/getTaxonomies", h.getTaxonomies)
	e.GET("/getRedactionReasons", h.getRedactionReasons)

	e.GET("/entity/:entityType", h.getEntity)

	// User and Group Management
	e.GET("/users", h.getUsers)
	e.GET("/users/:userID", h.getUserByID)
	e.GET("/groups", h.getGroups)
	e.GET("/groups/:groupID", h.getGroupByID)
	e.GET("/application/:applicationID/usersAndGroups", h.getUsersAndGroupsByApplicationID)

	e.POST("/users", h.createUsers)
	e.POST("/groups", h.createGroups)
	e.POST("/group/:groupID/users", h.addUsersToGroup)
	e.POST("/application/:applicationID/users", h.addUsersOrGroupsToApplication)
	e.POST("/application/:applicationID/groups", h.addUsersOrGroupsToApplication)

	e.GET("/groups/:groupID/users", h.getUsersByGroupID)
	e.GET("/users/:userID/groups", h.getGroupsByUserID)

	e.POST("/createApplication", h.createApplication)

	e.POST("/submitFtpIngestionData", h.submitFtpIngestionData)
	e.POST("/submitFileIngestionData", h.submitFileIngestionData)

	e.GET("/getGlobalSearches", h.getGlobalSearches)
	e.POST("/createGlobalSearches", h.createGlobalSearches)
	e.POST("/updateGlobalSearches", h.updateGlobalSearches)

	e.POST("/submitTagger", h.submitTagger)

	e.POST("/importUsersAndGroups", h.importUsersAndGroups)
	e.POST("/importGlobalSearchesAndTaggers", h.importGlobalSearchesAndTaggers)

	e.POST("/addRedactionReason", h.addRedactionReason)
	e.POST("/addCustodian", h.addCustodian)
}

type DataIngestionParams struct {
	Application string
	Engine      string
	Datasource  string
	Template    string
	Custodian   string
	Path        string
}

func newDataIngestionParams(c echo.Context) *DataIngestionParams {
	return &DataIngestionParams{
		Application: c.QueryParam("application"),
		Engine:      c.QueryParam("engine"),
		Datasource:  c.QueryParam("dataSource"),
		Template:    c.QueryParam("dataSourceTemplate"),
		Custodian:   c.QueryParam("custodian"),
	}
}

func geFtpParams(c echo.Context) DataIngestionParams {

	// remove leading slash
	ftpPath := c.QueryParam("ftpPath")
	if len(ftpPath) > 0 && ftpPath[0] == '/' {
		ftpPath = ftpPath[1:]
	}
	ftpPath = fmt.Sprintf("ftp://localhost/%s", ftpPath)

	params := newDataIngestionParams(c)
	params.Path = ftpPath

	return *params
}

func geFileParams(c echo.Context) DataIngestionParams {

	filePath := c.QueryParam("filePath")

	params := newDataIngestionParams(c)
	params.Path = filePath
	log.Debug().Msgf("params: %+v", params)

	return *params
}

func (h *Handler) handleADPError(c echo.Context, err error) error {
	return c.JSON(
		http.StatusInternalServerError,
		echo.Map{"error": err.Error()},
	)
}

func (h *Handler) handleValidationError(c echo.Context, err error) error {
	return c.JSON(
		http.StatusBadRequest,
		echo.Map{"error": err.Error()},
	)
}

func (h *Handler) getEntity(c echo.Context) error {
	userName := c.Get("user").(string)

	entityType := c.Param("entityType")
	switch entityType {
	case "documentHold", "axcelerate", "dataSource", "singleMindServer", "mergingMeta":
		var opts []func(*adp.ListEntitiesConfiguration)

		opts = append(opts, adp.WithListEntitiesType(entityType))

		if c.QueryParam("security") != "false" {
			opts = append(opts, adp.WithListEntitiesUserHasAccess(userName))
		}
		if c.QueryParam("workspace") != "" {
			opts = append(opts, adp.WithListEntitiesWorkspace(c.QueryParam("workspace")))
		}
		entities, err := h.service.ADPsvc.ListEntities(opts...)
		if err != nil {
			return h.handleADPError(c, err)
		}

		if c.QueryParam("globalTemplate") == "true" {
			var selected []adp.Entity
			for _, entity := range entities {
				if entity.GlobalTemplateFlag {
					selected = append(selected, entity)
				}
			}
			entities = selected
		}

		if len(entities) == 0 {
			return c.JSON(http.StatusNotFound, echo.Map{"error": service.ErrEntityNotFound.Error()})
		}

		return c.JSON(http.StatusOK, entities)
	default:
		return h.handleValidationError(c, service.ErrValidEntityTypeRequired)
	}

}

func createDataSourceOptions(params DataIngestionParams) []func(*adp.CreateDataSourceConfiguration) {
	opts := []func(*adp.CreateDataSourceConfiguration){
		adp.WithCreateDatasourceDatasourceIdentifier(params.Datasource),
		adp.WithCreateDatasourceDatasourceTemplate(params.Template),
	}

	if params.Engine != "" {
		opts = append(opts, adp.WithCreateDatasourceEngineIdentifier(params.Engine))
	} else if params.Application != "" {
		opts = append(opts, adp.WithCreateDatasourceApplicationIdentifier(params.Application))
	}

	return opts
}

func configDataSourceOptions(params DataIngestionParams) []func(*adp.ConfigureDataSourceConfiguration) {
	configs := []adp.ConfigTableMapsArg{
		{
			Action:       "Update",
			Column:       "0",
			Row:          0,
			Substitution: "",
			TableName:    "crawlSeedURIs",
			Value:        params.Path,
		},
	}

	if params.Custodian != "" {
		configs = append(configs,
			adp.ConfigTableMapsArg{
				Action:       "Update",
				Column:       "0",
				Row:          0,
				Substitution: "",
				TableName:    "crawlLocationClassifierRules",
				Value:        "*",
			},
			adp.ConfigTableMapsArg{
				Action:       "Update",
				Column:       "1",
				Row:          0,
				Substitution: "",
				TableName:    "crawlLocationClassifierRules",
				Value:        params.Custodian,
			},
			adp.ConfigTableMapsArg{
				Action:       "Update",
				Column:       "2",
				Row:          0,
				Substitution: "",
				TableName:    "crawlLocationClassifierRules",
				Value:        "rm_custodian",
			},
		)
	}

	opts := []func(*adp.ConfigureDataSourceConfiguration){
		adp.WithConfigureDataSourceNames(params.Datasource),
		adp.WithConfigureDataSourceMetaDataMappingToConfigTables(configs),
	}
	return opts
}

func startDataSourceOptions(params DataIngestionParams) []func(*adp.StartDataSourceConfiguration) {
	opts := []func(*adp.StartDataSourceConfiguration){
		adp.WithStartDataSourceDataSourceName(params.Datasource),
		adp.WithStartDataSourceSynchronous(false),
	}
	return opts
}

func (h *Handler) submitIngestionData(c echo.Context, params DataIngestionParams) error {
	log.Debug().Msgf("params: %+v", params)

	createDataSourceOpts := createDataSourceOptions(params)
	if err := h.service.ADPsvc.CreateDataSource(createDataSourceOpts...); err != nil {
		log.Error().Err(err).Msg("failed to create datasource")
		return h.handleADPError(c, err)
	}

	configDataSourceOpts := configDataSourceOptions(params)
	if err := h.service.ADPsvc.ConfigureDataSource(configDataSourceOpts...); err != nil {
		log.Error().Err(err).Msg("failed to configure datasource")
		return h.handleADPError(c, err)
	}

	startDataSourceOpts := startDataSourceOptions(params)
	if err := h.service.ADPsvc.StartDataSource(startDataSourceOpts...); err != nil {
		log.Error().Err(err).Msg("failed to start datasource")
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, nil)
}

func (h *Handler) submitFtpIngestionData(c echo.Context) error {
	params := geFtpParams(c)
	return h.submitIngestionData(c, params)

}

func (h *Handler) submitFileIngestionData(c echo.Context) error {
	params := geFileParams(c)
	return h.submitIngestionData(c, params)
}

// getDocumentHolds returns all document holds the user has access to.
//
// This endpoint first extracts the user name from the echo context and then uses it to query the ADP server for all document holds the user has access to.
// The result is then returned as JSON.
func (h *Handler) getDocumentHolds(c echo.Context) error {
	userName := c.Get("user").(string)

	res, err := h.service.ADPsvc.ListDocumentHoldsByUser(userName)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

// getAxcelerates returns all axcelerates the user has access to.
//
// This endpoint first extracts the user name from the echo context and then uses it to query the ADP server for all axcelerates the user has access to.
// The result is then returned as JSON.
func (h *Handler) getAxcelerates(c echo.Context) error {
	userName := c.Get("user").(string)

	res, err := h.service.ADPsvc.ListAxceleratesByUser(userName)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

// getEngines returns all engines that are associated with the given application and that the user has access to.
//
// This endpoint first extracts the user name from the echo context and then uses it to query the ADP server for all engines that are associated with the given application
// and that the user has access to.
// The result is then returned as JSON.
func (h *Handler) getEngines(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return h.handleValidationError(c, service.ErrApplicationRequired)
	}

	userName := c.Get("user").(string)

	opts := []func(*adp.ListEntitiesConfiguration){
		adp.WithListEntitiesType("singleMindServer"),
		adp.WithListEntitiesRelatedEntity(app),
		adp.WithListEntitiesUserHasAccess(userName),
	}

	res, err := h.service.ADPsvc.ListEntities(opts...)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

// getCustodians returns all custodians that are associated with the given application and that the user has access to.
//
// This endpoint first extracts the user name from the echo context and then uses it to query the ADP server for all custodians that are associated with the given application
// and that the user has access to.
// The result is then returned as JSON.
func (h *Handler) getCustodians(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return h.handleValidationError(c, service.ErrApplicationRequired)
	}

	res, err := h.service.ADPsvc.GetCustodiansByApplicationID(app)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

// getDataSourceTemplate returns the given data source template.
//
// This endpoint first checks if the given template name starts with "dataSource.".
// If not, it prefixes the name with "dataSource." and then queries the ADP server for the given template.
// The result is then returned as JSON.
func (h *Handler) getDataSourceTemplates(c echo.Context) error {
	userName := c.Get("user").(string)
	res, err := h.service.ADPsvc.ListDatasourcesByUser(userName)
	if err != nil {
		return h.handleADPError(c, err)
	}

	var templates []adp.Entity
	for _, ds := range res {
		if strings.Contains(strings.ToLower(ds.DisplayName), "template") {
			templates = append(templates, ds)
		}
	}

	if len(templates) == 0 {
		return c.JSON(
			http.StatusNotFound,
			echo.Map{"error": service.ErrTemplateNotFound.Error()},
		)
	}

	return c.JSON(http.StatusOK, templates)
}

func saveToTempFile(r *multipart.FileHeader) (string, error) {
	src, err := r.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open the uploaded file: %v", err)
	}
	defer src.Close()

	// Create a temporary file to store the uploaded Excel file
	tempFile, err := os.CreateTemp("", "upload-*.xlsx")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}

	if _, err = io.Copy(tempFile, src); err != nil {
		return "", fmt.Errorf("failed to copy the uploaded to temp file: %v", err)
	}

	return tempFile.Name(), nil
}

func (h *Handler) importUsersAndGroups(c echo.Context) error {
	userName := c.Get("user").(string)

	r, err := c.FormFile("usersAndGroups")
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to retrieve the uploaded file from form %v", err))
	}

	tempFile, err := saveToTempFile(r)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
	}
	defer os.Remove(tempFile)

	userGroupInput, err := service.GetUsersGroupsRoles(tempFile)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	users, groups, err := h.service.ADPsvc.GetAllUsersAndGroups()
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := service.VerifyUsers(userGroupInput.Users, users); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := service.VerifyGroups(userGroupInput.Groups, groups); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	documentHolds, err := h.service.ADPsvc.ListDocumentHoldsByUser(userName)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	log.Debug().Msgf("user [%s] has access to documentHolds: %+v", userName, documentHolds)

	if err := service.VerifyApplications(documentHolds, userGroupInput.ApplicationRoles); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	log.Debug().Msgf("user [%s] has access to all applications in the sheet", userName)

	var opts []func(*adp.ManageUsersAndGroupsConfiguration) = []func(*adp.ManageUsersAndGroupsConfiguration){}

	opts = service.SetupManageUsersAndGroupsOptions(userGroupInput)
	resp, err := h.service.ADPsvc.ManageUsersAndGroups(opts...)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	log.Debug().Msgf("Setup Users and Groups Response: %+v", resp)

	ids := make([]string, 0)
	for _, documentHold := range documentHolds {
		ids = append(ids, documentHold.ID)
	}
	appIDs := strings.Join(ids, ",")
	log.Debug().Msgf("ids: %+v", ids)

	opts = []func(*adp.ManageUsersAndGroupsConfiguration){
		adp.WithManageUsersAndGroupsAppIdsToFilterFor(appIDs),
		adp.WithManageUsersAndGroupsReturnAllUsersUnderGroup("true"),
	}
	resp, err = h.service.ADPsvc.ManageUsersAndGroups(opts...)
	log.Debug().Msgf("Load Application Security Setting Response: %+v", resp)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) submitTagger(c echo.Context) error {
	var err error

	application := c.QueryParam("application")
	if application == "" {
		return h.handleValidationError(c, service.ErrApplicationRequired)
	}

	parts := strings.Split(application, ".")
	applicationType := parts[0]

	tags := []adp.TaggerInfo{
		{
			ID:             c.QueryParam("id"),
			GlobalSearchID: c.QueryParam("globalSearch"),
			Description:    c.QueryParam("description"),
			TermTaxonomy:   c.QueryParam("termTaxonomy"),
			TypeTaxonomy:   c.QueryParam("typeTaxonomy"),
		},
	}

	log.Debug().Msgf("tags: %+v", tags)
	js, _ := json.Marshal(tags)
	log.Debug().Msgf("js: %+v", string(js))

	err = h.service.ADPsvc.ManageTaggers(
		adp.WithAdpManTagsApplicationIdentifier(application),
		adp.WithAdpManTagsApplicationType(applicationType),
		adp.WithAdpManTagsJSONInstall(string(js)),
		adp.WithAdpManTagsWait4Completion("true"),
	)

	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, nil)
}

func (h *Handler) getTaxonomies(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return h.handleValidationError(c, service.ErrApplicationRequired)
	}

	entities, err := h.service.ADPsvc.ListEntitiesByRelatedEntity("dataModel", app)
	if err != nil {
		return h.handleADPError(c, err)
	}

	dataModel := entities[0].ID
	log.Debug().Msgf("get dataModel: %s", dataModel)

	props, err := h.service.ADPsvc.GetIndexConfigurationTable(dataModel)
	if err != nil {
		return h.handleADPError(c, err)
	}

	taxonomies := []string{}
	for key, prop := range props {
		if prop.StructuredView {
			taxonomies = append(taxonomies, key)
		}
	}
	return c.JSON(http.StatusOK, taxonomies)
}

func (h *Handler) getFieldProperties(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return h.handleValidationError(c, service.ErrApplicationRequired)
	}

	log.Debug().Msgf("application: %s", app)
	entities, err := h.service.ADPsvc.ListEntitiesByRelatedEntity("dataModel", app)
	if err != nil {
		return h.handleADPError(c, err)
	}

	dataModel := entities[0].ID
	log.Debug().Msgf("get dataModel: %s", dataModel)

	props, err := h.service.ADPsvc.GetFieldProperties(dataModel)
	if err != nil {
		return h.handleADPError(c, err)
	}

	fieldProperties := make(map[string]string)
	for key, prop := range props {
		fieldProperties[key] = prop.DisplayName
	}
	return c.JSON(http.StatusOK, fieldProperties)
}

func (h *Handler) importGlobalSearchesAndTaggers(c echo.Context) error {
	var err error

	r, err := c.FormFile("globalSearchesAndTaggers")
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve the uploaded file from form")
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to retrieve the uploaded file from form %v", err))
	}

	tempFile, err := saveToTempFile(r)
	if err != nil {
		log.Error().Err(err).Msg("failed to save to temp file")
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
	}
	defer os.Remove(tempFile)

	settings, err := service.GetGloalSearchesAndTaggers(tempFile)
	if err != nil {
		log.Error().Err(err).Msg("failed to get global searches and taggers")
		return c.JSON(http.StatusBadRequest, err)
	}

	js, _ := json.Marshal(settings.GlobalSearchSettings)
	fmt.Println("js: ", adp.Prettify(string(js)))

	_, err = h.service.ADPsvc.GlobalSearches(
		adp.WithGlobalSearchesCreateUpdateGlobalSearches(string(js)),
	)

	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	settings.TaggerSettings = []service.TaggerSetting{}

	for _, taggerSetting := range settings.TaggerSettings {
		js, _ := json.Marshal(taggerSetting.TaggerInfos)
		log.Debug().Msgf("js: %+v", string(js))

		parts := strings.Split(taggerSetting.Application, ".")
		applicationType := parts[0]

		err = h.service.ADPsvc.ManageTaggers(
			adp.WithAdpManTagsApplicationIdentifier(taggerSetting.Application),
			adp.WithAdpManTagsApplicationType(applicationType),
			adp.WithAdpManTagsJSONInstall(string(js)),
			adp.WithAdpManTagsWait4Completion("true"),
		)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
	}

	return c.JSON(http.StatusOK, nil)
}

func (h *Handler) getRedactionReasons(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return h.handleValidationError(c, service.ErrApplicationRequired)
	}

	res, err := h.service.ADPsvc.GetCategories(app, "rmRedactReason")
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) addRedactionReason(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return h.handleValidationError(c, service.ErrApplicationRequired)
	}

	redactionReason := c.QueryParam("redactionReason")
	if redactionReason == "" {
		return h.handleValidationError(c, service.ErrRedactionReasonRequired)
	}

	res, err := h.service.ADPsvc.CreateOrUpdateCategory(app, "Redaction Reason", redactionReason, redactionReason)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) addCustodian(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return h.handleValidationError(c, service.ErrApplicationRequired)
	}

	custodian := c.QueryParam("custodian")
	if custodian == "" {
		return h.handleValidationError(c, service.ErrCustodianRequired)
	}

	res, err := h.service.ADPsvc.CreateOrUpdateCategory(app, "Custodian", custodian, custodian)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) getWorkspaces(c echo.Context) error {
	res, err := h.service.ADPsvc.ListWorkspaces()
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) getHosts(c echo.Context) error {
	res, err := h.service.ADPsvc.ListHosts()
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

// NOTES: binding query parameters in echo only works with GET/DELETE
type CreateApplicationQueryParams struct {
	ApplicationType string
	ApplicationName string
	Workspace       string
	Host            string
	Template        string
}

func (h *Handler) createApplication(c echo.Context) error {
	opts, err := checkCreateApplicationParams(c)

	if err != nil {
		log.Debug().Msgf("check create application params: %+v", err)
		return h.handleValidationError(c, err)
	}

	res, err := h.service.ADPsvc.CreateApplication(opts...)
	if err != nil {
		return h.handleADPError(c, err)
	}

	// newAppID := res.ApplicationIdentifier
	if c.QueryParam("dropTemplate") == "true" {
		log.Debug().Msgf("dropping template: %s", res.ApplicationIdentifier)
		err = h.service.ADPsvc.DropTemplate(res.ApplicationIdentifier)
		if err != nil {
			return h.handleADPError(c, err)
		}
	}

	if c.QueryParam("startApplication") == "true" {
		log.Debug().Msgf("starting application: %s", res.ApplicationIdentifier)
		executionID, err := h.service.ADPsvc.StartApplicationAsync(res.ApplicationIdentifier)
		if err != nil {
			return h.handleADPError(c, err)
		}
		log.Debug().Msgf("executionID: %s", executionID)
	}

	return nil
}

func checkCreateApplicationParams(c echo.Context) ([]func(*adp.CreateApplicationConfiguration), error) {
	queryParams := CreateApplicationQueryParams{
		ApplicationType: c.QueryParam("applicationType"),
		ApplicationName: c.QueryParam("applicationName"),
		Workspace:       c.QueryParam("workspace"),
		Host:            c.QueryParam("host"),
		Template:        c.QueryParam("template"),
	}

	var opts []func(*adp.CreateApplicationConfiguration)
	if queryParams.ApplicationType == "documentHold" || queryParams.ApplicationType == "axcelerateStandalone" {
		opts = append(opts, adp.WithCreateApplicationApplicationType(queryParams.ApplicationType))
	} else {
		return nil, service.ErrApplicationTypeNotSupported
	}

	opts = append(opts, adp.WithCreateApplicationApplicationType(queryParams.ApplicationType))
	if queryParams.ApplicationName != "" {
		opts = append(opts, adp.WithCreateApplicationApplicationName(queryParams.ApplicationName))
	} else {
		return nil, service.ErrApplicationNameRequired
	}

	if queryParams.Workspace != "" {
		opts = append(opts, adp.WithCreateApplicationApplicationWorkspace(queryParams.Workspace))
	}

	if queryParams.Host != "" {
		opts = append(opts, adp.WithCreateApplicationApplicationHost(queryParams.Host))
	}

	if queryParams.Template != "" {
		opts = append(opts, adp.WithCreateApplicationApplicationTemplate(queryParams.Template))
	} else {
		return nil, service.ErrTemplateRequired
	}

	return opts, nil
}

func (h *Handler) getTemplates(c echo.Context) error {
	var err error

	userName := c.Get("user").(string)
	entityType := c.QueryParam("entityType")

	var availableTemplates []adp.Entity

	switch entityType {
	case "documentHold", "axcelerate", "dataSource", "singleMindServer", "mergingMeta":
		availableTemplates, err = h.service.ADPsvc.ListAvailableTemplates(entityType, userName)
		if err != nil {
			return h.handleADPError(c, err)
		}
		if len(availableTemplates) == 0 {
			return c.JSON(http.StatusNotFound, echo.Map{"error": service.ErrTemplateNotFound.Error()})
		}
	default:
		return h.handleValidationError(c, service.ErrValidEntityTypeRequired)
	}

	return c.JSON(http.StatusOK, availableTemplates)
}

func (h *Handler) getUsers(c echo.Context) error {
	users, _, err := h.service.ADPsvc.GetAllUsersAndGroups()
	if err != nil {
		return h.handleADPError(c, err)
	}

	if len(users) == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": service.ErrUserNotFound.Error()})
	}
	return c.JSON(http.StatusOK, users)
}

func (h *Handler) getUserByID(c echo.Context) error {
	id := c.Param("userID")
	user, err := h.service.ADPsvc.GetUserByID(id)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, user)
}

func (h *Handler) getGroups(c echo.Context) error {
	_, groups, rec := h.service.ADPsvc.GetAllUsersAndGroups()
	if rec != nil {
		return h.handleADPError(c, rec)
	}

	if len(groups) == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": service.ErrGroupNotFound.Error()})
	}
	return c.JSON(http.StatusOK, groups)
}

func (h *Handler) getGroupByID(c echo.Context) error {
	id := c.Param("groupID")
	group, err := h.service.ADPsvc.GetGroupByID(id)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, group)
}

func (h *Handler) getUsersByGroupID(c echo.Context) error {
	id := c.Param("groupID")
	groups, err := h.service.ADPsvc.GetUsersByGroupID(id)
	if err != nil {
		return h.handleADPError(c, err)
	}

	if len(groups) == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": service.ErrGroupNotFound.Error()})
	}
	return c.JSON(http.StatusOK, groups)
}

func (h *Handler) getGroupsByUserID(c echo.Context) error {
	id := c.Param("userID")
	groups, err := h.service.ADPsvc.GetGroupsByUserID(id)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, groups)
}

func (h *Handler) createUsers(c echo.Context) error {
	var users []adp.UserDefinition

	err := c.Bind(&users)
	if err != nil {
		return h.handleValidationError(c, err)
	}

	log.Debug().Msgf("users: %+v", users)

	if err := h.service.ADPsvc.AddUsers(users); err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, users)
}

func (h *Handler) createGroups(c echo.Context) error {
	var groups []adp.GroupDefinition

	err := c.Bind(&groups)
	if err != nil {
		return h.handleValidationError(c, err)
	}

	log.Debug().Msgf("groups: %+v", groups)

	if err := h.service.ADPsvc.AddGroups(groups); err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, groups)
}

func (h *Handler) addUsersToGroup(c echo.Context) error {
	groupID := c.Param("groupID")

	var users []string
	err := c.Bind(&users)
	if err != nil {
		return h.handleValidationError(c, err)
	}

	if err := h.service.ADPsvc.AddUsersToGroup(users, groupID); err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Users added to group successfully"})
}

func (h *Handler) addUsersOrGroupsToApplication(c echo.Context) error {
	applicationID := c.Param("applicationID")

	var inputRoles []adp.UserOrGroupToRoles
	err := c.Bind(&inputRoles)
	if err != nil {
		return h.handleValidationError(c, err)
	}

	// Convert UserOrGroupToRoles to ApplicationRoles
	var appRoles []adp.ApplicationRoles
	for _, role := range inputRoles {
		appRoles = append(appRoles, adp.ApplicationRoles{
			Enabled:               true,
			GroupOrUserName:       role.Name,
			ApplicationIdentifier: applicationID,
			Roles:                 role.Roles,
		})
	}

	log.Debug().Msgf("%s : converted roles: %+v", applicationID, appRoles)

	if err := h.service.ADPsvc.AssignUsersOrGroupsToApplication(appRoles, applicationID); err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Users or groups added to application successfully"})
}

func (h *Handler) getUsersAndGroupsByApplicationID(c echo.Context) error {
	applicationID := c.Param("applicationID")

	users, groups, err := h.service.ADPsvc.GetUsersAndGroupsByApplicationID(applicationID)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"users": users, "groups": groups})
}

func (h *Handler) getGlobalSearches(c echo.Context) error {
	res, err := h.service.ADPsvc.ListGlobalSearches()
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) createGlobalSearches(c echo.Context) error {
	var gsdef []adp.GlobalSearchDefinition

	err := c.Bind(&gsdef)
	if err != nil {
		return h.handleValidationError(c, err)
	}

	log.Debug().Msgf("[New] Global Search Definition: %+v", gsdef)

	res, err := h.service.ADPsvc.CreateGlobalSearches(gsdef)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) updateGlobalSearches(c echo.Context) error {
	var gsdef []adp.GlobalSearchDefinition

	err := c.Bind(&gsdef)
	if err != nil {
		return h.handleValidationError(c, err)
	}
	log.Debug().Msgf("[Update] Global Search Definition: %+v", gsdef)

	res, err := h.service.ADPsvc.UpdateGlobalSearches(gsdef)
	if err != nil {
		return h.handleADPError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}
