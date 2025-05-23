package service

import (
	"github.com/rs/zerolog/log"
	adp "github.com/xifanyan/adp"
	"github.com/xuri/excelize/v2"
)

type TaggerSetting struct {
	Application string
	TaggerInfos []adp.TaggerInfo
}

type GlobalSearchesAndTaggersInput struct {
	TaggerSettings       []TaggerSetting
	GlobalSearchSettings []adp.GlobalSearch
}

func getTaggers(rows [][]string) []TaggerSetting {
	var settings []TaggerSetting
	var application string

	for _, row := range rows {
		//  skip the first row
		if row[0] == "Application" {
			continue
		}

		if application == "" && row[0] != "" {
			application = row[0]
		}

		settings = append(settings, TaggerSetting{
			Application: application,
			TaggerInfos: []adp.TaggerInfo{
				{
					ID:             row[1],
					Description:    row[2],
					GlobalSearchID: row[3],
					TermTaxonomy:   row[4],
					TypeTaxonomy:   row[5],
				},
			},
		},
		)
	}

	return settings
}

func getGlobalSearchConfigurationFromSheet(rows [][]string) []adp.GlobalSearch {
	var currentSearch adp.GlobalSearch
	var globalSearches []adp.GlobalSearch

	isNewSearch := true

	for _, row := range rows {
		if row[0] == "" && row[1] == "" && row[2] == "" && row[3] == "" {
			continue
		}

		//  skip the header
		if row[0] == "ID" {
			continue
		}

		if row[0] != "" {
			if !isNewSearch {
				globalSearches = append(globalSearches, currentSearch)
			}

			currentSearch = adp.GlobalSearch{
				ID:          row[0],
				DisplayName: row[1],
				Description: row[2],
				QueryBundle: adp.QueryBundle{
					ActiveQueryParts: make([]adp.ActiveQueryPart, 0),
				},
				SearchParameters: map[string][]string{
					"rm_main":                {"*", "false", "false", "true"},
					"rm_taxonomy_rm_doctype": {"eMail"},
				},
			}

			isNewSearch = false
		}

		if !isNewSearch && row[3] != "" {
			currentSearch.QueryBundle.ActiveQueryParts = append(currentSearch.QueryBundle.ActiveQueryParts,
				adp.ActiveQueryPart{
					Query: row[3],
					Valid: true,
				},
			)
		}
	}

	if !isNewSearch {
		globalSearches = append(globalSearches, currentSearch)
	}

	return globalSearches
}

func GetGloalSearchesAndTaggers(fn string) (*GlobalSearchesAndTaggersInput, error) {

	f, err := excelize.OpenFile(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows, err := f.GetRows("Taggers")
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("Taggers: %+v", rows)
	var input *GlobalSearchesAndTaggersInput

	if len(rows) > 0 {
		input = &GlobalSearchesAndTaggersInput{
			TaggerSettings: getTaggers(rows),
		}
	}

	rows, err = f.GetRows("GlobalSearches")
	if len(rows) > 0 {
		input.GlobalSearchSettings = getGlobalSearchConfigurationFromSheet(rows)
	}

	return input, err
}
