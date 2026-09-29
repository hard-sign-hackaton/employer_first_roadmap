// import-moscow-catalog loads the reviewed real catalog into an explicitly
// selected empty database. It refuses to mix it with an existing catalog.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"efr_bot/database"
	"efr_bot/models"
	"gorm.io/gorm"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := database.Open(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := ensureCatalogEmpty(db); err != nil {
		log.Fatal(err)
	}
	if err := database.SeedMoscowCatalog(db); err != nil {
		log.Fatalf("import Moscow catalog: %v", err)
	}
	log.Print("Московский каталог импортирован")
}

func ensureCatalogEmpty(db *gorm.DB) error {
	for _, model := range []any{&models.Company{}, &models.University{}, &models.CareerDirection{}, &models.EducationProgram{}, &models.Roadmap{}} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("target database is not empty (contains %T); refusing to mix catalogs", model)
		}
	}
	return nil
}
