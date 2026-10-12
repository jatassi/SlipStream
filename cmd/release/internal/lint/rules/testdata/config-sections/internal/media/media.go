package media

type Settings struct{}

func DefaultSettings() Settings { return Settings{} }

type Config struct{}

func (Config) DefaultEndpoints() Config { return Config{} }

func (c Config) defaults() Config { return c.DefaultEndpoints() }
