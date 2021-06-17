package environments

type EnvLoader interface {
	Defaults() map[string]string
	Load(env *Env) error
}

type SimpleEnvLoader map[string]string

var _ EnvLoader = SimpleEnvLoader{}

func (b SimpleEnvLoader) Defaults() map[string]string {
	return b
}

func (b SimpleEnvLoader) Load(env *Env) error {
	return nil
}
