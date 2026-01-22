package storage

import (
	"fmt"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type WebdavStorageConfig struct {
	BaseConfig
	URL          string `toml:"url" mapstructure:"url" json:"url"`
	Username     string `toml:"username" mapstructure:"username" json:"username"`
	Password     string `toml:"password" mapstructure:"password" json:"password"`
	BasePath     string `toml:"base_path" mapstructure:"base_path" json:"base_path"`
	ReceiverURL  string `toml:"receiver_url" mapstructure:"receiver_url" json:"receiver_url"`
	ChunkSizeMB  int    `toml:"chunk_size_mb" mapstructure:"chunk_size_mb" json:"chunk_size_mb"`
	ChunkRetries int    `toml:"chunk_retries" mapstructure:"chunk_retries" json:"chunk_retries"`
}

func (w *WebdavStorageConfig) Validate() error {
	if w.BasePath == "" {
		return fmt.Errorf("base_path is required for webdav storage")
	}
	if w.ReceiverURL == "" {
		if w.URL == "" {
			return fmt.Errorf("url is required for webdav storage")
		}
		if w.Username == "" || w.Password == "" {
			return fmt.Errorf("username and password is required for webdav storage")
		}
	}
	return nil
}

func (w *WebdavStorageConfig) GetType() storenum.StorageType {
	return storenum.Webdav
}

func (w *WebdavStorageConfig) GetName() string {
	return w.Name
}
