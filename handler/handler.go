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

	e.GET("/getApplications", h.getDocumentHolds)
	e.GET("/getRnaApplications", h.getAxcelerates)
	e.GET("/getEngines", h.getEngines)
	// e.GET("/getDataSourceTemplate", h.getDataSourceTemplate)
	e.GET("/getDataSourceTemplates", h.getDataSourceTemplates)
	e.GET("/getCustodians", h.getCustodians)
	e.GET("/getGlobalSearches", h.getGlobalSearches)
	e.GET("/getFieldProperties", h.getFieldProperties)
	e.GET("/getRedactionReasons", h.getRedactionReasons)

	e.POST("/submitFtpIngestionData", h.submiteFtpIngestionData)
	e.POST("/submitTagger", h.submitTagger)

	e.POST("/importUsersAndGroups", h.importUsersAndGroups)
	e.POST("/importGlobalSearchesAndTaggers", h.importGlobalSearchesAndTaggers)

	e.POST("/addRedactionReason", h.addRedactionReason)
}

type SubmitQueryParams struct {
	Application string
	Engine      string
	Datasource  string
	Template    string
	Custodian   string
	FtpPath     string
}

// getParams extracts query parameters from the given echo context and returns them as a SubmitQueryParams
// It also removes the leading slash from the ftpPath parameter if present and adds the "ftp://localhost/"
// prefix to it.
func getParams(c echo.Context) SubmitQueryParams {

	// remove leading slash
	ftpPath := c.QueryParam("ftpPath")
	if len(ftpPath) > 0 && ftpPath[0] == '/' {
		ftpPath = ftpPath[1:]
	}
	ftpPath = fmt.Sprintf("ftp://localhost/%s", ftpPath)

	return SubmitQueryParams{
		Application: c.QueryParam("application"),
		Engine:      c.QueryParam("engine"),
		Datasource:  c.QueryParam("dataSource"),
		Template:    c.QueryParam("dataSourceTemplate"),
		Custodian:   c.QueryParam("custodian"),
		FtpPath:     ftpPath,
	}

}

// submiteFtpIngestionData submits a new ftp ingestion datasource to the ADP server with the given parameters.
//
// It first creates a new datasource with the given identifier and template.
// If the engine parameter is given, it will configure the datasource to use the given engine,
// otherwise it will configure the datasource to use the given application.
// It will then configure the datasource to ingest the given ftp path with the given custodian.
// Finally, it will start the datasource asynchronously.
//
// It will return a 400 response if any of the above steps fail.
func (h *Handler) submiteFtpIngestionData(c echo.Context) error {

	params := getParams(c)
	log.Debug().Msgf("params: %+v", params)

	opts := []func(*adp.CreateDataSourceConfiguration){
		adp.WithCreateDatasourceDatasourceIdentifier(params.Datasource),
		adp.WithCreateDatasourceDatasourceTemplate(params.Template),
	}

	if params.Engine != "" {
		opts = append(opts, adp.WithCreateDatasourceEngineIdentifier(params.Engine))
	} else if params.Application != "" {
		opts = append(opts, adp.WithCreateDatasourceApplicationIdentifier(params.Application))
	}

	if err := h.service.ADPsvc.CreateDataSource(opts...); err != nil {
		log.Error().Err(err).Msg("failed to create datasource")
		return c.JSON(http.StatusBadRequest, err)
	}

	configs := []adp.ConfigTableMapsArg{
		{
			Action:       "Update",
			Column:       "0",
			Row:          0,
			Substitution: "",
			TableName:    "crawlSeedURIs",
			Value:        params.FtpPath,
		},
		{
			Action:       "Update",
			Column:       "0",
			Row:          0,
			Substitution: "",
			TableName:    "crawlLocationClassifierRules",
			Value:        "*",
		},
		{
			Action:       "Update",
			Column:       "1",
			Row:          0,
			Substitution: "",
			TableName:    "crawlLocationClassifierRules",
			Value:        params.Custodian,
		},
		{
			Action:       "Update",
			Column:       "2",
			Row:          0,
			Substitution: "",
			TableName:    "crawlLocationClassifierRules",
			Value:        "rm_custodian",
		},
	}

	if err := h.service.ADPsvc.ConfigureDataSource(
		adp.WithConfigureDataSourceNames(params.Datasource),
		adp.WithConfigureDataSourceMetaDataMappingToConfigTables(configs),
	); err != nil {
		log.Error().Err(err).Msg("failed to configure datasource")
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := h.service.ADPsvc.StartDataSource(
		adp.WithStartDataSourceDataSourceName(params.Datasource),
		adp.WithStartDataSourceSynchronous(false),
	); err != nil {
		log.Error().Err(err).Msg("failed to start datasource")
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, "Submitted")
}

// getDocumentHolds returns all document holds the user has access to.
//
// This endpoint first extracts the user name from the echo context and then uses it to query the ADP server for all document holds the user has access to.
// The result is then returned as JSON.
func (h *Handler) getDocumentHolds(c echo.Context) error {
	userName := c.Get("user").(string)

	res, err := h.service.ADPsvc.ListDocumentHoldsByUser(userName)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
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
		return c.JSON(http.StatusBadRequest, err)
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
		return c.JSON(http.StatusBadRequest, "application is required")
	}

	userName := c.Get("user").(string)

	opts := []func(*adp.ListEntitiesConfiguration){
		adp.WithListEntitiesType("singleMindServer"),
		adp.WithListEntitiesRelatedEntity(app),
		adp.WithListEntitiesUserHasAccess(userName),
	}

	res, err := h.service.ADPsvc.ListEntities(opts...)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
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
		return c.JSON(http.StatusBadRequest, "application is required")
	}

	res, err := h.service.ADPsvc.GetCustodiansByApplicationID(app)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
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
		return c.JSON(http.StatusBadRequest, err)
	}

	var templates []adp.Entity
	for _, ds := range res {
		if strings.Contains(strings.ToLower(ds.DisplayName), "template") {
			templates = append(templates, ds)
		}
	}

	if len(templates) == 0 {
		return c.JSON(http.StatusBadRequest, "no templates found")
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

	documentHolds, err := h.service.ADPsvc.ListDocumentHoldsByUser(userName)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	log.Debug().Msgf("user [%s] has access to documentHolds: %+v", userName, documentHolds)

	if err := service.VerifyApplications(documentHolds, userGroupInput.ApplicationRoles); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	log.Debug().Msgf("user [%s] has access to all applications in the sheet", userName)

	var opts []func(*adp.ManageUsersAndGruopsConfiguration) = []func(*adp.ManageUsersAndGruopsConfiguration){}

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

	opts = []func(*adp.ManageUsersAndGruopsConfiguration){
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
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, nil)
}

func (h *Handler) getGlobalSearches(c echo.Context) error {
	resp, err := h.service.ADPsvc.GlobalSearches()
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	var globalSearchIDs []string
	for _, gs := range resp {
		globalSearchIDs = append(globalSearchIDs, gs.ID)
	}
	return c.JSON(http.StatusOK, globalSearchIDs)
}

func (h *Handler) getFieldProperties(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return c.JSON(http.StatusBadRequest, "application is required")
	}

	log.Debug().Msgf("application: %s", app)
	entities, err := h.service.ADPsvc.ListEntitiesByRelatedEntity("dataModel", app)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	dataModel := entities[0].ID
	log.Debug().Msgf("get dataModel: %s", dataModel)

	props, err := h.service.ADPsvc.GetFieldProperties(dataModel)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
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
		return c.JSON(http.StatusBadRequest, "application is required")
	}

	res, err := h.service.ADPsvc.GetCategories(app, "rmRedactReason")
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) addRedactionReason(c echo.Context) error {
	app := c.QueryParam("application")
	if app == "" {
		return c.JSON(http.StatusBadRequest, "application is required")
	}

	redactionReason := c.QueryParam("redactionReason")
	if redactionReason == "" {
		return c.JSON(http.StatusBadRequest, "redactionReason is required")
	}

	res, err := h.service.ADPsvc.CreateOrUpdateCategory(app, "Redaction Reason", redactionReason, redactionReason)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, res)
}
