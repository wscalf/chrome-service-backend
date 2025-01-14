package service

import (
	"encoding/json"
	"reflect"

	"github.com/RedHatInsights/chrome-service-backend/rest/database"
	"github.com/RedHatInsights/chrome-service-backend/rest/models"
	"github.com/RedHatInsights/chrome-service-backend/rest/util"
	"gorm.io/gorm"
)

func GetAllUserDashboardTemplates(userId uint) ([]models.DashboardTemplate, error) {
	var userDashboardTemplates []models.DashboardTemplate

	result := database.DB.Where("user_identity_id = ?", userId).Find(&userDashboardTemplates)

	return userDashboardTemplates, result.Error
}

func GetUserDashboardTemplate(userId uint, dashboard models.AvailableTemplates) ([]models.DashboardTemplate, error) {
	var userDashboardTemplates []models.DashboardTemplate

	templateBase := models.DashboardTemplateBase{
		Name:        dashboard.String(),
		DisplayName: util.BaseTemplates[dashboard].DisplayName,
	}
	result := database.DB.Where("user_identity_id = ? AND name = ?", userId, dashboard).Find(&userDashboardTemplates)

	if result.RowsAffected == 0 {
		if result.RowsAffected == 0 && result.Error != nil {
			return userDashboardTemplates, result.Error
		}

		baseTemplate := util.BaseTemplates[dashboard]

		dashboardTemplate := models.DashboardTemplate{
			UserIdentityID: userId,
			Default:        true,
			TemplateBase:   templateBase,
			TemplateConfig: baseTemplate.TemplateConfig,
		}

		result = database.DB.Create(&dashboardTemplate)

		userDashboardTemplates = append(userDashboardTemplates, dashboardTemplate)
	}

	return userDashboardTemplates, result.Error
}

func GetDashboardTemplate(userId uint, dashboard models.AvailableTemplates) ([]models.DashboardTemplate, error) {
	var userDashboardTemplates []models.DashboardTemplate
	var err error
	if dashboard.String() == "" {
		userDashboardTemplates, err = GetAllUserDashboardTemplates(userId)
	} else {
		userDashboardTemplates, err = GetUserDashboardTemplate(userId, dashboard)
	}

	return userDashboardTemplates, err
}

func UpdateDashboardTemplate(templateId uint, userId uint, dashboardTemplate models.DashboardTemplate) (models.DashboardTemplate, error) {
	var userDashboardTemplate models.DashboardTemplate
	var err error

	result := database.DB.Find(&userDashboardTemplate, templateId)
	if result.RowsAffected == 0 || result.Error != nil {
		return userDashboardTemplate, gorm.ErrRecordNotFound
	}

	if userDashboardTemplate.UserIdentityID != userId {
		return userDashboardTemplate, util.ErrNotAuthorized
	}

	configs := reflect.ValueOf(dashboardTemplate.TemplateConfig)
	typeOfS := configs.Type()

	for i := 0; i < configs.NumField(); i++ {
		var items []models.GridItem
		json.Unmarshal(configs.Field(i).Bytes(), &items)
		layoutSize := typeOfS.Field(i).Tag.Get("json")
		for _, gi := range items {
			// initialize coordinates if they do not exist
			if gi.Y == 0 {
				gi.Y = 0
			}
			if gi.X == 0 {
				gi.X = 0
			}

			err = gi.IsValid(models.GridSizes(layoutSize))
			if err != nil {
				return userDashboardTemplate, err
			}
		}

		if len(items) > 0 {
			// replace only non empty items, not the whole config
			dashboardTemplate.TemplateConfig.SetLayoutSizeItems(typeOfS.Field(i).Name, items)
		}
	}

	// Update only the templates, no other fields are allowed to be updated
	database.DB.Model(&userDashboardTemplate).Updates(models.DashboardTemplate{
		TemplateConfig: dashboardTemplate.TemplateConfig,
	})

	return userDashboardTemplate, err
}

func GetAllBaseTemplates() []models.BaseDashboardTemplate {
	var templates []models.BaseDashboardTemplate
	for _, template := range util.BaseTemplates {
		templates = append(templates, template)
	}

	return templates
}

func GetDashboardTemplateBase(dashboard models.AvailableTemplates) (models.BaseDashboardTemplate, error) {
	var baseTemplate models.BaseDashboardTemplate

	err := dashboard.IsValid()

	if err != nil {
		return baseTemplate, err
	}

	baseTemplate = util.BaseTemplates[dashboard]

	return baseTemplate, err
}

func CopyDashboardTemplate(accountId uint, dashboardTemplateId uint) (models.DashboardTemplate, error) {
	var dashboardTemplate models.DashboardTemplate

	result := database.DB.Find(&dashboardTemplate, dashboardTemplateId)
	if result.RowsAffected == 0 || result.Error != nil {
		return dashboardTemplate, gorm.ErrRecordNotFound
	}

	newDashboardTemplate := models.DashboardTemplate{
		UserIdentityID: accountId,
		TemplateBase:   dashboardTemplate.TemplateBase,
		TemplateConfig: dashboardTemplate.TemplateConfig,
	}

	result = database.DB.Create(&newDashboardTemplate)

	return newDashboardTemplate, result.Error
}

func DeleteTemplate(accountId uint, dashboardTemplateId uint) error {
	var dashboardTemplate models.DashboardTemplate

	result := database.DB.Find(&dashboardTemplate, dashboardTemplateId)
	if result.RowsAffected == 0 || result.Error != nil {
		return gorm.ErrRecordNotFound
	}

	if dashboardTemplate.UserIdentityID != accountId {
		return util.ErrNotAuthorized
	}

	database.DB.Delete(&dashboardTemplate)

	return result.Error
}

func ChangeDefaultTemplate(accountId uint, dashboardId uint) (models.DashboardTemplate, error) {
	var dashboardTemplate models.DashboardTemplate

	result := database.DB.Find(&dashboardTemplate, dashboardId)
	if result.RowsAffected == 0 || result.Error != nil {
		return dashboardTemplate, gorm.ErrRecordNotFound
	}

	if dashboardTemplate.UserIdentityID != accountId {
		return dashboardTemplate, util.ErrNotAuthorized
	}

	dashboardType := dashboardTemplate.TemplateBase.Name

	result = database.DB.Model(models.DashboardTemplate{}).Where("user_identity_id = ? AND name = ?", accountId, dashboardType).Update("default", false)

	if result.Error != nil {
		return dashboardTemplate, result.Error
	}

	result = database.DB.Model(&dashboardTemplate).Updates(models.DashboardTemplate{
		Default: true,
	})

	return dashboardTemplate, result.Error
}
