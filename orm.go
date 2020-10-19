package flow

import "gorm.io/gorm"

type OrmConfig struct {
	Enable   bool
	UserName string
	Password string
	DbName   string
	Host     string
	Port     int
	Pool     *OrmPool
}

type OrmPool struct {
	MaxIdle         int
	MaxOpen         int
	ConnMaxLifeTime int64
	ConnMaxIdleTime int64
}

func defOrmConfig() *OrmConfig {
	return &OrmConfig{
		Enable:   false,
		UserName: "root",
		Password: "root",
		Host:     "127.0.0.1",
		Port:     3306,
		Pool:     defOrmPool(),
	}
}

func defOrmPool() *OrmPool {
	return &OrmPool{
		MaxIdle:         5,
		MaxOpen:         10,
		ConnMaxLifeTime: 30000,
		ConnMaxIdleTime: 10000,
	}
}

type Orm struct {
	db     *gorm.DB
	models map[string]*Model
}

func (o *Orm) Register(name string, model *Model) *Orm {
	o.models[name] = model
	return o
}

func (o *Orm) Use(name string) *Model {
	return o.models[name]
}

func (o *Orm) Create(value interface{}) error {
	if err := o.db.Create(value).Error; err != nil {
		return err
	}
	return nil
}

func (o *Orm) Find(value interface{}) error {
	if err := o.db.Find(value).Error; err != nil {
		return err
	}
	return nil
}

func defOrm() *Orm {
	return &Orm{
		models: make(map[string]*Model),
	}
}

type Model struct {
	ID          uint
	UpdatedTime uint
	CreatedTime uint
}

func (m *Model) AddItem() (*Model, error) {
	result := app.orm.db.Create(m)
	if result.Error != nil {
		return nil, result.Error
	}
	return m, nil
}
