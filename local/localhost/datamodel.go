package localhost

import (
	"log"

	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/midoblgsm/ubiquity/model"
	"github.com/midoblgsm/ubiquity/resources"
)

//go:generate counterfeiter -o ../../fakes/fake_SpectrumDataModel.go . SpectrumDataModel
type LocalhostDataModel interface {
	CreateVolumeTable() error
	DeleteVolume(name string) error
	InsertVolume(volume resources.Volume) error
	GetVolume(name string) (resources.Volume, bool, error)
	ListVolumes() ([]resources.Volume, error)
	UpdateVolumeMountpoint(name string, mountpoint string) error
}

type localhostDataModel struct {
	log      *log.Logger
	database *gorm.DB
	backend  string
}

func NewLocalhostDataModel(log *log.Logger, db *gorm.DB, backend string) LocalhostDataModel {
	return &localhostDataModel{log: log, database: db, backend: backend}
}

func (d *localhostDataModel) CreateVolumeTable() error {
	d.log.Println("localhostDataModel: Create Volumes Table start")
	defer d.log.Println("localhostDataModel: Create Volumes Table end")

	if err := d.database.AutoMigrate(&resources.Volume{}).Error; err != nil {
		return err
	}
	return nil
}

func (d *localhostDataModel) DeleteVolume(name string) error {
	d.log.Println("localhostDataModel: DeleteVolume start")
	defer d.log.Println("localhostDataModel: DeleteVolume end")

	volume, exists, err := d.GetVolume(name)

	if err != nil {
		return err
	}
	if exists == false {
		return fmt.Errorf("Volume : %s not found", name)
	}

	if err := d.database.Delete(&volume).Error; err != nil {
		return err
	}
	if err := model.DeleteVolume(d.database, &volume).Error; err != nil {
		return err
	}
	return nil
}

func (d *localhostDataModel) InsertVolume(volume resources.Volume) error {
	d.log.Println("localhostDataModel: InsertFilesetVolume start")
	defer d.log.Println("localhostDataModel: InsertFilesetVolume end")

	return d.insertVolume(volume)
}

func (d *localhostDataModel) insertVolume(volume resources.Volume) error {
	d.log.Println("localhostDataModel: insertVolume start")
	defer d.log.Println("localhostDataModel: insertVolume end")
	if err := d.database.Create(&volume).Error; err != nil {
		return err
	}
	return nil
}

func (d *localhostDataModel) GetVolume(name string) (resources.Volume, bool, error) {
	d.log.Println("localhostDataModel: GetVolume start")
	defer d.log.Println("localhostDataModel: GetVolume end")

	volume, err := model.GetVolume(d.database, name, d.backend)
	if err != nil {
		if err.Error() == "record not found" {
			return resources.Volume{}, false, nil
		}
		return resources.Volume{}, false, err
	}

	var localhostVolume resources.Volume
	if err := d.database.Where("volume_id = ?", volume.ID).First(&localhostVolume).Error; err != nil {
		if err.Error() == "record not found" {
			return resources.Volume{}, false, nil
		}
		return resources.Volume{}, false, err
	}
	return localhostVolume, true, nil
}

func (d *localhostDataModel) ListVolumes() ([]resources.Volume, error) {
	d.log.Println("localhostDataModel: ListVolumes start")
	defer d.log.Println("localhostDataModel: ListVolumes end")

	var volumesInDb []resources.Volume

	if err := d.database.Where("backend = ?", d.backend).Find(&volumesInDb).Error; err != nil {
		return nil, err
	}

	return volumesInDb, nil
}

func (d *localhostDataModel) UpdateVolumeMountpoint(name string, mountpoint string) error {
	d.log.Println("localhostDataModel: UpdateVolumeMountpoint start")
	defer d.log.Println("localhostDataModel: UpdateVolumeMountpoint end")

	volume, err := model.GetVolume(d.database, name, d.backend)
	if err != nil {
		return err
	}

	if err = model.UpdateVolumeMountpoint(d.database, &volume, mountpoint); err != nil {
		return fmt.Errorf("Error updating mountpoint of volume %s to %s: %s", volume.Name, mountpoint, err.Error())
	}
	return nil
}
